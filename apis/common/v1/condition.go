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
	"sort"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Equal returns true if the condition is identical to the supplied condition,
// ignoring the LastTransitionTime.
func (c *Condition) Equal(other *Condition) bool {
	return c.Kind == other.Kind &&
		c.Status == other.Status &&
		c.Reason == other.Reason &&
		c.Message == other.Message
}

// WithMessage returns a condition by adding the provided message to existing
// condition.
func (c *Condition) WithMessage(msg string) *Condition {
	c.Message = msg
	return c
}

// NewConditionedStatus returns a stat with the supplied conditions set.
func NewConditionedStatus(c ...*Condition) *ConditionedStatus {
	s := &ConditionedStatus{}
	s.SetConditions(c...)
	return s
}

// GetCondition returns the condition for the given ConditionKind if exists,
// otherwise returns nil
func (s *ConditionedStatus) GetACondition(ck ConditionKind) Condition {
	for _, c := range s.Conditions {
		if c.Kind == ck {
			return *c
		}
	}
	return Condition{Kind: ck, Status: ConditionStatus_ConditionStatus_Unknown}
}

// SetConditions sets the supplied conditions, replacing any existing conditions
// of the same kind. This is a no-op if all supplied conditions are identical,
// ignoring the last transition time, to those already set.
func (s *ConditionedStatus) SetConditions(c ...*Condition) {
	for _, new := range c {
		exists := false
		for i, existing := range s.Conditions {
			if existing.Kind != new.Kind {
				continue
			}

			if existing.Equal(new) {
				exists = true
				continue
			}

			s.Conditions[i] = new
			exists = true
		}
		if !exists {
			s.Conditions = append(s.Conditions, new)
		}
	}
}

// Equal returns true if the status is identical to the supplied status,
// ignoring the LastTransitionTimes and order of statuses.
func (s *ConditionedStatus) Equal(other *ConditionedStatus) bool {
	if s == nil || other == nil {
		return s == nil && other == nil
	}

	if len(other.Conditions) != len(s.Conditions) {
		return false
	}

	sc := make([]*Condition, len(s.Conditions))
	copy(sc, s.Conditions)

	oc := make([]*Condition, len(other.Conditions))
	copy(oc, other.Conditions)

	// We should not have more than one condition of each kind.
	sort.Slice(sc, func(i, j int) bool { return sc[i].Kind < sc[j].Kind })
	sort.Slice(oc, func(i, j int) bool { return oc[i].Kind < oc[j].Kind })

	for i := range sc {
		if !sc[i].Equal(oc[i]) {
			return false
		}
	}

	return true
}

// Unknown returns a condition that indicates the resource is in an
// unknown status.
func Unknown() Condition {
	t := metav1.Now()
	return Condition{
		Kind:               ConditionKind_ConditionKind_Ready,
		Status:             ConditionStatus_ConditionStatus_False,
		LastTransitionTime: &t,
		Reason:             ConditionReason_ConditionReason_Unknown,
	}
}

// Creating returns a condition that indicates the resource is currently
// being created.
func Creating() Condition {
	t := metav1.Now()
	return Condition{
		Kind:               ConditionKind_ConditionKind_Ready,
		Status:             ConditionStatus_ConditionStatus_False,
		LastTransitionTime: &t,
		Reason:             ConditionReason_ConditionReason_Creating,
	}
}

// Updating returns a condition that indicates the resource is currently
// being updated.
func Updating() Condition {
	t := metav1.Now()
	return Condition{
		Kind:               ConditionKind_ConditionKind_Ready,
		Status:             ConditionStatus_ConditionStatus_False,
		LastTransitionTime: &t,
		Reason:             ConditionReason_ConditionReason_Updating,
	}
}

// Deleting returns a condition that indicates the resource is currently
// being deleted.
func Deleting() Condition {
	t := metav1.Now()
	return Condition{
		Kind:               ConditionKind_ConditionKind_Ready,
		Status:             ConditionStatus_ConditionStatus_False,
		LastTransitionTime: &t,
		Reason:             ConditionReason_ConditionReason_Deleting,
	}
}

// Available returns a condition that indicates the resource is
// currently observed to be available for use.
func Available() Condition {
	t := metav1.Now()
	return Condition{
		Kind:               ConditionKind_ConditionKind_Ready,
		Status:             ConditionStatus_ConditionStatus_True,
		LastTransitionTime: &t,
		Reason:             ConditionReason_ConditionReason_Available,
	}
}

