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

type OdaKind string

const (
	OdaKindOrganization    OdaKind = "organization"
	OdaKindDeployment      OdaKind = "deployment"
	OdaKindAvailabiityZone OdaKind = "availabilityZone"
	OdaKindUnknown         OdaKind = "unknown"
	OdaKindResourceName    OdaKind = "resourceName"
)

func (s OdaKind) String() string {
	switch s {
	case OdaKindOrganization:
		return "organization"
	case OdaKindDeployment:
		return "deployment"
	case OdaKindAvailabiityZone:
		return "availabilityZone"
	case OdaKindResourceName:
		return "resourceName"
	case OdaKindUnknown:
		return "unknown"
	}
	return "unknown"
}

type OdaInfo struct {
	//Oda []Tag `json:"oda,omitempty"`
	Oda map[string]string `json:"oda,omitempty"`
}

func (x *OdaInfo) GetOrganization() string {
	t, ok := x.Oda[OdaKindOrganization.String()]
	if !ok {
		return ""
	}
	return t
}

func (x *OdaInfo) GetDeployment() string {
	t, ok := x.Oda[OdaKindDeployment.String()]
	if !ok {
		return ""
	}
	return t
}

func (x *OdaInfo) GetAvailabilityZone() string {
	t, ok := x.Oda[OdaKindAvailabiityZone.String()]
	if !ok {
		return ""
	}
	return t
}

func (x *OdaInfo) SetOrganization(s string) {
	if x.Oda == nil {
		x.Oda = map[string]string{}
	}
	x.Oda[OdaKindOrganization.String()] = s
}

func (x *OdaInfo) SetDeployment(s string) {
	if x.Oda == nil {
		x.Oda = map[string]string{}
	}
	x.Oda[OdaKindDeployment.String()] = s
}

func (x *OdaInfo) SetAvailabilityZone(s string) {
	if x.Oda == nil {
		x.Oda = map[string]string{}
	}
	x.Oda[OdaKindAvailabiityZone.String()] = s
}

func (x *OdaInfo) SetResourceName(s string) {
	if x.Oda == nil {
		x.Oda = map[string]string{}
	}
	x.Oda[OdaKindResourceName.String()] = s
}
