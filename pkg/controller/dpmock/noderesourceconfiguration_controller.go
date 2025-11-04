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

package dpmock

import (
	"context"
	"errors"

	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	dpmockv1alpha1 "volcano.sh/deviceplugin-mock/api/dpmock/v1alpha1"
	"volcano.sh/deviceplugin-mock/pkg/util"
)

// NodeResourceConfigurationReconciler reconciles a NodeResourceConfiguration object
type NodeResourceConfigurationReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	resolvers []ResourceResolver
}

func NewNodeResourceConfigurationReconciler(client client.Client, scheme *runtime.Scheme) *NodeResourceConfigurationReconciler {
	return &NodeResourceConfigurationReconciler{
		Client:    client,
		Scheme:    scheme,
		resolvers: []ResourceResolver{&nameResolver{}, &nodeResourceResolver{cli: client}},
	}
}

// +kubebuilder:rbac:groups=dpmock.volcano.sh,resources=noderesourceconfigurations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=dpmock.volcano.sh,resources=noderesourceconfigurations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=dpmock.volcano.sh,resources=noderesourceconfigurations/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the NodeResourceConfiguration object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.22.1/pkg/reconcile
func (r *NodeResourceConfigurationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	nrcfg := &dpmockv1alpha1.NodeResourceConfiguration{}
	err := r.Get(ctx, req.NamespacedName, nrcfg)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if nrcfg.DeletionTimestamp != nil {
		return ctrl.Result{}, nil
	}

	klog.V(3).InfoS("NodeResourceConfiguration reconciling", "name", nrcfg.Name)
	defer klog.V(3).InfoS("NodeResourceConfiguration reconciled", "name", nrcfg.Name)

	var resDescList []dpmockv1alpha1.ResourceDescription
	for resIdx := range nrcfg.Spec.Resources {
		var resDesc *dpmockv1alpha1.ResourceDescription
		resDesc, err = r.resolve(ctx, nrcfg, resIdx)
		if err != nil {
			if errors.Is(err, ErrUnsupported) {
				klog.ErrorS(err, "skip unsupported item", "name", nrcfg.Name, "resIdx", resIdx)
				continue
			} else {
				klog.ErrorS(err, "failed to resolve resource", "name", nrcfg.Name, "resIdx", resIdx)
				return ctrl.Result{}, err
			}
		}
		resDescList = append(resDescList, *resDesc)
	}

	if !equality.Semantic.DeepEqual(nrcfg.Status.ResourceDescriptions, resDescList) {
		nrcfg.Status.ResourceDescriptions = resDescList
		if err = r.Status().Update(ctx, nrcfg); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

func (r *NodeResourceConfigurationReconciler) resolve(ctx context.Context, nrcfg *dpmockv1alpha1.NodeResourceConfiguration, resIdx int) (*dpmockv1alpha1.ResourceDescription, error) {
	for _, resolver := range r.resolvers {
		resDesc, err := resolver.Resolve(ctx, nrcfg, resIdx)
		if err != nil {
			if errors.Is(err, ErrUnsupported) {
				continue
			}
			return nil, err
		}
		klog.V(3).InfoS("NodeResourceConfiguration item resolved", "name", nrcfg.Name, "resIdx", resIdx, "resolver", resolver.Name())
		return resDesc, nil
	}
	return nil, ErrUnsupported
}

func (r *NodeResourceConfigurationReconciler) getNodeResourceEventHandler() handler.Funcs {
	return handler.Funcs{
		CreateFunc: func(ctx context.Context, evt event.TypedCreateEvent[client.Object], q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
			nrName := evt.Object.GetName()
			r.nodeResourceEventHandler(ctx, nrName, q)
		},
		UpdateFunc: func(ctx context.Context, evt event.TypedUpdateEvent[client.Object], q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
			nrName := evt.ObjectNew.GetName()
			r.nodeResourceEventHandler(ctx, nrName, q)
		},
		DeleteFunc: func(ctx context.Context, evt event.TypedDeleteEvent[client.Object], q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
			nrName := evt.Object.GetName()
			r.nodeResourceEventHandler(ctx, nrName, q)
		},
		GenericFunc: nil,
	}
}

func (r *NodeResourceConfigurationReconciler) nodeResourceEventHandler(ctx context.Context, nrName string, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	var nrcfgList dpmockv1alpha1.NodeResourceConfigurationList
	if err := r.List(ctx, &nrcfgList); err != nil {
		klog.ErrorS(err, "failed to list NodeResourceConfiguration", "name", nrName)
		return
	}
	for _, nrcfg := range nrcfgList.Items {
		for _, res := range nrcfg.Spec.Resources {
			if !util.IsNodeResourceReference(res.ResourceRef) {
				continue
			}
			if res.ResourceRef.Name != nrName {
				continue
			}
			q.Add(ctrl.Request{NamespacedName: types.NamespacedName{Name: nrcfg.Name}})
			break
		}
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *NodeResourceConfigurationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dpmockv1alpha1.NodeResourceConfiguration{}).
		Named("dpmock-noderesourceconfiguration").
		Complete(r)
}