// Unavailable returns a condition that indicates the resource is not
// currently available for use. Unavailable should be set only when ndd
// expects the resource to be available but knows it is not, for example
// because its API reports it is unhealthy.
func Unavailable() Condition {
	t := metav1.Now()
	return Condition{
		Kind:               ConditionKind_ConditionKind_Ready,
		Status:             ConditionStatus_ConditionStatus_False,
		LastTransitionTime: &t,
		Reason:             ConditionReason_ConditionReason_Unavailable,
	}
}

// Failed returns a condition that indicates the resource
// failed to get instantiated.
func Failed(msg string) Condition {
	t := metav1.Now()
	return Condition{
		Kind:               ConditionKind_ConditionKind_Ready,
		Status:             ConditionStatus_ConditionStatus_False,
		LastTransitionTime: &t,
		Reason:             ConditionReason_ConditionReason_Failed,
		Message:            msg,
	}
}

// Pending returns a condition that indicates the resource is not
// currently available for use and is still being processed
func Pending() Condition {
	t := metav1.Now()
	return Condition{
		Kind:               ConditionKind_ConditionKind_Ready,
		Status:             ConditionStatus_ConditionStatus_False,
		LastTransitionTime: &t,
		Reason:             ConditionReason_ConditionReason_Pending,
	}
}

// ReconcileSuccess returns a condition indicating that ndd successfully
// completed the most recent reconciliation of the resource.
func ReconcileSuccess() Condition {
	t := metav1.Now()
	return Condition{
		Kind:               ConditionKind_ConditionKind_Synced,
		Status:             ConditionStatus_ConditionStatus_True,
		LastTransitionTime: &t,
		Reason:             ConditionReason_ConditionReason_ReconcileSuccess,
	}
}

// ReconcileError returns a condition indicating that ndd encountered an
// error while reconciling the resource. This could mean ndd was
// unable to update the resource to reflect its desired state, or that
// ndd was unable to determine the current actual state of the resource.
func ReconcileError(err error) Condition {
	t := metav1.Now()
	return Condition{
		Kind:               ConditionKind_ConditionKind_Synced,
		Status:             ConditionStatus_ConditionStatus_False,
		LastTransitionTime: &t,
		Reason:             ConditionReason_ConditionReason_ReconcileFailed,
	}
}

// TargetFound returns a condition that indicates the resource has
// target(s) available for use.
func TargetFound() Condition {
	t := metav1.Now()
	return Condition{
		Kind:               ConditionKind_ConditionKind_Target,
		Status:             ConditionStatus_ConditionStatus_True,
		LastTransitionTime: &t,
		Reason:             ConditionReason_ConditionReason_Success,
	}
}

// TargetNotFound returns a condition that indicates the resource has no
// target(s) available for use.
func TargetNotFound() Condition {
	t := metav1.Now()
	return Condition{
		Kind:               ConditionKind_ConditionKind_Target,
		Status:             ConditionStatus_ConditionStatus_False,
		LastTransitionTime: &t,
		Reason:             ConditionReason_ConditionReason_Failed,
	}
}

// LeafRefValidationSuccess returns a condition that indicates
// the resource leafreference(s) are found or no leafrefs exist
func RootPathValidationSuccess() Condition {
	t := metav1.Now()
	return Condition{
		Kind:               ConditionKind_ConditionKind_RootPath,
		Status:             ConditionStatus_ConditionStatus_True,
		LastTransitionTime: &t,
		Reason:             ConditionReason_ConditionReason_Success,
	}
}

// LeafRefValidationFailure returns a condition that indicates
// the resource leafreference(s) are missing
func RootPathValidationFailure() Condition {
	t := metav1.Now()
	return Condition{
		Kind:               ConditionKind_ConditionKind_RootPath,
		Status:             ConditionStatus_ConditionStatus_False,
		LastTransitionTime: &t,
		Reason:             ConditionReason_ConditionReason_Failed,
	}
}

// LeafRefValidationUnknown returns a condition that indicates
// the internal leafref validation is unknown
func RootPathValidationUnknown() Condition {
	t := metav1.Now()
	return Condition{
		Kind:               ConditionKind_ConditionKind_RootPath,
		Status:             ConditionStatus_ConditionStatus_False,
		LastTransitionTime: &t,
		Reason:             ConditionReason_ConditionReason_Unknown,
	}
}
