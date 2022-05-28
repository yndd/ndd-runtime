/*
Copyright 2021 NDD.

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

package v1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

// LabelKeyTargetName is added to TargetUsages to relate them to their
// Target.
const LabelKeyTargetName = "ndd.yndd.io/target"

// A Reference to a named object.
type Reference struct {
	// Name of the referenced object.
	Name string `json:"name"`
}

// A TypedReference refers to an object by Name, Kind, and APIVersion. It is
// commonly used to reference cluster-scoped objects or objects where the
// namespace is already known.
type TypedReference struct {
	// APIVersion of the referenced object.
	APIVersion string `json:"apiVersion"`

	// Kind of the referenced object.
	Kind string `json:"kind"`

	// Name of the referenced object.
	Name string `json:"name"`

	// UID of the referenced object.
	// +optional
	UID types.UID `json:"uid,omitempty"`
}

// A Selector selects an object.
type Selector struct {
	// MatchLabels ensures an object with matching labels is selected.
	MatchLabels map[string]string `json:"matchLabels,omitempty"`

	// MatchControllerRef ensures an object with the same controller reference
	// as the selecting object is selected.
	MatchControllerRef *bool `json:"matchControllerRef,omitempty"`
}

// SetGroupVersionKind sets the Kind and APIVersion of a TypedReference.
func (obj *TypedReference) SetGroupVersionKind(gvk schema.GroupVersionKind) {
	obj.APIVersion, obj.Kind = gvk.ToAPIVersionAndKind()
}

// GroupVersionKind gets the GroupVersionKind of a TypedReference.
func (obj *TypedReference) GroupVersionKind() schema.GroupVersionKind {
	return schema.FromAPIVersionAndKind(obj.APIVersion, obj.Kind)
}

// GetObjectKind get the ObjectKind of a TypedReference.
func (obj *TypedReference) GetObjectKind() schema.ObjectKind { return obj }

// A ResourceSpec defines the desired state of a managed resource.
type ResourceSpec struct {
	// Lifecycle determines the deletion and deployment lifecycle policies the resource
	// will follow
	Lifecycle Lifecycle `json:"lifecycle,omitempty"`

	// TargetReference specifies which target will be used to
	// perform crud operations for the managed resource
	TargetReference *Reference `json:"targetRef,omitempty"`
}

// ResourceStatus represents the observed state of a managed resource.
type ResourceStatus struct {
	// the condition status
	ConditionedStatus `json:",inline"`
	// the health condition status
	Health HealthConditionedStatus `json:"health,omitempty"`
	// the oda info
	OdaInfo `json:",inline"`
	// rootPaths define the rootPaths of the cr, used to monitor the resource status
	RootPaths []string `json:"rootPaths,omitempty"`
}

// A TargetStatus defines the observed status of a target.
type TargetStatus struct {
	ConditionedStatus `json:",inline"`

	// Users of this target configuration.
	Users int64 `json:"users,omitempty"`
}

type TransactionResourceStatus struct {
	// the condition status
	ConditionedStatus `json:",inline"`

	// conditionsStatus per device
	Device map[string]ConditionedStatus `json:"device,omitempty"`
}

/*
type TransactionDeviceStatus struct {
	//DeviceName *string `json:"deviceName,omitempty"`
	// the condition status
	ConditionedStatus `json:",inline"`
}
*/

// A TargetUsage is a record that a particular managed resource is using
// a particular target configuration.
type TargetUsage struct {
	// TargetReference to the target being used.
	TargetReference Reference `json:"TargetRef"`

	// ResourceReference to the managed resource using the target config.
	ResourceReference TypedReference `json:"resourceRef"`
}

type ResourceName struct {
	Name string `json:"name,omitempty"`
}
