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
	"os"
	"path/filepath"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/informers"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"

	"volcano.sh/deviceplugin-mock/client-go/clientset/versioned"
	"volcano.sh/deviceplugin-mock/client-go/informers/externalversions"
)

var (
	clientSet ClientSet
)

type KubeClientInitFunction func(ClientSet) error

type ClientSet struct {
	KubeClient          kubernetes.Interface
	KubeInformerFactory informers.SharedInformerFactory

	DpmockClient          versioned.Interface
	DpmockInformerFactory externalversions.SharedInformerFactory
}

func GetClientSet() ClientSet {
	return clientSet
}

func initializeKubeClientSet(ctx context.Context, args *Args, initFuncList ...KubeClientInitFunction) error {
	// get kube config
	config, err := getKubeConfig()
	if err != nil {
		return err
	}
	config.QPS = float32(args.KubeClientQPS)
	config.Burst = args.KubeClientBurst

	// init kube client
	clientSet.KubeClient, err = kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}
	kubeInformerFactory := informers.NewSharedInformerFactory(clientSet.KubeClient, 60*time.Second)
	kubeInformerFactory.InformerFor(&v1.Node{},
		func(client kubernetes.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
			tweakListOptions := func(options *metav1.ListOptions) {
				options.FieldSelector = fields.OneTermEqualSelector(metav1.ObjectNameField, GetEnvs().NodeName).String()
			}
			return coreinformers.NewFilteredNodeInformer(client, resyncPeriod, cache.Indexers{}, tweakListOptions)
		})
	kubeInformerFactory.InformerFor(&v1.Pod{},
		func(client kubernetes.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
			tweakListOptions := func(options *metav1.ListOptions) {
				options.FieldSelector = fields.OneTermEqualSelector("spec.nodeName", GetEnvs().NodeName).String()
			}
			return coreinformers.NewFilteredPodInformer(client, metav1.NamespaceAll, resyncPeriod, cache.Indexers{}, tweakListOptions)
		})
	clientSet.KubeInformerFactory = kubeInformerFactory

	// init dpmock client
	clientSet.DpmockClient, err = versioned.NewForConfig(config)
	if err != nil {
		return err
	}
	clientSet.DpmockInformerFactory = externalversions.NewSharedInformerFactory(clientSet.DpmockClient, 60*time.Second)

	// invoke init functions
	for _, f := range initFuncList {
		if err = f(clientSet); err != nil {
			return err
		}
	}

	// start and wait cache sync
	clientSet.KubeInformerFactory.Start(ctx.Done())
	clientSet.DpmockInformerFactory.Start(ctx.Done())
	clientSet.KubeInformerFactory.WaitForCacheSync(ctx.Done())
	clientSet.DpmockInformerFactory.WaitForCacheSync(ctx.Done())

	return nil
}

func getKubeConfig() (*rest.Config, error) {
	config, err := rest.InClusterConfig()
	if err == nil {
		return config, nil
	}

	kubeConfig := os.Getenv("KUBECONFIG")
	if kubeConfig == "" {
		kubeConfig = filepath.Join(os.Getenv("HOME"), ".kube", "config")
	}
	return clientcmd.BuildConfigFromFlags("", kubeConfig)
}
