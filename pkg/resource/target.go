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

	"github.com/pkg/errors"
	nddv1 "github.com/yndd/ndd-runtime/apis/common/v1"
	"github.com/yndd/ndd-runtime/pkg/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	errMissingTargetRef = "managed resource does not reference a target reference"
	errApplyTargetUsage = "cannot apply TargetUsage"
)

type errMissingRef struct{ error }

func (m errMissingRef) MissingReference() bool { return true }

// A Tracker tracks managed resources.
type Tracker interface {
	// Track the supplied managed resource.
	Track(ctx context.Context, mg Managed) error
}

// A TrackerFn is a function that tracks managed resources.
type TrackerFn func(ctx context.Context, mg Managed) error

// Track the supplied managed resource.
func (fn TrackerFn) Track(ctx context.Context, mg Managed) error {
	return fn(ctx, mg)
}

// A TargetUsageTracker tracks usages of a Target by creating or
// updating the appropriate TargetUsage.
type TargetUsageTracker struct {
	c  Applicator
	of TargetUsage
}

// NewTargetUsageTracker creates a TargetUsageTracker.
func NewTargetUsageTracker(c client.Client, of TargetUsage) *TargetUsageTracker {
	return &TargetUsageTracker{c: NewAPIUpdatingApplicator(c), of: of}
}

// Track that the supplied Managed resource is using the Target it
// references by creating or updating a TargetUsage. Track should be
// called _before_ attempting to use the Target. This ensures the
// managed resource's usage is updated if the managed resource is updated to
// reference a misconfigured Target.
func (u *TargetUsageTracker) Track(ctx context.Context, mg Managed) error {
	pcu := u.of.DeepCopyObject().(TargetUsage)
	gvk := mg.GetObjectKind().GroupVersionKind()
	ref := mg.GetTargetReference()
	if ref == nil {
		return errMissingRef{errors.New(errMissingTargetRef)}
	}

	pcu.SetName(string(mg.GetUID()))
	pcu.SetLabels(map[string]string{nddv1.LabelKeyTargetName: ref.Name})
	pcu.SetOwnerReferences([]metav1.OwnerReference{meta.AsController(meta.TypedReferenceTo(mg, gvk))})
	pcu.SetTargetReference(nddv1.Reference{Name: ref.Name})
	pcu.SetResourceReference(nddv1.TypedReference{
		ApiVersion: gvk.GroupVersion().String(),
		Kind:       gvk.Kind,
		Name:       mg.GetName(),
	})

	err := u.c.Apply(ctx, pcu,
		MustBeControllableBy(mg.GetUID()),
		AllowUpdateIf(func(current, _ runtime.Object) bool {
			return current.(TargetUsage).GetTargetReference().Name != pcu.GetTargetReference().Name
		}),
	)
	return errors.Wrap(Ignore(IsNotAllowed, err), errApplyTargetUsage)
}
