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
	// GetCrStatus gets the status of the cr like pending, failed, not ready and also of the device is exhausted or not
	// this is used to trigger subsequent actions in the reconciler
	GetCrStatus(ctx context.Context, mg resource.Managed, systemCfg *ygotnddp.Device) (CrObservation, error)

	// ValidateCrSpecUpdate merges the running config and the new spec and validates the result.
	// If the result was not successfull the error message is returned to provide ctx of the error
	ValidateCrSpecUpdate(ctx context.Context, mg resource.Managed, runningCfg []byte) (CrSpecObservation, error)

	// ValidateCrSpecDelete deletes the rootPaths from the running config and validates the result.
	// If the result was not successfull the error message is returned to provide ctx of the error
	ValidateCrSpecDelete(ctx context.Context, mg resource.Managed, runningCfg []byte) (CrSpecObservation, error)

	// GetCrSpecDiff checks if the current cr spec differs from the original cr spec and triggers the necessary updates
	// to align the current cr spec with original spec. The method returns gnmi deletes and updates
	GetCrSpecDiff(ctx context.Context, mg resource.Managed, systemCfg *ygotnddp.Device) (CrSpecDiffObservation, error)

	// GetCrActualDiff checks if the current cr spec differs from the running config and triggers the necessary updates
	// to align the running config with the cr spec. The method returns gnmi deletes and updates
	GetCrActualDiff(ctx context.Context, mg resource.Managed, runningCfg []byte) (CrActualDiffObservation, error)

	// GetRootPaths returns the rootPaths of the current Spec
	GetRootPaths(ctx context.Context, mg resource.Managed) ([]string, error)
}

type ValidatorFn struct {
	GetCrStatusFn          func(ctx context.Context, mg resource.Managed, systemCfg *ygotnddp.Device) (CrObservation, error)
	ValidateCrSpecUpdateFn func(ctx context.Context, mg resource.Managed, runningCfg []byte) (CrSpecObservation, error)
	ValidateCrSpecDeleteFn func(ctx context.Context, mg resource.Managed, runningCfg []byte) (CrSpecObservation, error)
	GetCrSpecDiffFn        func(ctx context.Context, mg resource.Managed, systemCfg *ygotnddp.Device) (CrSpecDiffObservation, error)
	GetCrActualDiffFn      func(ctx context.Context, mg resource.Managed, runningCfg []byte) (CrActualDiffObservation, error)
	GetRootPathsFn         func(ctx context.Context, mg resource.Managed) ([]string, error)
}

func (e ValidatorFn) GetCrStatus(ctx context.Context, mg resource.Managed, systemCfg *ygotnddp.Device) (CrObservation, error) {
	return e.GetCrStatusFn(ctx, mg, systemCfg)
}

func (e ValidatorFn) ValidateCrSpecUpdate(ctx context.Context, mg resource.Managed, runningCfg []byte) (CrSpecObservation, error) {
	return e.ValidateCrSpecUpdateFn(ctx, mg, runningCfg)
}

func (e ValidatorFn) ValidateCrSpecDelete(ctx context.Context, mg resource.Managed, runningCfg []byte) (CrSpecObservation, error) {
	return e.ValidateCrSpecUpdateFn(ctx, mg, runningCfg)
}

func (e ValidatorFn) GetCrSpecDiff(ctx context.Context, mg resource.Managed, systemCfg *ygotnddp.Device) (CrSpecDiffObservation, error) {
	return e.GetCrSpecDiffFn(ctx, mg, systemCfg)
}

func (e ValidatorFn) GetCrActualDiff(ctx context.Context, mg resource.Managed, runningCfg []byte) (CrActualDiffObservation, error) {
	return e.GetCrActualDiffFn(ctx, mg, runningCfg)
}

func (e ValidatorFn) GetRootPaths(ctx context.Context, mg resource.Managed) ([]string, error) {
	return e.GetRootPathsFn(ctx, mg)
}

type NopValidator struct{}

func (e *NopValidator) GetCrStatus(ctx context.Context, mg resource.Managed, systemCfg *ygotnddp.Device) (CrObservation, error) {
	return CrObservation{}, nil
}

func (e *NopValidator) ValidateCrSpecUpdate(ctx context.Context, mg resource.Managed, runningCfg []byte) (CrSpecObservation, error) {
	return CrSpecObservation{}, nil
}

func (e *NopValidator) ValidateCrSpecDelete(ctx context.Context, mg resource.Managed, runningCfg []byte) (CrSpecObservation, error) {
	return CrSpecObservation{}, nil
}

func (e *NopValidator) GetCrSpecDiff(ctx context.Context, mg resource.Managed, systemCfg *ygotnddp.Device) (CrSpecDiffObservation, error) {
	return CrSpecDiffObservation{}, nil
}

func (e *NopValidator) GetCrActualDiff(ctx context.Context, mg resource.Managed, runningCfg []byte) (CrActualDiffObservation, error) {
	return CrActualDiffObservation{}, nil
}

func (e *NopValidator) GetRootPaths(ctx context.Context, mg resource.Managed) ([]string, error) {
	return []string{}, nil
}

type CrObservation struct {
	// indicates if the device is exhausted or not, this can happen when too many api calls
	// occured towards the device
	Exhausted bool
	// indicates if the device in the proxy-cache is ready or not,
	// during proxy-cache initialization it can occur that the cache is not ready
	Ready bool
	// indicates if the MR is not yet reconciled, the reconciler should wait for further actions
	Pending bool
	// indicates if a MR exists in the proxy-cache
	Exists bool
	// indicates if the MR was not successfully applied to the device
	// unless the resourceSpec changes the transaction would not be successfull
	// we dont try to reconcile unless the spec changed
	Failed bool
	// Provides additional information why a MR is in failed status
	Message string
}

type CrSpecObservation struct {
	// validation was successfull
	Success bool
	// Provides additional information why a validation failured occured
	Message string
}

type CrSpecDiffObservation struct {
	// indicates if the spec was changed and subsequent actionsare required
	Deletes []*gnmi.Path
	Updates []*gnmi.Update
}

type CrActualDiffObservation struct {
	// ResourceHasData can be true when a managed resource is created, but the
	// device had already data in that resource. The data needs to get aligned
	// with the intended resource data
	HasData bool
	// ResourceUpToDate should be true if the corresponding external resource
	// appears to be up-to-date with the resourceSpec
	IsUpToDate bool
	// when the resource is not up to date these 2 parameter determine what to do to realign the resource to the spec
	Deletes []*gnmi.Path
	Updates []*gnmi.Update
}
