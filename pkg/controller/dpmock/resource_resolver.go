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
	"bytes"
	"context"
	"errors"
	"fmt"
	"html"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	dpmockv1alpha1 "volcano.sh/deviceplugin-mock/api/dpmock/v1alpha1"
	"volcano.sh/deviceplugin-mock/pkg/util"
)

type ResourceResolver interface {
	Name() string
	Resolve(ctx context.Context, nrcfg *dpmockv1alpha1.NodeResourceConfiguration, resIdx int) (desc *dpmockv1alpha1.ResourceDescription, err error)
}

type nameResolver struct{}

func (n *nameResolver) Name() string {
	return "name-resolver"
}

func (n *nameResolver) Resolve(_ context.Context, nrcfg *dpmockv1alpha1.NodeResourceConfiguration, resIdx int) (*dpmockv1alpha1.ResourceDescription, error) {
	if resIdx >= len(nrcfg.Spec.Resources) {
		return nil, errors.New("resIdx out of range")
	}

	res := &nrcfg.Spec.Resources[resIdx]
	if res.ResourceName == "" {
		return nil, ErrUnsupported
	}

	var capacity resource.Quantity
	if res.Capacity != nil {
		capacity = *res.Capacity
	}

	capInt, ok := res.Capacity.AsInt64()
	if !ok {
		return nil, errors.New("parse capacity to int64 failed")
	}

	basicDesc := &dpmockv1alpha1.ResourceBasicDescription{
		ResourceName: res.ResourceName,
		Capacity:     capacity,
		DeviceIDFormat: dpmockv1alpha1.DeviceIDFormat{
			Prefix: map[string]int32{DefaultDevicePrefix: int32(capInt)},
		},
	}

	patch, err := parseNodePatch(res.NodePatchTemplate, basicDesc)
	if err != nil {
		return nil, fmt.Errorf("parse node patch template failed: %w", err)
	}

	var undoPatch []byte
	if res.NodeUndoPatch != "" {
		undoPatch, err = yaml.YAMLToJSON([]byte(res.NodeUndoPatch))
		if err != nil {
			return nil, fmt.Errorf("parse node undo patch failed: %w", err)
		}
	}

	return &dpmockv1alpha1.ResourceDescription{
		ResourceBasicDescription: *basicDesc,
		NodePatch:                patch,
		NodeUndoPatch:            string(undoPatch),
	}, nil
}

type nodeResourceResolver struct {
	cli client.Client
}

func (n *nodeResourceResolver) Name() string {
	return "node-resource-resolver"
}

func (n *nodeResourceResolver) Resolve(ctx context.Context, nrcfg *dpmockv1alpha1.NodeResourceConfiguration, resIdx int) (*dpmockv1alpha1.ResourceDescription, error) {
	if resIdx >= len(nrcfg.Spec.Resources) {
		return nil, errors.New("resIdx out of range")
	}

	res := &nrcfg.Spec.Resources[resIdx]
	nr, err := n.getResourceRef(ctx, res.ResourceRef)
	if err != nil {
		return nil, err
	}

	prefixes, err := n.getPrefixes(ctx, nr, nrcfg)
	if err != nil {
		return nil, err
	}
	capacity := n.getCapacity(res, nr)
	capInt, ok := capacity.AsInt64()
	if !ok {
		return nil, errors.New("parse capacity to int64 failed")
	}

	prefixMap := map[string]int32{}
	for _, prefix := range prefixes {
		prefixMap[prefix] = int32(capInt)
	}

	capacityTotal := capacity.DeepCopy()
	capacityTotal.Mul(int64(len(prefixes)))

	basicDesc := &dpmockv1alpha1.ResourceBasicDescription{
		ResourceName: nr.Spec.ResourceName,
		Capacity:     capacityTotal,
		DeviceIDFormat: dpmockv1alpha1.DeviceIDFormat{
			Prefix:       prefixMap,
			Delimiter:    nr.Spec.DeviceIDGeneratePolicy.Delimiter,
			OrdinalStart: nr.Spec.DeviceIDGeneratePolicy.OrdinalStart,
		},
	}

	nodePatchTemplate := n.getNodePatchTemplate(res, nr)
	patch, err := parseNodePatch(nodePatchTemplate, basicDesc)
	if err != nil {
		return nil, fmt.Errorf("parse node patch template failed: %w", err)
	}

	var undoPatch []byte
	nodeUndoPatch := n.getNodeUndoPatch(res, nr)
	if nodeUndoPatch != "" {
		undoPatch, err = yaml.YAMLToJSON([]byte(nodeUndoPatch))
		if err != nil {
			return nil, fmt.Errorf("parse node undo patch failed: %w", err)
		}
	}

	return &dpmockv1alpha1.ResourceDescription{
		ResourceBasicDescription: *basicDesc,
		NodePatch:                patch,
		NodeUndoPatch:            string(undoPatch),
	}, nil
}

