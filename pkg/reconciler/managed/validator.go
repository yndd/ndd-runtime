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
	ValidateResource(ctx context.Context, mg resource.Managed, systemCfg *ygotnddp.Device) (ValidateResourceObservation, error)

	ValidateConfig(ctx context.Context, mg resource.Managed, systemCfg *ygotnddp.Device, runningCfg []byte) (ValidateConfigObservation, error)
}

type ValidatorFn struct {
	ValidateResourceFn func(ctx context.Context, mg resource.Managed, systemCfg *ygotnddp.Device) (ValidateResourceObservation, error)
	ValidateConfigFn   func(ctx context.Context, mg resource.Managed, systemCfg *ygotnddp.Device, runningCfg []byte) (ValidateConfigObservation, error)
}

func (e ValidatorFn) ValidateResource(ctx context.Context, mg resource.Managed, systemCfg *ygotnddp.Device) (ValidateResourceObservation, error) {
	return e.ValidateResourceFn(ctx, mg, systemCfg)
}

func (e ValidatorFn) ValidateConfig(ctx context.Context, mg resource.Managed, systemCfg *ygotnddp.Device, runningCfg []byte) (ValidateConfigObservation, error) {
	return e.ValidateConfigFn(ctx, mg, systemCfg, runningCfg)
}

type NopValidator struct{}

func (e *NopValidator) ValidateResource(ctx context.Context, mg resource.Managed, systemCfg *ygotnddp.Device) (ValidateResourceObservation, error) {
	return ValidateResourceObservation{}, nil
}

func (e *NopValidator) ValidateConfig(ctx context.Context, mg resource.Managed, systemCfg *ygotnddp.Device, runningCfg []byte) (ValidateConfigObservation, error) {
	return ValidateConfigObservation{}, nil
}

type ValidateResourceObservation struct {
	// indicates if the device is exhausted or not, this can happen when too many transactions
	// occured towards the device
	Exhausted bool
	// indicates if the device in the proxy-cache is ready or not, during proxy-cache
	// startup this can occur during proxy-cache initialization
	Ready bool
	// indicates if the MR is not yet reconciled, the reconciler should wait for further actions
	Pending bool
	// indicates if a MR exists in the cache
	Exists bool
	// indicates if the resource spec was not successfully applied to the device
	// unless the resourceSpec changes the transaction would not be successfull
	// we dont try to reconcile unless the spec changed
	Failed bool
	// Provides additional information why a failure occurs
	Message string
	
}

type ValidateConfigObservation struct {
	ValidateSucces bool
	// Provides additional information why a validation failured occured
	Message     string
	// indicates if the spec was changed and subsequent actionsare required
	Changed     bool
	RootPaths   []string
	DeletePaths []*gnmi.Path
}
