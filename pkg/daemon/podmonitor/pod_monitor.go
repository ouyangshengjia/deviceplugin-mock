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

package podmonitor

import (
	"context"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	podresourcesv1 "k8s.io/kubelet/pkg/apis/podresources/v1"
	"k8s.io/kubernetes/pkg/kubelet/apis/podresources"

	"volcano.sh/deviceplugin-mock/pkg/daemon/framework"
	"volcano.sh/deviceplugin-mock/pkg/util"
)

const (
	socketPath                 = "/var/lib/kubelet/pod-resources/kubelet.sock"
	defaultPodResourcesMaxSize = 16 * 1024 * 1024
	connectionTimeout          = 5 * time.Second

	PodListKey      = "pod-monitor/pod-list"
	PodResourcesKey = "pod-monitor/pod-resources"
)

var monitor *Monitor

type Monitor struct {
	podResourceClient podresourcesv1.PodResourcesListerClient
	kubeletClient     framework.KubeletInterface
}

type PodResource struct {
	types.NamespacedName
	Resources map[string][]string // [ResourceName][]DeviceID
}

func init() {
	monitor = &Monitor{}
	framework.RegisterService(monitor, framework.WithOfficial())
}

func (m *Monitor) Name() string {
	return "daemon-pod-monitor"
}

func (m *Monitor) Initialize() error {
	client, _, err := podresources.GetV1Client("unix://"+socketPath, connectionTimeout, defaultPodResourcesMaxSize)
	if err != nil {
		return err
	}
	m.podResourceClient = client

	m.kubeletClient = framework.GetClientSet().KubeletClient

	return nil
}

func (m *Monitor) Run(ctx context.Context) error {
	wait.UntilWithContext(ctx, m.updateStorage, time.Second)

	return framework.ErrContextDone
}

func (m *Monitor) updateStorage(ctx context.Context) {
	m.fetchPods(ctx)
	m.fetchPodResources(ctx)
}

func (m *Monitor) fetchPods(ctx context.Context) {
	podList, err := m.kubeletClient.ListAllPods(ctx)
	if err != nil {
		klog.ErrorS(err, "failed to list pods")
		return
	}

	pods := make(map[string]*v1.Pod)
	for _, pod := range podList.Items {
		pods[util.GetNamespacedName(pod.Namespace, pod.Name)] = pod.DeepCopy()
	}

	framework.GetStorage().Set(PodListKey, pods)
}

func (m *Monitor) fetchPodResources(ctx context.Context) {
	resp, err := m.podResourceClient.List(ctx, &podresourcesv1.ListPodResourcesRequest{})
	if err != nil {
		klog.ErrorS(err, "failed to list pod resources")
		return
	}
	if resp == nil {
		klog.Error("failed to list pod resources: empty response")
		return
	}

	podResources := m.parseResponse(resp)

	framework.GetStorage().Set(PodResourcesKey, podResources)
}

func (m *Monitor) parseResponse(resp *podresourcesv1.ListPodResourcesResponse) map[string]*PodResource {
	podResources := make(map[string]*PodResource)
	for _, pr := range resp.PodResources {
		if pr == nil {
			continue
		}

		resources := make(map[string][]string)
		for _, container := range pr.Containers {
			for _, device := range container.Devices {
				resources[device.ResourceName] = append(resources[device.ResourceName], device.DeviceIds...)
			}
		}

		podResource := PodResource{
			NamespacedName: types.NamespacedName{
				Namespace: pr.Namespace,
				Name:      pr.Name,
			},
			Resources: resources,
		}
		podResources[podResource.NamespacedName.String()] = &podResource
	}
	return podResources
}
