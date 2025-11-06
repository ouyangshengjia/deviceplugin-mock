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

package deviceplugin

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path"
	"sync/atomic"

	"google.golang.org/grpc"
	"k8s.io/klog/v2"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"

	"volcano.sh/deviceplugin-mock/pkg/daemon/framework"
)

type Server interface {
	Serve(ctx context.Context) error
	UpdateDeviceIDs(deviceIDs []string)
}

type server struct {
	pluginapi.UnimplementedDevicePluginServer

	resourceName   string
	endpoint       string
	deviceIDs      ListMonitor[string]
	kubeletConnEvt chan any
	allocateID     atomic.Uint64
}

func NewDpServer(resourceName string, endpoint string) Server {
	return &server{
		resourceName: resourceName,
		endpoint:     endpoint,
		deviceIDs:    NewListMonitor[string](),
	}
}

func (s *server) Serve(ctx context.Context) error {
	s.kubeletConnEvt = make(chan any, 1)

	if _, err := os.Stat(s.endpoint); err == nil {
		if err = os.Remove(s.endpoint); err != nil {
			return fmt.Errorf("failed to remove endpoint file %s: %w", s.endpoint, err)
		}
	}

	if _, err := os.Stat(s.endpoint); os.IsNotExist(err) {
		dir := path.Dir(s.endpoint)
		if err = os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s for endpoint: %w", dir, err)
		}
	}

	sock, err := net.Listen("unix", s.endpoint)
	if err != nil {
		return fmt.Errorf("failed to listen unix socket %s: %w", s.endpoint, err)
	}
	defer sock.Close()

	grpcServer := grpc.NewServer()
	defer grpcServer.GracefulStop()

	s.registerDevicePluginServer(grpcServer)

	var errCh chan error
	go func() {
		err = grpcServer.Serve(sock)
		if err != nil {
			errCh <- err
			close(errCh)
		}
	}()

	if err = s.registerToKubelet(ctx); err != nil {
		return fmt.Errorf("failed to register to kubelet: %w", err)
	}
	defer func() {
		if e := s.deregister(framework.ContextOnExit()); e != nil {
			klog.ErrorS(err, "failed to deregister from kubelet", "resource", s.resourceName)
		}
	}()

	klog.V(3).InfoS("resource register to kubelet success", "resource", s.resourceName)

	for {
		select {
		case e := <-errCh:
			return fmt.Errorf("grpc server terminated with error: %w", e)
		case <-s.kubeletConnEvt:
			return errors.New("kubelet disconnected")
		case <-ctx.Done():
			return nil
		}
	}
}

func (s *server) UpdateDeviceIDs(deviceIDs []string) {
	s.deviceIDs.Update(deviceIDs)
}
