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

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ResourceInfo describes the resource information to be mocked to the selected nodes.
// One and only one of the fields must be specified.
type ResourceInfo struct {
	// ResourceName indicates the name of the resource.
	// +optional
	ResourceName string `json:"resourceName,omitempty" protobuf:"bytes,1,opt,name=resourceName"`

	// ResourceRef describes the resource information by referencing other objects.
	// This field can only reference a NodeResource currently.
	// +optional
	ResourceRef *ResourceReference `json:"resourceRef,omitempty" protobuf:"bytes,2,opt,name=resourceRef"`
}

// ResourceRequirement describes the details of the resource requirement.
type ResourceRequirement struct {
	// ResourceInfo describes the information of the resource that is expected to be mocked.
	ResourceInfo `json:",inline"`

	// Capacity indicates the resource capacity in a node.
	//
	// In the case that ResourceInfo reference a ResourceReference, the default capacity can be defined by the reference.
	// +optional
	Capacity *resource.Quantity `json:"capacity,omitempty" protobuf:"bytes,1,opt,name=capacity"`

	// NodePatchTemplate defines a node patch body based on Go Template rendering.
	// The context of the Go Template is set to the ResourceBasicDescription corresponding to current resource in status field.
	// For example, you can get the resolved result of the deviceID format by using ".deviceIDFormat".
	//
	// In the case that ResourceInfo reference a ResourceReference, the default NodePatchTemplate can be defined by the reference.
	// +optional
	NodePatchTemplate string `json:"nodePatchTemplate,omitempty" protobuf:"bytes,2,opt,name=nodePatchTemplate"`

	// NodeUndoPatch defines a node patch body to be executed when the resource is removed in the node.
	//
	// In the case that ResourceInfo reference a ResourceReference, the default NodeUndoPatch can be defined by the reference.
	// +optional
	NodeUndoPatch string `json:"nodeUndoPatch,omitempty" protobuf:"bytes,3,opt,name=nodeUndoPatch"`
}

// NodeResourceConfigurationSpec defines the desired state of NodeResourceConfiguration
type NodeResourceConfigurationSpec struct {
	// NodeSelector selects a set of nodes by using a label selector.
	// The resource configuration will only be applied to the nodes whose labels matches this selector.
	// +optional
	NodeSelector *metav1.LabelSelector `json:"nodeSelector,omitempty" protobuf:"bytes,1,opt,name=nodeSelector"`

	// Resources describe the mock requirement of custom resource.
	Resources []ResourceRequirement `json:"resources,omitempty" protobuf:"bytes,2,rep,name=resources"`
}

// DeviceIDFormat describe the format of the deviceID serial.
// For example, given:
//
// Prefix:       {"dev0": 2, "dev1": 2}
// Delimiter:    "-"
// OrdinalStart: 0
//
// Then the deviceID list should be {"dev0-0", "dev0-1", "dev1-0", "dev1-1"}
type DeviceIDFormat struct {
	// The map key means the prefix of the deviceID and the map value means the amount of IDs corresponding to the prefix.
	Prefix map[string]int32 `json:"prefix,omitempty" protobuf:"bytes,1,rep,name=prefix"`

	// Delimiter indicates the concatenation between prefix and ordinal in the deviceID.
	// +optional
	Delimiter string `json:"delimiter,omitempty" protobuf:"bytes,2,opt,name=delimiter"`

	// OrdinalStart indicates the starting number of the ID serial.
	OrdinalStart int32 `json:"ordinalStart,omitempty" protobuf:"varint,3,opt,name=ordinalStart"`
}

type ResourceBasicDescription struct {
	// ResourceName indicates the name of the resource.
	ResourceName string `json:"resourceName,omitempty" protobuf:"bytes,1,opt,name=resourceName"`

	// Capacity indicates the capacity of the resource.
	Capacity resource.Quantity `json:"capacity,omitempty" protobuf:"bytes,2,opt,name=capacity"`

	// DeviceIDFormat represents the ultimate format for generating the deviceID serial.
	DeviceIDFormat DeviceIDFormat `json:"deviceIDFormat,omitempty" protobuf:"bytes,3,opt,name=deviceIDFormat"`
}

type ResourceDescription struct {
	ResourceBasicDescription `json:",inline"`

	// NodePatch holds the Go Template rendering result of ResourceRequirement.NodePatchTemplate.
	// +optional
	NodePatch string `json:"nodePatch,omitempty" protobuf:"bytes,1,opt,name=nodePatch"`

	// NodeUndoPatch holds a node patch body to be executed when the resource is removed in the node.
	// +optional
	NodeUndoPatch string `json:"nodeUndoPatch,omitempty" protobuf:"bytes,2,opt,name=nodeUndoPatch"`
}

// NodeResourceConfigurationStatus defines the observed state of NodeResourceConfiguration.
type NodeResourceConfigurationStatus struct {
	// ResourceDescriptions hold the ultimate resolved results of the resource descriptions that defined in NodeResourceConfigurationSpec.
	ResourceDescriptions []ResourceDescription `json:"resourceDescriptions,omitempty" protobuf:"bytes,1,rep,name=resourceDescriptions"`

	// NumberDesired indicates the number of nodes that should apply the configuration.
	NumberDesired int32 `json:"numberDesired,omitempty" protobuf:"bytes,2,opt,name=numberDesired"`

	// NumberAvailable indicates the number of nodes that should apply the configuration and
	// have one daemon pod running and available.
	NumberAvailable int32 `json:"numberAvailable,omitempty" protobuf:"bytes,3,opt,name=numberAvailable"`

	// NumberSynchronized indicates the number of nodes that has synchronized the configuration.
	NumberSynchronized int32 `json:"numberSynchronized,omitempty" protobuf:"bytes,4,opt,name=numberSynchronized"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,shortName={"nrcfg"}

// NodeResourceConfiguration is the Schema for the noderesourceconfigurations API
type NodeResourceConfiguration struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty,omitzero"`

	// spec defines the desired state of NodeResourceConfiguration
	// +required
	Spec NodeResourceConfigurationSpec `json:"spec"`

	// status defines the observed state of NodeResourceConfiguration
	// +optional
	Status NodeResourceConfigurationStatus `json:"status,omitempty,omitzero"`
}

// +kubebuilder:object:root=true

// NodeResourceConfigurationList contains a list of NodeResourceConfiguration
type NodeResourceConfigurationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NodeResourceConfiguration `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NodeResourceConfiguration{}, &NodeResourceConfigurationList{})
}
