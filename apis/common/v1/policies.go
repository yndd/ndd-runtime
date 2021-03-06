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

type Lifecycle struct {
	// Active specifies if the managed resource is active or plannned
	// +kubebuilder:validation:Enum=`active`;`planned`
	// +kubebuilder:default=active
	DeploymentPolicy DeploymentPolicy `json:"deploymentPolicy,omitempty"`

	// DeletionPolicy specifies what will happen to the underlying external
	// when this managed resource is deleted - either "delete" or "orphan" the
	// external resource.
	// +kubebuilder:validation:Enum=`delete`;`orphan`
	// +kubebuilder:default=delete
	DeletionPolicy DeletionPolicy `json:"deletionPolicy,omitempty"`
}

// A DeploymentPolicy determines what should happen to the underlying external
// resource when a managed resource is deployed.
type DeploymentPolicy string

const (
	// DeploymentActive means the external resource will deployed
	DeploymentActive DeploymentPolicy = "active"

	// DeploymentPlanned means the resource identifier will be allocated but not deployed
	DeploymentPlanned DeploymentPolicy = "planned"
)

// A DeletionPolicy determines what should happen to the underlying external
// resource when a managed resource is deleted.
type DeletionPolicy string

const (
	// DeletionOrphan means the external resource will orphaned when its managed
	// resource is deleted.
	DeletionOrphan DeletionPolicy = "orphan"

	// DeletionDelete means both the  external resource will be deleted when its
	// managed resource is deleted.
	DeletionDelete DeletionPolicy = "delete"
)
