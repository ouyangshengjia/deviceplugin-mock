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

type StaticPrefixGeneratePolicy []string

// PrefixGeneratePolicy defines the policies for generating the prefix of the deviceID serial.
// One and only one of the fields must be specified.
type PrefixGeneratePolicy struct {
	// Static indicates that the deviceID serial is prefixed with the static string list.
	// +optional
	Static StaticPrefixGeneratePolicy `json:"static,omitempty" protobuf:"bytes,1,opt,name=static"`

	// ParentResourceRef indicates that the deviceID serial is prefixed with the deviceID list of the parent resource.
	// This field can only reference a NodeResource currently.
	// +optional
	ParentResourceRef *ResourceReference `json:"parentResourceRef,omitempty" protobuf:"bytes,2,opt,name=parentResourceRef"`
}

type DeviceIDGeneratePolicy struct {
	// Prefix defines the prefix of the deviceID serial.
	// +optional
	Prefix *PrefixGeneratePolicy `json:"prefix,omitempty" protobuf:"bytes,1,opt,name=prefix"`

	// Delimiter defines the delimiter of the ID serial.
	// +optional
	Delimiter string `json:"delimiter,omitempty" protobuf:"bytes,2,opt,name=delimiter"`

	// OrdinalStart defines the starting number of the ID serial.
	// +optional
	OrdinalStart int32 `json:"ordinalStart,omitempty" protobuf:"varint,3,opt,name=ordinalStart"`
}

// NodeResourceSpec defines the desired state of NodeResource
type NodeResourceSpec struct {
	// ResourceName defines custom resource name.
	ResourceName string `json:"resourceName,omitempty" protobuf:"bytes,1,opt,name=resourceName"`

	// DeviceIDGeneratePolicy describes how to generate deviceID serial.
	// +optional
	DeviceIDGeneratePolicy *DeviceIDGeneratePolicy `json:"deviceIDGeneratePolicy,omitempty" protobuf:"bytes,2,opt,name=deviceIDGeneratePolicy"`

	// DefaultCapacity defines the default capacity of the resource.
	// +optional
	DefaultCapacity resource.Quantity `json:"defaultCapacity,omitempty" protobuf:"bytes,3,opt,name=defaultCapacity"`

	// DefaultNodePatchTemplate defines a default node patch body based on Go Template rendering.
	// +optional
	DefaultNodePatchTemplate string `json:"defaultNodePatchTemplate,omitempty" protobuf:"bytes,4,opt,name=defaultNodePatchTemplate"`

	// DefaultNodeUndoPatch defines a default node patch body to be executed when the resource is removed in the node.
	// +optional
	DefaultNodeUndoPatch string `json:"defaultNodeUndoPatch,omitempty" protobuf:"bytes,5,opt,name=defaultNodeUndoPatch"`
}

// NodeResourceStatus defines the observed state of NodeResource.
type NodeResourceStatus struct {
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,shortName={"nr"}

// NodeResource is the Schema for the noderesources API
type NodeResource struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty,omitzero"`

	// spec defines the desired state of NodeResource
	// +required
	Spec NodeResourceSpec `json:"spec"`

	// status defines the observed state of NodeResource
	// +optional
	Status NodeResourceStatus `json:"status,omitempty,omitzero"`
}

// +kubebuilder:object:root=true

// NodeResourceList contains a list of NodeResource
type NodeResourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NodeResource `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NodeResource{}, &NodeResourceList{})
}
