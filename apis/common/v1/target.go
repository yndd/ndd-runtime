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
	"github.com/karimra/gnmic/target"
	"github.com/karimra/gnmic/types"
)

// +k8s:deepcopy-gen=false
type Target struct {
	// Name of the Target
	Name   string  `json:"name,omitempty"`
	// Configuration that is used by the gnmi client
	Config *types.TargetConfig `json:"config,omitempty"`
	// Target gnmi client
	Target *target.Target  `json:"target,omitempty"`
}

func (t *Target) IsTargetDeleted(ns []string) bool {
	for _, an := range ns {
		if an == t.Name {
			return false
		}
	}
	return true
}
