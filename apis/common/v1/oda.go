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

import "github.com/yndd/ndd-runtime/pkg/utils"

type OdaKind string

const (
	OdaKindOrganization    OdaKind = "organization"
	OdaKindDeployment      OdaKind = "deployment"
	OdaKindAvailabiityZone OdaKind = "availability-zone"
	OdaKindUnknown         OdaKind = "unknown"
)

func (s OdaKind) String() string {
	switch s {
	case OdaKindOrganization:
		return "organization"
	case OdaKindDeployment:
		return "deployment"
	case OdaKindAvailabiityZone:
		return "availability-zone"
	case OdaKindUnknown:
		return "unknown"
	}
	return "unknown"
}

type OdaInfo struct {
	Oda []Tag `json:"oda,omitempty"`
}

func NewOdaInfo(c ...Tag) *OdaInfo {
	x := &OdaInfo{}
	x.SetTags(c...)
	return x
}

// GetCondition returns the condition for the given ConditionKind if exists,
// otherwise returns nil
func (x *OdaInfo) GetOda(k string) Tag {
	for _, t := range x.Oda {
		if *t.Key == k {
			return t
		}
	}
	return Tag{Key: &k, Value: utils.StringPtr(OdaKindUnknown.String())}
}

// SetTags sets the supplied tags, replacing any existing tag
// of the same kind. This is a no-op if all supplied tags are identical,
// ignoring the last transition time, to those already set.
func (s *OdaInfo) SetTags(c ...Tag) {
	for _, new := range c {
		exists := false
		for i, existing := range s.Oda {
			if *existing.Key != *new.Key {
				continue
			}

			if existing.Equal(new) {
				exists = true
				continue
			}

			s.Oda[i] = new
			exists = true
		}
		if !exists {
			s.Oda = append(s.Oda, new)
		}
	}
}

func (x *OdaInfo) GetOrganization() string {
	t := x.GetOda(string(OdaKindOrganization))
	return *t.Value
}

func (x *OdaInfo) GetDeployment() string {
	t := x.GetOda(string(OdaKindDeployment))
	return *t.Value
}

func (x *OdaInfo) GetAvailabilityZone() string {
	t := x.GetOda(string(OdaKindAvailabiityZone))
	return *t.Value
}

func (x *OdaInfo) SetOrganization(s string) {
	tags := []Tag{
		{Key: utils.StringPtr(OdaKindOrganization.String()), Value: utils.StringPtr(s)},
	}
	x.SetTags(tags...)
}

func (x *OdaInfo) SetDeployment(s string) {
	tags := []Tag{
		{Key: utils.StringPtr(OdaKindDeployment.String()), Value: utils.StringPtr(s)},
	}
	x.SetTags(tags...)
}

func (x *OdaInfo) SetAvailabilityZone(s string) {
	tags := []Tag{
		{Key: utils.StringPtr(OdaKindAvailabiityZone.String()), Value: utils.StringPtr(s)},
	}
	x.SetTags(tags...)
}
