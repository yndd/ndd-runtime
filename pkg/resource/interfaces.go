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

package resource

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nddv1 "github.com/yndd/ndd-runtime/apis/common/v1"
)

// A Conditioned may have conditions set or retrieved. Conditions are typically
// indicate the status of both a resource and its reconciliation process.
type Conditioned interface {
	SetConditions(c ...nddv1.Condition)
	GetCondition(ck nddv1.ConditionKind) nddv1.Condition
}

type HealthConditioned interface {
	SetHealthConditions(c nddv1.HealthConditionedStatus)
}

type RootPaths interface {
	SetRootPaths(rootPaths []string)
	GetRootPaths() []string
}

// A TargetReferencer may reference a target resource.
type TargetReferencer interface {
	GetTargetReference() *nddv1.Reference
	SetTargetReference(p *nddv1.Reference)
}

// A RequiredTargetReferencer may reference a target config resource.
// Unlike TargetReferencer, the reference is required (i.e. not nil).
type RequiredTargetReferencer interface {
	GetTargetReference() nddv1.Reference
	SetTargetReference(p nddv1.Reference)
}

// A RequiredTypedResourceReferencer can reference a resource.
type RequiredTypedResourceReferencer interface {
	SetResourceReference(r nddv1.TypedReference)
	GetResourceReference() nddv1.TypedReference
}

// An Lifecycle resource may specify a DeletionPolicy and/or DeploymentPolicy.
type Lifecycle interface {
	GetDeletionPolicy() nddv1.DeletionPolicy
	SetDeletionPolicy(p nddv1.DeletionPolicy)
	GetDeploymentPolicy() nddv1.DeploymentPolicy
	SetDeploymentPolicy(p nddv1.DeploymentPolicy)
}

// A UserCounter can count how many users it has.
type UserCounter interface {
	SetUsers(i int64)
	GetUsers() int64
}

// An Object is a Kubernetes object.
type Object interface {
	metav1.Object
	runtime.Object
}

// A Managed is a Kubernetes object representing a concrete managed
// resource (e.g. a CloudSQL instance).
type Managed interface {
	Object

	TargetReferencer
	Lifecycle

	Conditioned
	HealthConditioned
	RootPaths
}

// A ManagedList is a list of managed resources.
type ManagedList interface {
	client.ObjectList

	// GetItems returns the list of managed resources.
	GetItems() []Managed
}

// A Target configures a Target Device Driver.
type Target interface {
	Object

	UserCounter
	Conditioned
}

// A TargetUsage indicates a usage of a target config.
type TargetUsage interface {
	Object

	RequiredTargetReferencer
	RequiredTypedResourceReferencer
}

// A TargetUsageList is a list of targets usages.
type TargetUsageList interface {
	client.ObjectList

	// GetItems returns the list of target usages.
	GetItems() []TargetUsage
}

// A Finalizer manages the finalizers on the resource.
type Finalizer interface {
	AddFinalizer(ctx context.Context, obj Object) error
	RemoveFinalizer(ctx context.Context, obj Object) error
	HasOtherFinalizer(ctx context.Context, obj Object) (bool, error)
	AddFinalizerString(ctx context.Context, obj Object, finalizerString string) error
	RemoveFinalizerString(ctx context.Context, obj Object, finalizerString string) error
}
