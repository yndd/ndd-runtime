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

package managed

import (
	"context"

	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/yndd/ndd-runtime/pkg/resource"
	"github.com/yndd/nddp-system/pkg/ygotnddp"
)

type Validator interface {
	ValidateRootPaths(ctx context.Context, mg resource.Managed, resourceList map[string]*ygotnddp.NddpSystem_Gvk) (ValidateRootPathsObservation, error)
}

type ValidatorFn struct {
	ValidateRootPathsFn func(ctx context.Context, mg resource.Managed, resourceList map[string]*ygotnddp.NddpSystem_Gvk) (ValidateRootPathsObservation, error)
}

func (e ValidatorFn) ValidateRootPaths(ctx context.Context, mg resource.Managed, resourceList map[string]*ygotnddp.NddpSystem_Gvk) (ValidateRootPathsObservation, error) {
	return e.ValidateRootPathsFn(ctx, mg, resourceList)
}

type NopValidator struct{}

func (e *NopValidator) ValidateRootPaths(ctx context.Context, mg resource.Managed, resourceList map[string]*ygotnddp.NddpSystem_Gvk) (ValidateRootPathsObservation, error) {
	return ValidateRootPathsObservation{}, nil
}

type ValidateRootPathsObservation struct {
	Changed     bool
	RootPaths   []string
	HierPaths   map[string][]string
	DeletePaths []*gnmi.Path
}
