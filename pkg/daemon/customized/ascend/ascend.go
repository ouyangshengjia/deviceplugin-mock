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

package ascend

import (
	"context"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"

	"volcano.sh/deviceplugin-mock/pkg/daemon/framework"
	"volcano.sh/deviceplugin-mock/pkg/daemon/podmonitor"
	"volcano.sh/deviceplugin-mock/pkg/util"
)

type AscendService struct {
	kubeClient       kubernetes.Interface
	podModifyMethods map[string]func(pod *v1.Pod, pr *podmonitor.PodResource) error
}

func init() {
	ascend := &AscendService{
		podModifyMethods: map[string]func(pod *v1.Pod, pr *podmonitor.PodResource) error{
			// todo
		},
	}
	framework.RegisterService(ascend)
}

func (a *AscendService) Name() string {
	return "daemon-ascend"
}

func (a *AscendService) Initialize() error {
	a.kubeClient = framework.GetClientSet().KubeClient

	return nil
}

func (a *AscendService) Run(ctx context.Context) error {
	wait.UntilWithContext(ctx, a.podDeviceInfoHandler, 2*time.Second)

	return framework.ErrContextDone
}

func (a *AscendService) podDeviceInfoHandler(ctx context.Context) {
	obj, ok := framework.GetStorage().Get(podmonitor.PodListKey)
	if !ok {
		klog.V(4).Info("pod list not found")
		return
	}
	podSnapShots, ok := obj.(map[string]*v1.Pod)
	if !ok {
		klog.Errorf("pod list type error, got type %T", obj)
		return
	}

	obj, ok = framework.GetStorage().Get(podmonitor.PodResourcesKey)
	if !ok {
		klog.V(4).Info("pod resources not found")
		return
	}
	prs, ok := obj.(map[string]*podmonitor.PodResource)
	if !ok {
		klog.Errorf("pod resources type error, got type %T", obj)
		return
	}

	for _, podSnapshot := range podSnapShots {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if isPodTerminated(podSnapshot) {
			continue
		}

		podName := util.GetNamespacedName(podSnapshot.Namespace, podSnapshot.Name)

		pr, exist := prs[podName]
		if !exist {
			klog.ErrorS(nil, "pod resource not found in storage", "podName", podName)
			continue
		}

		// Use the pods cache obtained from kubelet to determine whether a pod update is needed, to reduce the frequency of accessing the api-server
		podSnapshotModified := a.modifyPod(podSnapshot, pr)
		if equality.Semantic.DeepEqual(podSnapshotModified, podSnapshot) {
			klog.V(5).InfoS("pod does not need to be updated", "podName", podName)
			continue
		}

		// Fetch the latest pod info from the api-server if the pod is need to be updated.
		// The pod info obtained from kubelet will not update the ResourceVersion field, so it can not be used to do an update request.
		pod, err := a.kubeClient.CoreV1().Pods(podSnapshot.Namespace).Get(ctx, podSnapshot.Name, metav1.GetOptions{})
		if err != nil {
			klog.ErrorS(err, "failed to get pod from api-server", "podName", podName)
			continue
		}

		// Exclude cases where pod have the same name but different UID, such as in pod recreation scenarios.
		if pod.UID != podSnapshot.UID {
			klog.Warningf("pod '%s' UID obtainedd form api-server(%v) snfr kubelet(%v) is inconsistent", pod.Name, pod.UID, podSnapshot.UID)
			continue
		}

		if isPodTerminated(pod) {
			continue
		}

		podModified := a.modifyPod(pod, pr)
		if equality.Semantic.DeepEqual(podModified, pod) {
			klog.V(5).InfoS("pod does not need to be updated", "podName", podName)
			continue
		}

		_, err = a.kubeClient.CoreV1().Pods(podSnapshot.Namespace).Update(ctx, podModified, metav1.UpdateOptions{})
		if err != nil {
			klog.ErrorS(err, "failed to update pod", "podName", podName)
			continue
		}

		klog.V(3).InfoS("pod updated", "podName", podName)
	}
}

func (a *AscendService) modifyPod(pod *v1.Pod, pr *podmonitor.PodResource) *v1.Pod {
	newPod := pod.DeepCopy()
	for name, f := range a.podModifyMethods {
		err := f(newPod, pr)
		if err != nil {
			klog.ErrorS(err, "failed to modify pod", "method", name, "pod", util.GetNamespacedName(pod.Namespace, pod.Name))
			continue
		}
	}
	return newPod
}

func isPodTerminated(pod *v1.Pod) bool {
	return pod.DeletionTimestamp != nil || pod.Status.Phase == v1.PodSucceeded || pod.Status.Phase == v1.PodFailed
}
