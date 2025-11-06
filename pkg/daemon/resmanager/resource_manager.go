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

package resmanager

import (
	"context"
	"fmt"
	"path"
	"sync/atomic"
	"time"

	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
	"sigs.k8s.io/yaml"

	dpmockv1alpha1 "volcano.sh/deviceplugin-mock/api/dpmock/v1alpha1"
	"volcano.sh/deviceplugin-mock/pkg/daemon/framework"
	"volcano.sh/deviceplugin-mock/pkg/daemon/resmanager/deviceplugin"
	"volcano.sh/deviceplugin-mock/pkg/util"
)

const (
	pluginDir          = "volcano"
	endpointNameLength = 8
)

type ResourceManager interface {
	Run(ctx context.Context)
	Update(desc *dpmockv1alpha1.ResourceDescription)
}

func New(resourceName string) ResourceManager {
	manager := resourceManager{
		resourceName: resourceName,
		dpServer:     deviceplugin.NewDpServer(resourceName, path.Join(pluginapi.DevicePluginPath, pluginDir, rand.String(endpointNameLength)+".sock")),
	}
	return &manager
}

type resourceManager struct {
	resourceName string
	dpServer     deviceplugin.Server

	desc         atomic.Pointer[dpmockv1alpha1.ResourceDescription]
	synchronized atomic.Bool
}

func (m *resourceManager) Run(ctx context.Context) {
	klog.V(3).InfoS("resource manager start", "resourceName", m.resourceName)
	defer klog.V(3).InfoS("resource manager end", "resourceName", m.resourceName)

	go wait.UntilWithContext(ctx, func(ctx context.Context) {
		if err := m.dpServer.Serve(ctx); err != nil {
			klog.ErrorS(err, "dp server terminated with error", "resourceName", m.resourceName)
		}
	}, 5*time.Second)

	go wait.UntilWithContext(ctx, func(ctx context.Context) {
		if m.synchronized.Load() {
			return
		}

		m.synchronized.Store(true)
		if err := m.syncConfigOnce(ctx); err != nil {
			klog.ErrorS(err, "sync config once failed", "resourceName", m.resourceName)
			m.synchronized.Store(false)
		}
	}, time.Second)

	<-ctx.Done()

	m.onExit()
}

func (m *resourceManager) Update(desc *dpmockv1alpha1.ResourceDescription) {
	if equality.Semantic.DeepEqual(m.desc.Load(), desc) {
		return
	}

	m.desc.Store(desc.DeepCopy())
	m.synchronized.Store(false)
}

func (m *resourceManager) syncConfigOnce(ctx context.Context) error {
	desc := m.desc.Load()
	if desc == nil {
		return nil
	}

	deviceIDs := util.BuildDeviceID(&desc.DeviceIDFormat)
	m.dpServer.UpdateDeviceIDs(deviceIDs)
	klog.V(3).InfoS("resource update deviceIDs", "resourceName", m.resourceName, "format", desc.DeviceIDFormat)

	if desc.NodePatch != "" {
		if err := patchToNode(ctx, []byte(desc.NodePatch)); err != nil {
			return err
		}
		klog.V(3).InfoS("resource patch to node success", "resourceName", m.resourceName, "patch", desc.NodePatch)
	}

	return nil
}
func (m *resourceManager) onExit() {
	desc := m.desc.Load()
	if desc != nil && desc.NodeUndoPatch != "" {
		if err := patchToNode(framework.ContextOnExit(), []byte(desc.NodeUndoPatch)); err != nil {
			klog.ErrorS(err, "resource undo patch to node failed", "resourceName", m.resourceName, "undoPatch", desc.NodeUndoPatch)
		} else {
			klog.V(3).InfoS("resource undo patch to node success", "resourceName", m.resourceName, "undoPatch", desc.NodeUndoPatch)
		}
	}
}

func patchToNode(ctx context.Context, patch []byte) error {
	jsonPatch, err := yaml.YAMLToJSON(patch)
	if err != nil {
		return err
	}

	_, err = framework.GetClientSet().KubeClient.CoreV1().Nodes().
		Patch(ctx, framework.GetEnvs().NodeName, types.StrategicMergePatchType, jsonPatch, metav1.PatchOptions{})
	if err != nil {
		klog.V(5).InfoS("patch to node fail", "error", err, "patch", string(jsonPatch))
		return fmt.Errorf("patch to node fail: %w", err)
	}
	klog.V(3).InfoS("patch to node success", "patch", string(jsonPatch))

	return nil
}
