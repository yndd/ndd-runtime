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

func (x *OdaInfo) GetOrganization() string {
	t, ok := x.Oda[OdaKind_OdaKind_Organization.String()]
	if !ok {
		return ""
	}
	return t
}

func (x *OdaInfo) GetDeployment() string {
	t, ok := x.Oda[OdaKind_OdaKind_Deployment.String()]
	if !ok {
		return ""
	}
	return t
}

func (x *OdaInfo) GetAvailabilityZone() string {
	t, ok := x.Oda[OdaKind_OdaKind_AvailabilityZone.String()]
	if !ok {
		return ""
	}
	return t
}

func (x *OdaInfo) SetOrganization(s string) {
	x.Oda[OdaKind_OdaKind_Organization.String()] = s
}

func (x *OdaInfo) SetDeployment(s string) {
	x.Oda[OdaKind_OdaKind_Deployment.String()] = s
}

func (x *OdaInfo) SetAvailabilityZone(s string) {
	x.Oda[OdaKind_OdaKind_AvailabilityZone.String()] = s
}
