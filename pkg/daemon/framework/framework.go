/*
Copyright 2025 The Volcano Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package framework

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"slices"
	"sync"
	"syscall"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
)

type Service interface {
	Name() string
	Initialize() error
	Run(ctx context.Context) error
}

var (
	Context        = context.Background()
	ErrContextDone = errors.New("context done")
	services       = make(map[string]serviceWithOptions)
	lock           sync.Mutex
)

type serviceOptions struct {
	official           bool
	kubeClientInitFunc KubeClientInitFunction
}

type serviceWithOptions struct {
	Service
	opts serviceOptions
}

type ServiceOption func(options *serviceOptions)

func WithOfficial() ServiceOption {
	return func(options *serviceOptions) {
		options.official = true
	}
}

func WithKubeClientInitFunc(f KubeClientInitFunction) ServiceOption {
	return func(options *serviceOptions) {
		options.kubeClientInitFunc = f
	}
}

func RegisterService(svc Service, opts ...ServiceOption) {
	if svc == nil {
		return
	}

	service := serviceWithOptions{Service: svc, opts: serviceOptions{}}
	for _, opt := range opts {
		opt(&service.opts)
	}

	lock.Lock()
	defer lock.Unlock()
	services[svc.Name()] = service
}

func Initialize(args *Args) error {
	if err := initializeEnvironment(); err != nil {
		return fmt.Errorf("environment initialization failed: %w", err)
	}

	initializeStorage()

	// ignore unselected services
	customizedServices := args.ParseCustomizedServiceList()
	for name, service := range services {
		if service.opts.official {
			continue
		}
		if slices.Contains(customizedServices, name) {
			continue
		}
		klog.V(3).InfoS("ignored customized service", "service", name)
		delete(services, name)
	}

	// init kube client set
	var kubeClientInitFuncList []KubeClientInitFunction
	for _, service := range services {
		if service.opts.kubeClientInitFunc != nil {
			kubeClientInitFuncList = append(kubeClientInitFuncList, service.opts.kubeClientInitFunc)
		}
	}
	if err := initializeKubeClientSet(Context, args, kubeClientInitFuncList...); err != nil {
		return fmt.Errorf("kubernetes client set initialization failed: %w", err)
	}
	klog.V(3).Info("kubernetes client set initialized")

	// init services
	for _, svc := range services {
		if err := svc.Initialize(); err != nil {
			return fmt.Errorf("service '%s' initialization failed: %w", svc.Name(), err)
		}
		klog.V(3).InfoS("service initialized", "service", svc.Name())
	}

	return nil
}

func Run() {
	ctx, cancel := context.WithCancel(Context)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM, syscall.SIGQUIT)

	for _, s := range services {
		go func(svc Service) {
			wait.UntilWithContext(ctx, func(ctx context.Context) {
				klog.V(3).InfoS("service starting", "service", svc.Name())
				err := svc.Run(ctx)
				klog.ErrorS(err, "service stopped", "service", svc.Name())
			}, 2*time.Second)
		}(s)
	}

	sig := <-sigChan
	klog.V(3).InfoS("framework is terminated by signal", "signal", sig)

	cancel()
	time.Sleep(2 * time.Second)
}

func ContextOnExit() context.Context {
	return context.Background() // todo return context with timeout
}
