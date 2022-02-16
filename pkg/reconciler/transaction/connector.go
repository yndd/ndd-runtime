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

package transaction

import (
	"context"

	"github.com/yndd/ndd-runtime/pkg/tresource"
	systemv1alpha1 "github.com/yndd/nddp-system/apis/system/v1alpha1"
)

// ConnectionDetails created or updated during an operation on an external
// resource, for example usernames, passwords, endpoints, ports, etc.
type ConnectionDetails map[string][]byte

// An ExternalConnecter produces a new ExternalClient given the supplied
// Managed resource.
type ExternalConnecter interface {
	// Connect to the provider specified by the supplied managed resource and
	// produce an ExternalClient.
	Connect(ctx context.Context, tr tresource.Transaction, deviceName string) (ExternalClient, error)
}

// An ExternalConnectorFn is a function that satisfies the ExternalConnecter
// interface.
type ExternalConnectorFn func(ctx context.Context, tr tresource.Transaction, deviceName string) (ExternalClient, error)

// Connect to the provider specified by the supplied managed resource and
// produce an ExternalClient.
func (ec ExternalConnectorFn) Connect(ctx context.Context, tr tresource.Transaction, deviceName string) (ExternalClient, error) {
	return ec(ctx, tr, deviceName)
}

// An ExternalClient manages the lifecycle of an external resource.
// None of the calls here should be blocking. All of the calls should be
// idempotent. For example, Create call should not return AlreadyExists error
// if it's called again with the same parameters or Delete call should not
// return error if there is an ongoing deletion or resource does not exist.
type ExternalClient interface {
	// Observe the transaction in the cache
	Observe(ctx context.Context, tr tresource.Transaction) (ExternalObservation, error)

	// Create a transaction in the cache
	Create(ctx context.Context, tr tresource.Transaction, gvkList []string) error

	// Delete a transaction in the cache
	Delete(ctx context.Context, tr tresource.Transaction, gvkList []string) error

	// Close
	Close()
}

// ExternalClientFns are a series of functions that satisfy the ExternalClient
// interface.
type ExternalClientFns struct {
	ObserveFn func(ctx context.Context, tr tresource.Transaction) (ExternalObservation, error)
	CreateFn  func(ctx context.Context, tr tresource.Transaction, gvkList []string) error
	DeleteFn  func(ctx context.Context, tr tresource.Transaction, gvkList []string) error
	CloseFn   func()
}

// Observe the external resource the supplied Managed resource represents, if
// any.
func (e ExternalClientFns) Observe(ctx context.Context, tr tresource.Transaction) (ExternalObservation, error) {
	return e.ObserveFn(ctx, tr)
}

// Create an external resource per the specifications of the supplied Managed
// resource.
func (e ExternalClientFns) Create(ctx context.Context, tr tresource.Transaction, gvkList []string) error {
	return e.CreateFn(ctx, tr, gvkList)
}

// Delete the external resource upon deletion of its associated Managed
// resource.
func (e ExternalClientFns) Delete(ctx context.Context, tr tresource.Transaction, gvkList []string) error {
	return e.DeleteFn(ctx, tr, gvkList)
}

// GetResourceName returns the resource matching the path
func (e ExternalClientFns) Close() {}

// A NopConnecter does nothing.
type NopConnecter struct{}

// Connect returns a NopClient. It never returns an error.
func (c *NopConnecter) Connect(_ context.Context, _ tresource.Transaction, _ string) (ExternalClient, error) {
	return &NopClient{}, nil
}

// A NopClient does nothing.
type NopClient struct{}

// Observe does nothing. It returns an empty ExternalObservation and no error.
func (c *NopClient) Observe(ctx context.Context, tr tresource.Transaction) (ExternalObservation, error) {
	return ExternalObservation{}, nil
}

// Create does nothing. It returns an empty ExternalCreation and no error.
func (c *NopClient) Create(ctx context.Context, tr tresource.Transaction, gvkList []string) error { return nil }

// Delete does nothing. It never returns an error.
func (c *NopClient) Delete(ctx context.Context, tr tresource.Transaction, gvkList []string) error { return nil }

func (c *NopClient) Close() {}

// An ExternalObservation is the result of an observation of an external
// resource.
type ExternalObservation struct {
	// indicated if the cache is exhausted or not, can reflect the status of the devic
	Exhausted bool
	// indicated if the cache is ready or not, during cache startup this can occur
	// when the cache is still initializing
	Ready bool
	// ResourceExists must be true if a corresponding transaction exists in the cache
	// but status is pending
	Exists bool
	// indicates if the transaction was successfully applied to the device or not
	// represents failed or success status of the transaction
	Success bool
	// inidcates the
	Pending bool
	// the gvk resources applied to the cache
	GvkResourceList []*systemv1alpha1.Gvk
}
