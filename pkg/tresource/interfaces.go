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

package tresource

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

type DeviceConditioned interface {
	SetDeviceConditions(d string, c ...nddv1.Condition)
	GetDeviceCondition(d string, ck nddv1.ConditionKind) nddv1.Condition
}

type Generation interface {
	GetOwnerGeneration() string
}

// An Object is a Kubernetes object.
type Object interface {
	metav1.Object
	runtime.Object
}

// A Managed is a Kubernetes object representing a concrete managed
// resource (e.g. a CloudSQL instance).
type Transaction interface {
	Object

	Conditioned
	DeviceConditioned
	Generation
}

// A ManagedList is a list of managed resources.
type TransactionList interface {
	client.ObjectList

	// GetItems returns the list of managed resources.
	GetItems() []Transaction
}

// A Finalizer manages the finalizers on the resource.
type Finalizer interface {
	AddFinalizer(ctx context.Context, obj Object) error
	RemoveFinalizer(ctx context.Context, obj Object) error
	HasOtherFinalizer(ctx context.Context, obj Object) (bool, error)
	AddFinalizerString(ctx context.Context, obj Object, finalizerString string) error
	RemoveFinalizerString(ctx context.Context, obj Object, finalizerString string) error
}