func (n *nodeResourceResolver) getResourceRef(ctx context.Context, ref *dpmockv1alpha1.ResourceReference) (*dpmockv1alpha1.NodeResource, error) {
	if util.IsNodeResourceReference(ref) {
		return nil, ErrUnsupported
	}

	nr := &dpmockv1alpha1.NodeResource{}
	if err := n.cli.Get(ctx, types.NamespacedName{Name: ref.Name}, nr); err != nil {
		if apierrors.IsNotFound(err) {
			klog.V(5).InfoS("get resource ref fail", "err", err, "name", ref.Name)
			return nil, ErrUnsupported
		}
		return nil, err
	}

	return nr, nil
}

func (n *nodeResourceResolver) getCapacity(res *dpmockv1alpha1.ResourceRequirement, nr *dpmockv1alpha1.NodeResource) resource.Quantity {
	if res.Capacity != nil {
		return *res.Capacity
	} else {
		return nr.Spec.DefaultCapacity
	}
}

func (n *nodeResourceResolver) getPrefixes(ctx context.Context, nr *dpmockv1alpha1.NodeResource, nrcfg *dpmockv1alpha1.NodeResourceConfiguration) ([]string, error) {
	prefix := nr.Spec.DeviceIDGeneratePolicy.Prefix
	if len(prefix.Static) > 0 {
		return prefix.Static, nil
	}

	if prefix.ParentResourceRef == nil {
		return nil, errors.New("empty prefix generate policy")
	}

	nrRef, err := n.getResourceRef(ctx, prefix.ParentResourceRef)
	if err != nil {
		return nil, err
	}

	if nrRef.Spec.DeviceIDGeneratePolicy.Prefix.ParentResourceRef != nil {
		klog.ErrorS(nil, "NodeResource contains unsupported cascade reference", "name", nr.Name)
		return nil, ErrUnsupported
	}

	resNameRef := nrRef.Spec.ResourceName

	for _, resRef := range nrcfg.Status.ResourceDescriptions {
		if resRef.ResourceName == resNameRef {
			return util.BuildDeviceID(&resRef.DeviceIDFormat), nil
		}
	}

	return nil, nil
}

func (n *nodeResourceResolver) getNodeUndoPatch(res *dpmockv1alpha1.ResourceRequirement, nr *dpmockv1alpha1.NodeResource) string {
	if res.NodeUndoPatch != "" {
		return res.NodeUndoPatch
	} else {
		return nr.Spec.DefaultNodeUndoPatch
	}
}

func (n *nodeResourceResolver) getNodePatchTemplate(res *dpmockv1alpha1.ResourceRequirement, nr *dpmockv1alpha1.NodeResource) string {
	if res.NodePatchTemplate != "" {
		return res.NodePatchTemplate
	} else {
		return nr.Spec.DefaultNodePatchTemplate
	}
}

func parseNodePatch(text string, jsonContext any) (string, error) {
	if text == "" {
		return "", nil
	}

	inst, err := util.GoTemplateParse(text)
	if err != nil {
		return "", err
	}

	bb, err := json.Marshal(jsonContext)
	if err != nil {
		return "", err
	}
	var m map[string]interface{}
	if err = json.Unmarshal(bb, &m); err != nil {
		return "", err
	}

	yamlPatch := &bytes.Buffer{}
	err = inst.Execute(yamlPatch, m)
	if err != nil {
		return "", err
	}

	jsonPatch, err := yaml.YAMLToJSON([]byte(html.UnescapeString(yamlPatch.String())))
	if err != nil {
		return "", err
	}

	return string(jsonPatch), nil
}
