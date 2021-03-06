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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type HealthCondition struct {
	// Kind of this condition. At most one of each condition kind may apply to
	// a resource at any point in time.
	ResourceName string `json:"resourceName"`

	HealthKind string `json:"healthKind"`

	// Status of this condition; is it currently True, False, or Unknown?
	Status corev1.ConditionStatus `json:"status"`

	// LastTransitionTime is the last time this condition transitioned from one
	// status to another.
	LastTransitionTime metav1.Time `json:"lastTransitionTime"`

	// A Reason for this condition's last transition from one status to another.
	Reason string `json:"reason,omitempty"`

	// A Message containing details about this condition's last transition from
	// one status to another, if any.
	// +optional
	Message string `json:"message,omitempty"`
}

// Equal returns true if the condition is identical to the supplied condition,
// ignoring the LastTransitionTime.
func (c HealthCondition) Equal(other HealthCondition) bool {
	return c.ResourceName == other.ResourceName &&
		c.HealthKind == other.HealthKind &&
		c.Status == other.Status &&
		c.Reason == other.Reason &&
		c.Message == other.Message
}

// WithMessage returns a condition by adding the provided message to existing
// condition.
func (c HealthCondition) WithMessage(msg string) HealthCondition {
	c.Message = msg
	return c
}

type HealthConditionedStatus struct {
	// Status of the health in percentage
	Percentage uint32 `json:"percentage,omitempty"`

	// LastTransitionTime is the last time this condition transitioned from one
	// status to another.
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`

	// HealthConditions that determine the health status.
	// +optional
	HealthConditions []HealthCondition `json:"healthConditions,omitempty"`
}

// NewConditionedStatus returns a stat with the supplied conditions set.
func NewHealthConditionedStatus(p uint32, c ...HealthCondition) *HealthConditionedStatus {
	s := &HealthConditionedStatus{
		Percentage:         p,
		LastTransitionTime: metav1.Now(),
	}
	s.SetHealthConditions(c...)
	return s
}

// GetCondition returns the condition for the given ConditionKind if exists,
// otherwise returns nil
func (s *HealthConditionedStatus) GetHealthCondition(resourceName, hck string) HealthCondition {
	for _, c := range s.HealthConditions {
		if c.ResourceName == resourceName && c.HealthKind == hck {
			return c
		}
	}
	return HealthCondition{ResourceName: resourceName, HealthKind: hck, Status: corev1.ConditionUnknown}
}

// SetConditions sets the supplied conditions, replacing any existing conditions
// of the same kind. This is a no-op if all supplied conditions are identical,
// ignoring the last transition time, to those already set.
func (s *HealthConditionedStatus) SetHealthConditions(c ...HealthCondition) {
	for _, new := range c {
		exists := false
		for i, existing := range s.HealthConditions {
			if existing.ResourceName != new.ResourceName {
				continue
			}

			if existing.HealthKind != new.HealthKind {
				continue
			}

			if existing.Equal(new) {
				exists = true
				continue
			}

			s.HealthConditions[i] = new
			exists = true
		}
		if !exists {
			s.HealthConditions = append(s.HealthConditions, new)
		}
	}
}

// Equal returns true if the status is identical to the supplied status,
// ignoring the LastTransitionTimes and order of statuses.
func (s *HealthConditionedStatus) Equal(other *HealthConditionedStatus) bool {
	if s == nil || other == nil {
		return s == nil && other == nil
	}

	if len(other.HealthConditions) != len(s.HealthConditions) {
		return false
	}

	sc := make([]HealthCondition, len(s.HealthConditions))
	copy(sc, s.HealthConditions)

	oc := make([]HealthCondition, len(other.HealthConditions))
	copy(oc, other.HealthConditions)

	// We should not have more than one condition of each kind.
	sort.Slice(sc, func(i, j int) bool { return sc[i].HealthKind < sc[j].HealthKind })
	sort.Slice(oc, func(i, j int) bool { return oc[i].HealthKind < oc[j].HealthKind })

	for i := range sc {
		if !sc[i].Equal(oc[i]) {
			return false
		}
	}

	return true
}
