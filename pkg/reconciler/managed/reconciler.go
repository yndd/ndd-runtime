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
	"fmt"
	"strings"
	"time"

	"github.com/openconfig/gnmi/proto/gnmi"
	nddv1 "github.com/yndd/ndd-runtime/apis/common/v1"
	"github.com/yndd/ndd-yang/pkg/parser"
	"github.com/yndd/ndd-yang/pkg/yparser"

	"github.com/pkg/errors"
	"github.com/yndd/ndd-runtime/pkg/event"
	"github.com/yndd/ndd-runtime/pkg/logging"
	"github.com/yndd/ndd-runtime/pkg/meta"
	"github.com/yndd/ndd-runtime/pkg/resource"
	"github.com/yndd/nddp-system/pkg/gvkresource"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	// timers
	managedFinalizerName = "finalizer.managedresource.ndd.yndd.io"
	reconcileGracePeriod = 30 * time.Second
	reconcileTimeout     = 1 * time.Minute
	shortWait            = 30 * time.Second
	mediumWait           = 1 * time.Minute
	veryShortWait        = 1 * time.Second
	longWait             = 1 * time.Minute

	defaultpollInterval = 1 * time.Minute

	// errors
	errGetManaged                        = "cannot get managed resource"
	errUpdateManagedAfterCreate          = "cannot update managed resource. this may have resulted in a leaked external resource"
	errReconcileConnect                  = "connect failed"
	errReconcileObserve                  = "observe failed"
	errReconcileCreate                   = "create failed"
	errReconcileUpdate                   = "update failed"
	errReconcileDelete                   = "delete failed"
	errReconcileGetConfig                = "get config failed"
	errReconcileGetResourceName          = "get resource name failed"
	errReconcileValidateLeafRef          = "leafref validation failed"
	errReconcileValidateParentDependency = "parent dependency validation failed"
	errReconcileValidateResourceIndexes  = "resource indexes validation failed"
	errUpdateManagedStatus               = "cannot update status of the managed resource"

	// Event reasons.
	reasonCannotConnect                  event.Reason = "CannotConnectToProvider"
	reasonCannotInitialize               event.Reason = "CannotInitializeManagedResource"
	reasonCannotResolveRefs              event.Reason = "CannotResolveResourceReferences"
	reasonCannotObserve                  event.Reason = "CannotObserveExternalResource"
	reasonCannotCreate                   event.Reason = "CannotCreateExternalResource"
	reasonCannotDelete                   event.Reason = "CannotDeleteExternalResource"
	reasonCannotPublish                  event.Reason = "CannotPublishConnectionDetails"
	reasonCannotUnpublish                event.Reason = "CannotUnpublishConnectionDetails"
	reasonCannotUpdate                   event.Reason = "CannotUpdateExternalResource"
	reasonCannotUpdateManaged            event.Reason = "CannotUpdateManagedResource"
	reasonCannotGetConfig                event.Reason = "CannotGetConfigExternalResource"
	reasonCannotGetResourceName          event.Reason = "CannotGetresourceNameExternalResource"
	reasonCannotValidateLeafRef          event.Reason = "CannotValidateLeafRef"
	reasonCannotValidateParentDependency event.Reason = "CannotValidateParentDependency"
	reasonCannotGetValidTarget           event.Reason = "CannotGetValidTarget"
	reasonValidateLeafRefFailed          event.Reason = "ValidatelLeafRefFailed"
	reasonValidateParentDependencyFailed event.Reason = "ValidateParentDependencyFailed"
	reasonCannotValidateResourceIndexes  event.Reason = "CannotValidateResourceIndexes"

	reasonDeleted event.Reason = "DeletedExternalResource"
	reasonCreated event.Reason = "CreatedExternalResource"
	reasonUpdated event.Reason = "UpdatedExternalResource"
)

// ControllerName returns the recommended name for controllers that use this
// package to reconcile a particular kind of managed resource.
func ControllerName(kind string) string {
	return "managed/" + strings.ToLower(kind)
}

// A Reconciler reconciles managed resources by creating and managing the
// lifecycle of an external resource, i.e. a resource in an external network
// device through an API. Each controller must watch the managed resource kind
// for which it is responsible.
type Reconciler struct {
	client     client.Client
	newManaged func() resource.Managed

	pollInterval time.Duration
	timeout      time.Duration
	parser       parser.Parser

	// The below structs embed the set of interfaces used to implement the
	// managed resource reconciler. We do this primarily for readability, so
	// that the reconciler logic reads r.external.Connect(),
	// r.managed.Delete(), etc.
	external  mrExternal
	managed   mrManaged
	validator mrValidator

	log    logging.Logger
	record event.Recorder
}

type mrValidator struct {
	Validator
}

func defaultMRValidator() mrValidator {
	return mrValidator{
		Validator: &NopValidator{},
	}
}

type mrManaged struct {
	resource.Finalizer
}

func defaultMRManaged(m manager.Manager) mrManaged {
	return mrManaged{
		Finalizer: resource.NewAPIFinalizer(m.GetClient(), managedFinalizerName),
	}
}

type mrExternal struct {
	ExternalConnecter
}

func defaultMRExternal() mrExternal {
	return mrExternal{
		ExternalConnecter: &NopConnecter{},
	}
}

// A ReconcilerOption configures a Reconciler.
type ReconcilerOption func(*Reconciler)

// WithTimeout specifies the timeout duration cumulatively for all the calls happen
// in the reconciliation function. In case the deadline exceeds, reconciler will
// still have some time to make the necessary calls to report the error such as
// status update.
func WithTimeout(duration time.Duration) ReconcilerOption {
	return func(r *Reconciler) {
		r.timeout = duration
	}
}

// WithPollInterval specifies how long the Reconciler should wait before queueing
// a new reconciliation after a successful reconcile. The Reconciler requeues
// after a specified duration when it is not actively waiting for an external
// operation, but wishes to check whether an existing external resource needs to
// be synced to its ndd Managed resource.
func WithPollInterval(after time.Duration) ReconcilerOption {
	return func(r *Reconciler) {
		r.pollInterval = after
	}
}

// WithExternalConnecter specifies how the Reconciler should connect to the API
// used to sync and delete external resources.
func WithExternalConnecter(c ExternalConnecter) ReconcilerOption {
	return func(r *Reconciler) {
		r.external.ExternalConnecter = c
	}
}

func WithValidator(v Validator) ReconcilerOption {
	return func(r *Reconciler) {
		r.validator.Validator = v
	}
}

// WithFinalizer specifies how the Reconciler should add and remove
// finalizers to and from the managed resource.
func WithFinalizer(f resource.Finalizer) ReconcilerOption {
	return func(r *Reconciler) {
		r.managed.Finalizer = f
	}
}

// WithLogger specifies how the Reconciler should log messages.
func WithLogger(l logging.Logger) ReconcilerOption {
	return func(r *Reconciler) {
		r.log = l
	}
}

// WithRecorder specifies how the Reconciler should record events.
func WithRecorder(er event.Recorder) ReconcilerOption {
	return func(r *Reconciler) {
		r.record = er
	}
}

func WithParser(l logging.Logger) ReconcilerOption {
	return func(r *Reconciler) {
		r.parser = *parser.NewParser(parser.WithLogger(l))
	}
}

// NewReconciler returns a Reconciler that reconciles managed resources of the
// supplied ManagedKind with resources in an external network device.
// It panics if asked to reconcile a managed resource kind that is
// not registered with the supplied manager's runtime.Scheme. The returned
// Reconciler reconciles with a dummy, no-op 'external system' by default;
// callers should supply an ExternalConnector that returns an ExternalClient
// capable of managing resources in a real system.
func NewReconciler(m manager.Manager, of resource.ManagedKind, o ...ReconcilerOption) *Reconciler {
	nm := func() resource.Managed {
		return resource.MustCreateObject(schema.GroupVersionKind(of), m.GetScheme()).(resource.Managed)
	}

	// Panic early if we've been asked to reconcile a resource kind that has not
	// been registered with our controller manager's scheme.
	_ = nm()

	r := &Reconciler{
		client:       m.GetClient(),
		newManaged:   nm,
		pollInterval: defaultpollInterval,
		timeout:      reconcileTimeout,
		managed:      defaultMRManaged(m),
		external:     defaultMRExternal(),
		validator:    defaultMRValidator(),
		//resolver:     defaultMRResolver(),
		log:    logging.NewNopLogger(),
		record: event.NewNopRecorder(),
	}

	for _, ro := range o {
		ro(r)
	}

	return r
}

// Reconcile a managed resource with an external resource.
func (r *Reconciler) Reconcile(_ context.Context, req reconcile.Request) (reconcile.Result, error) { // nolint:gocyclo
	log := r.log.WithValues("request", req)
	log.Debug("Reconciling")

	ctx, cancel := context.WithTimeout(context.Background(), r.timeout+reconcileGracePeriod)
	defer cancel()

	// will be augmented with a cancel function in the grpc call
	externalCtx := context.Background() // nolint:govet

	managed := r.newManaged()
	if err := r.client.Get(ctx, req.NamespacedName, managed); err != nil {
		// There's no need to requeue if we no longer exist. Otherwise we'll be
		// requeued implicitly because we return an error.
		log.Debug("Cannot get managed resource", "error", err)
		return reconcile.Result{}, errors.Wrap(resource.IgnoreNotFound(err), errGetManaged)
	}

	record := r.record.WithAnnotations("external-name", meta.GetExternalName(managed))
	log = log.WithValues(
		"uid", managed.GetUID(),
		"version", managed.GetResourceVersion(),
	)

	// this is done for transaction to ensure we add a finalizer
	if err := r.managed.AddFinalizer(ctx, managed); err != nil {
		// If this is the first time we encounter this issue we'll be requeued
		// implicitly when we update our status with the new error condition. If
		// not, we requeue explicitly, which will trigger backoff.
		log.Debug("Cannot add finalizer", "error", err)
		managed.SetConditions(nddv1.ReconcileError(err), nddv1.Unknown())
		return reconcile.Result{Requeue: true}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
	}

	// If managed resource has a deletion timestamp and and a deletion policy of
	// Orphan, we do not need to observe the external resource before attempting
	// to remove finalizer.
	if meta.WasDeleted(managed) && managed.GetDeletionPolicy() == nddv1.DeletionOrphan {
		log = log.WithValues("deletion-timestamp", managed.GetDeletionTimestamp())
		managed.SetConditions(nddv1.Deleting())


		// check if the resource has external leafref dependencies, if so it cannot be deleted
		hasOtherFinalizer, err := r.managed.HasOtherFinalizer(ctx, managed)
		if err != nil {
			log.Debug("Cannot get managed resource finalizer", "error", err)
			managed.SetConditions(nddv1.ReconcileError(err), nddv1.Unknown())
			return reconcile.Result{Requeue: true}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
		}

		// if other finalizer exists we cananot delete the resource and we have to requeue until the
		// resource get deleted
		if hasOtherFinalizer {
			return reconcile.Result{RequeueAfter: veryShortWait}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
		}

		// remove the finalizer from the external resources before deleting the finalizer from the local resource
		for _, externalResourceName := range managed.GetExternalLeafRefs() {
			// for all external resource name that were resolved we need to remove the finalizer from the external resource
			if err := r.HandleExternalResourceFinalizer(ctx, FinalizerOperationRemove, externalResourceName, managed); err != nil {
				managed.SetConditions(nddv1.ReconcileError(err), nddv1.Unknown())
				return reconcile.Result{Requeue: true}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
			}
		}

		if err := r.managed.RemoveFinalizer(ctx, managed); err != nil {
			// If this is the first time we encounter this issue we'll be
			// requeued implicitly when we update our status with the new error
			// condition. If not, we requeue explicitly, which will trigger
			// backoff.
			log.Debug("Cannot remove managed resource finalizer", "error", err)
			managed.SetConditions(nddv1.ReconcileError(err), nddv1.Unknown())
			return reconcile.Result{Requeue: true}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
		}

		// We've successfully removed our finalizer. If we assume we were the only
		// controller that added a finalizer to this resource then it should no
		// longer exist and thus there is no point trying to update its status.
		// log.Debug("Successfully deleted managed resource")
		return reconcile.Result{Requeue: false}, nil
	}

	external, err := r.external.Connect(externalCtx, managed)
	if err != nil {
		if meta.WasDeleted(managed) {
			// when there is no target and we were requested to be deleted we can remove the
			// finalizer since the target is no longer there and we assume cleanup will happen
			// during target delete/create
			// however if the resource has external leafref dependencies, we cannot delete
			// resource
			hasOtherFinalizer, err := r.managed.HasOtherFinalizer(ctx, managed)
			if err != nil {
				log.Debug("Cannot get managed resource finalizer", "error", err)
				managed.SetConditions(nddv1.ReconcileError(err), nddv1.Unknown())
				return reconcile.Result{Requeue: true}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
			}

			// if other finalizer exists we cananot delete the resource and we have to requeue until the
			// resource get deleted
			if hasOtherFinalizer {
				return reconcile.Result{RequeueAfter: veryShortWait}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
			}

			// remove the finalizer from the external resources before deleting the finalizer from the local resource
			for _, externalResourceName := range managed.GetExternalLeafRefs() {
				// for all external resource name that were resolved we need to remove the finalizer from the external resource
				if err := r.HandleExternalResourceFinalizer(ctx, FinalizerOperationRemove, externalResourceName, managed); err != nil {
					managed.SetConditions(nddv1.ReconcileError(err), nddv1.Unknown())
					return reconcile.Result{Requeue: true}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
				}
			}

			if err := r.managed.RemoveFinalizer(ctx, managed); err != nil {
				// If this is the first time we encounter this issue we'll be
				// requeued implicitly when we update our status with the new error
				// condition. If not, we requeue explicitly, which will trigger
				// backoff.
				log.Debug("Cannot remove managed resource finalizer", "error", err)
				managed.SetConditions(nddv1.ReconcileError(err), nddv1.Unknown())
				return reconcile.Result{Requeue: true}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
			}

			// We've successfully deleted our external resource (if necessary) and
			// removed our finalizer. If we assume we were the only controller that
			// added a finalizer to this resource then it should no longer exist and
			// thus there is no point trying to update its status.
			// log.Debug("Successfully deleted managed resource")
			return reconcile.Result{Requeue: false}, nil
		}
		// Set validation status to unknown if the target is not found
		managed.SetConditions(nddv1.ParentValidationUnknown(), nddv1.LeafRefValidationUnknown(), nddv1.LeafRefValidationUnknown())
		// set empty target
		managed.SetTarget(make([]string, 0))
		// if the target was not found it means the network node is not defined or not in a status
		// to handle the reconciliation. A reconcilation retry will be triggered
		if strings.Contains(fmt.Sprintf("%s", err), "not found") ||
			strings.Contains(fmt.Sprintf("%s", err), "not configured") ||
			strings.Contains(fmt.Sprintf("%s", err), "not ready") {
			//log.Debug("network node not found")
			record.Event(managed, event.Warning(reasonCannotGetValidTarget, err))
			managed.SetConditions(nddv1.TargetNotFound(), nddv1.Unavailable(), nddv1.ReconcileSuccess())
			return reconcile.Result{RequeueAfter: shortWait}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
		}
		//log.Debug("network node error different from not found")
		// We'll usually hit this case if our Provider or its secret are missing
		// or invalid. If this is first time we encounter this issue we'll be
		// requeued implicitly when we update our status with the new error
		// condition. If not, we requeue explicitly, which will trigger
		// backoff.
		//log.Debug("Cannot connect to network node device driver", "error", err)
		record.Event(managed, event.Warning(reasonCannotConnect, err))
		managed.SetConditions(nddv1.ReconcileError(errors.Wrap(err, errReconcileConnect)), nddv1.Unavailable(), nddv1.TargetFound())
		return reconcile.Result{Requeue: true}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
	}

	defer external.Close()

	// given we can connect to the network node device driver, the target is found
	// update codition and update the status field
	managed.SetConditions(nddv1.TargetFound())
	managed.SetTarget(external.GetTarget())

	observation, err := external.Observe(externalCtx, managed)
	log.Debug("Observation", "Observation", observation, "Transaction", gvkresource.GetTransaction(managed))
	if err != nil {
		// We'll usually hit this case if our Provider credentials are invalid
		// or insufficient for observing the external resource type we're
		// concerned with. If this is the first time we encounter this issue
		// we'll be requeued implicitly when we update our status with the new
		// error condition. If not, we requeue explicitly, which will
		// trigger backoff.
		log.Debug("Cannot observe external resource", "error", err)
		record.Event(managed, event.Warning(reasonCannotObserve, err))
		managed.SetConditions(nddv1.ReconcileError(errors.Wrap(err, errReconcileObserve)), nddv1.Unknown())
		return reconcile.Result{Requeue: true}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
	}

	if !observation.Ready {
		// When the cache is initializing we should not reconcile, so it is better to wait a reconciliation loop before retrying
		log.Debug("External resource cache is not ready", "requeue-after", time.Now().Add(shortWait))
		managed.SetConditions(nddv1.Unavailable())
		return reconcile.Result{RequeueAfter: shortWait}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
	}

	if observation.Exhausted {
		// When the cache is initializing we should not reconcile, so it is better to wait a reconciliation loop before retrying
		log.Debug("External resource cache is exhausted", "requeue-after", time.Now().Add(mediumWait))
		managed.SetConditions(nddv1.Unavailable())
		return reconcile.Result{RequeueAfter: mediumWait}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
	}

	if meta.WasDeleted(managed) {
		// delete triggered
		log = log.WithValues("deletion-timestamp", managed.GetDeletionTimestamp())
		managed.SetConditions(nddv1.Deleting())

		// for transactions we dont delete if the status is pending since the delete is happening in the transaction
		// when the transaction delete succeeded the status will be Exists == false
		if gvkresource.GetTransaction(managed) != gvkresource.TransactionNone && observation.Pending {
			return reconcile.Result{RequeueAfter: veryShortWait}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
		}

		// if the resource has external leafref dependencies, we cannot delete the
		// resource
		hasOtherFinalizer, err := r.managed.HasOtherFinalizer(ctx, managed)
		if err != nil {
			log.Debug("Cannot get managed resource finalizer", "error", err)
			managed.SetConditions(nddv1.ReconcileError(err), nddv1.Unknown())
			return reconcile.Result{Requeue: true}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
		}
		// if other finalizer exists we cananot delete the resource and we have to requeue until the
		// resource get deleted
		if hasOtherFinalizer {
			return reconcile.Result{RequeueAfter: veryShortWait}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
		}

		// We'll only reach this point if deletion policy is not orphan, so we
		// are safe to call external deletion if external resource exists or the
		// resource has data
		if observation.Exists || observation.HasData {
			if err := external.Delete(externalCtx, managed); err != nil {
				// We'll hit this condition if we can't delete our external
				// resource, for example if our provider credentials don't have
				// access to delete it. If this is the first time we encounter
				// this issue we'll be requeued implicitly when we update our
				// status with the new error condition. If not, we want requeue
				// explicitly, which will trigger backoff.
				log.Debug("Cannot delete external resource", "error", err)
				record.Event(managed, event.Warning(reasonCannotDelete, err))
				managed.SetConditions(nddv1.ReconcileError(errors.Wrap(err, errReconcileDelete)), nddv1.Unknown())
				return reconcile.Result{Requeue: true}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
			}

			// We've successfully requested deletion of our external resource.
			// We queue another reconcile after a short wait rather than
			// immediately finalizing our delete in order to verify that the
			// external resource was actually deleted. If it no longer exists
			// we'll skip this block on the next reconcile and proceed to
			// unpublish and finalize. If it still exists we'll re-enter this
			// block and try again.
			// log.Debug("Successfully requested deletion of external resource")
			record.Event(managed, event.Normal(reasonDeleted, "Successfully requested deletion of external resource"))
			managed.SetConditions(nddv1.ReconcileSuccess())
			return reconcile.Result{RequeueAfter: veryShortWait}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
		}

		// remove the finalizer from the external resources before deleting the finalizer from the local resource
		for _, externalResourceName := range managed.GetExternalLeafRefs() {
			// for all external resource name that were resolved we need to remove the finalizer from the external resource
			if err := r.HandleExternalResourceFinalizer(ctx, FinalizerOperationRemove, externalResourceName, managed); err != nil {
				managed.SetConditions(nddv1.ReconcileError(err), nddv1.Unknown())
				return reconcile.Result{Requeue: true}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
			}
		}

		if err := r.managed.RemoveFinalizer(ctx, managed); err != nil {
			// If this is the first time we encounter this issue we'll be
			// requeued implicitly when we update our status with the new error
			// condition. If not, we requeue explicitly, which will trigger
			// backoff.
			log.Debug("Cannot remove managed resource finalizer", "error", err)
			managed.SetConditions(nddv1.ReconcileError(err), nddv1.Unknown())
			return reconcile.Result{Requeue: true}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
		}

		// We've successfully deleted our external resource (if necessary) and
		// removed our finalizer. If we assume we were the only controller that
		// added a finalizer to this resource then it should no longer exist and
		// thus there is no point trying to update its status.
		// log.Debug("Successfully deleted managed resource")
		return reconcile.Result{Requeue: false}, nil
	}

	// this should be handled after the delete check otherwise if a resource was in pending state it would never be executed
	if observation.Exists && observation.Pending {
		//Action was not yet executed so there is no point in doing further reconciliation
		log.Debug("Action is not yet executed", "requeue-after", time.Now().Add(veryShortWait))
		managed.SetConditions(nddv1.Unavailable())
		return reconcile.Result{RequeueAfter: veryShortWait}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
	}

	if observation.Failed {
		// The resource was not successfully applied to the device, the spec should change to retry
		log.Debug("External resource cache failed", "requeue-after", time.Now().Add(shortWait))
		managed.SetConditions(nddv1.Failed())
		return reconcile.Result{RequeueAfter: shortWait}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
	}

	// for transactions we dont do validations
	// when the resource is up to date we dont do validations
	if gvkresource.GetTransaction(managed) == gvkresource.TransactionNone && !observation.IsUpToDate {
		// get the full configuration of the network node in order to do leafref and parent validation

		log.Debug("Validation", "observation", observation, "transaction", gvkresource.GetTransaction(managed))
		cfg, err := external.GetConfig(externalCtx, managed)
		if err != nil {
			log.Debug("Cannot get network node configuration", "error", err)
			record.Event(managed, event.Warning(reasonCannotGetConfig, err))
			managed.SetConditions(nddv1.ReconcileError(errors.Wrap(err, errReconcileGetConfig)), nddv1.Unknown())
			return reconcile.Result{Requeue: true}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
		}

		parentDependencyObservation, err := r.validator.ValidateParentDependency(ctx, managed, cfg)
		if err != nil {
			if observation.Exists {
				if err := external.Delete(externalCtx, managed); err != nil {
					// We'll hit this condition if we can't delete our external resource
					log.Debug("Cannot delete external resource", "error", err)
					record.Event(managed, event.Warning(reasonCannotDelete, err))
					managed.SetConditions(nddv1.ReconcileError(errors.Wrap(err, errReconcileDelete)), nddv1.Unknown())
					return reconcile.Result{Requeue: true}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
				}
			}
			log.Debug("Cannot validate parent dependency", "error", err)
			record.Event(managed, event.Warning(reasonCannotValidateParentDependency, err))
			managed.SetConditions(nddv1.ReconcileError(errors.Wrap(err, errReconcileValidateParentDependency)), nddv1.Unknown())
			return reconcile.Result{Requeue: true}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
		}
		if !parentDependencyObservation.Success {
			if observation.Exists {
				if err := external.Delete(externalCtx, managed); err != nil {
					// We'll hit this condition if we can't delete our external resource
					log.Debug("Cannot delete external resource", "error", err)
					record.Event(managed, event.Warning(reasonCannotDelete, err))
					managed.SetConditions(nddv1.ReconcileError(errors.Wrap(err, errReconcileDelete)), nddv1.Unknown())
					return reconcile.Result{Requeue: true}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
				}
			}
			log.Debug("parent dependency validation failed", "error", errors.New("validation failed"))
			record.Event(managed, event.Warning(reasonValidateParentDependencyFailed, errors.New("validation failed")))
			managed.SetConditions(nddv1.ParentValidationFailure(), nddv1.Unavailable(), nddv1.ReconcileSuccess())
			return reconcile.Result{RequeueAfter: veryShortWait}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
		}
		managed.SetConditions(nddv1.ParentValidationSuccess())

		leafrefObservation, err := r.validator.ValidateLeafRef(ctx, managed, cfg)
		if err != nil {
			// TBD it can be that a change results in a deletion but the resource itself is depending on another leafref
			// so this will fail
			// TBD COMMENTED FOR NOW, BUT THIS WOULD ONLY HAPPEN WHEN WE HAVE A DEPENDENCY WITH AN UMR ABD IT GOT DELETED
			/*
				if observation.ResourceExists {

					if err := external.Delete(externalCtx, managed); err != nil {
						// We'll hit this condition if we can't delete our external resource
						log.Debug("Cannot delete external resource", "error", err)
						record.Event(managed, event.Warning(reasonCannotDelete, err))
						managed.SetConditions(nddv1.ReconcileError(errors.Wrap(err, errReconcileDelete)), nddv1.Unknown())
						return reconcile.Result{Requeue: true}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
					}

				}
			*/
			log.Debug("Cannot validate external leafref", "error", err)
			record.Event(managed, event.Warning(reasonCannotValidateLeafRef, err))
			managed.SetConditions(nddv1.ReconcileError(errors.Wrap(err, errReconcileValidateLeafRef)), nddv1.Unknown())
			return reconcile.Result{Requeue: true}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
		}
		if !leafrefObservation.Success {
			// TBD COMMENTED FOR NOW, BUT THIS WOULD ONLY HAPPEN WHEN WE HAVE A DEPENDENCY WITH AN UMR ABD IT GOT DELETED
			/*
				if observation.ResourceExists {
						if err := external.Delete(externalCtx, managed); err != nil {
							// We'll hit this condition if we can't delete our external resource
							log.Debug("Cannot delete external resource", "error", err)
							record.Event(managed, event.Warning(reasonCannotDelete, err))
							managed.SetConditions(nddv1.ReconcileError(errors.Wrap(err, errReconcileDelete)), nddv1.Unknown())
							return reconcile.Result{Requeue: true}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
						}

				}
			*/
			log.Debug("external leafref validation failed", "error", errors.New("validation failed"))
			record.Event(managed, event.Warning(reasonValidateLeafRefFailed, errors.New("validation failed")))
			managed.SetConditions(nddv1.LeafRefValidationFailure(), nddv1.Unavailable(), nddv1.ReconcileSuccess())
			return reconcile.Result{RequeueAfter: veryShortWait}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
		}

		//log.Debug("External Leafref Validation", "resolved leafref", externalLeafrefObservation.ResolvedLeafRefs)
		// get the external resources that match the external leafRefs
		externalResourceNames := make([]string, 0)
		for _, resolvedLeafRef := range leafrefObservation.ResolvedLeafRefs {
			// the external finalizer handling is only required for external leafrefs
			if resolvedLeafRef.External {
				//log.WithValues("resolvedLeafRef", resolvedLeafRef)

				// for some special cases it might be that the remotePaths or multiple iso 1.
				// E.g. interface/subinterface or tunnel/vxlan-interface would be split in 2 paths
				remotePaths := yparser.GetRemotePathsFromResolvedLeafRef(resolvedLeafRef)

				for _, remotePath := range remotePaths {
					// get the resourceName from the device driver that matches the remotePath in the resolved LeafaRef
					externalResourceName, err := external.GetResourceName(ctx, managed, remotePath)
					if err != nil {
						log.Debug("Cannot get resource name", "error", err)
						record.Event(managed, event.Warning(reasonCannotGetResourceName, err))
						managed.SetConditions(nddv1.ReconcileError(errors.Wrap(err, errReconcileGetResourceName)), nddv1.Unknown())
						return reconcile.Result{Requeue: true}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
					}
					//log.Debug("External resource Name", "externalResourceName", externalResourceName, "remotePath", r.parser.GnmiPathToXPath(remotePath, true))
					// only append unique externalResourceName if the external resource is managed by the ndd provider
					if externalResourceName != "" {
						found := false
						for _, extResName := range externalResourceNames {
							// only append the externalResourceName for objects that are managed by the provider
							if extResName == externalResourceName {
								found = true
							}
						}
						if !found {
							externalResourceNames = append(externalResourceNames, externalResourceName)
						}
					} else {
						log.Debug("this is an external leafref of an umanaged resource of ndd, deletion of the remote leafRef will fail",
							"path", r.parser.GnmiPathToXPath(resolvedLeafRef.RemotePath, true))
					}
				}
			}
		}

		//log.Debug("leafref Validation", "externalResourceNames", externalResourceNames)
		for _, externalResourceName := range externalResourceNames {
			// for all external resolved names we need to add a finalizer to the external resource
			if err := r.HandleExternalResourceFinalizer(ctx, FinalizerOperationAdd, externalResourceName, managed); err != nil {
				managed.SetConditions(nddv1.ReconcileError(err), nddv1.Unknown())
				return reconcile.Result{Requeue: true}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
			}
		}
		//log.Debug("External Leafref Validation", "externalResourceNames", externalResourceNames)

		managed.SetConditions(nddv1.LeafRefValidationSuccess())
		// we need to check the delta between the existing External LeafRefs and the new LeaRefs
		// if there is a deletion we need remove the external finalizer of the object
		externalResourceNamesDeletes := FindExternalResourceNameDelta(managed.GetExternalLeafRefs(), externalResourceNames)
		for _, externalResourceName := range externalResourceNamesDeletes {
			// for all external resource name that are no longer resolved we need to remove the finalizer from the external resource
			if err := r.HandleExternalResourceFinalizer(ctx, FinalizerOperationRemove, externalResourceName, managed); err != nil {
				managed.SetConditions(nddv1.ReconcileError(err), nddv1.Unknown())
				return reconcile.Result{Requeue: true}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
			}
		}
		// Update the External LeafRefs in the Status field
		managed.SetExternalLeafRefs(externalResourceNames)

		// Validate if the resourceIndexes changed or not
		// if they changed we should delete the original resource
		resourceIndexesObservation, err := r.validator.ValidateResourceIndexes(ctx, managed)
		if err != nil {
			log.Debug("Cannot validate resource indexes", "error", err)
			record.Event(managed, event.Warning(reasonCannotValidateResourceIndexes, err))
			managed.SetConditions(nddv1.ReconcileError(errors.Wrap(err, errReconcileValidateResourceIndexes)), nddv1.Unknown())
			return reconcile.Result{Requeue: true}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
		}
		// if the resource indexes got deleted we need to delete the resources
		if resourceIndexesObservation.Changed {
			// we reuse the observation object
			observation := ExternalObservation{
				Deletes: resourceIndexesObservation.ResourceDeletes,
				Updates: make([]*gnmi.Update, 0), // no updates are necessary here
			}
			if err := external.Update(externalCtx, managed, observation); err != nil {
				// We'll hit this condition if we can't update our external resource,
				log.Debug("Cannot update external resource")
				record.Event(managed, event.Warning(reasonCannotUpdate, err))
				managed.SetConditions(nddv1.ReconcileError(errors.Wrap(err, errReconcileUpdate)), nddv1.Unknown())
				return reconcile.Result{Requeue: true}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
			}
		}
		// update the resourceIndexes, so we can compare the new data in the next reconcile
		managed.SetResourceIndexes(resourceIndexesObservation.ResourceIndexes)

		if err := r.managed.AddFinalizer(ctx, managed); err != nil {
			// If this is the first time we encounter this issue we'll be requeued
			// implicitly when we update our status with the new error condition. If
			// not, we requeue explicitly, which will trigger backoff.
			log.Debug("Cannot add finalizer", "error", err)
			managed.SetConditions(nddv1.ReconcileError(err), nddv1.Unknown())
			return reconcile.Result{Requeue: true}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
		}
	}

	//log.Debug("Observation", "observation", observation)
	if !observation.Exists {
		// if we go from an umnaged resource to a managed resource we can have dangling objects
		// which we have to clean
		// TBD IF WE NEED THIS SINCE THE OBSERVE DIFF SHOULD HANDLE THIS
		/*
			if observation.ResourceHasData {
				// TODO add logic for autopilot and non autopilot mode of operation

				// if we are in auto-pilot mode we should align the data and delete the objects
				// if they exist to align the resource with the intended data


				if len(observation.ResourceDeletes) != 0 {
					// remove the updates from the observation since they will get created
					// when we create the resource
					observation.ResourceUpdates = make([]*gnmi.Update, 0)
					if err := external.Update(externalCtx, managed, observation); err != nil {
						// We'll hit this condition if we can't update our external resource
						log.Debug("Cannot update external resource")
						record.Event(managed, event.Warning(reasonCannotUpdate, err))
						managed.SetConditions(nddv1.ReconcileError(errors.Wrap(err, errReconcileUpdate)), nddv1.Unknown())
						return reconcile.Result{Requeue: true}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
					}
				}
				// if we are not in auto-pilot mode we should create the object
			}
		*/

		if err := external.Create(externalCtx, managed, false); err != nil {
			// We'll hit this condition if the grpc connection fails.
			// If this is the first time we encounter this
			// issue we'll be requeued implicitly when we update our status with
			// the new error condition. If not, we requeue explicitly, which will trigger backoff.
			log.Debug("Cannot create external resource", "error", err)
			record.Event(managed, event.Warning(reasonCannotCreate, err))
			managed.SetConditions(nddv1.ReconcileError(errors.Wrap(err, errReconcileCreate)), nddv1.Unknown())
			return reconcile.Result{Requeue: true}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
		}
		managed.SetConditions(nddv1.Creating())

		// We've successfully created our external resource. In many cases the
		// creation process takes a little time to finish. We requeue explicitly
		// order to observe the external resource to determine whether it's
		// ready for use.
		//log.Debug("Successfully requested creation of external resource")
		record.Event(managed, event.Normal(reasonCreated, "Successfully requested creation of external resource"))
		managed.SetConditions(nddv1.ReconcileSuccess())
		return reconcile.Result{RequeueAfter: veryShortWait}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
	}
	// resource exists
	if !observation.HasData {
		// the resource got deleted, so we need to recreate the resource
		if err := external.Create(externalCtx, managed, true); err != nil {
			// We'll hit this condition if the grpc connection fails.
			// If this is the first time we encounter this
			// issue we'll be requeued implicitly when we update our status with
			// the new error condition. If not, we requeue explicitly, which will trigger backoff.
			log.Debug("Cannot create external resource", "error", err)
			record.Event(managed, event.Warning(reasonCannotCreate, err))
			managed.SetConditions(nddv1.ReconcileError(errors.Wrap(err, errReconcileCreate)), nddv1.Unknown())
			return reconcile.Result{Requeue: true}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
		}
		managed.SetConditions(nddv1.Creating())

		// We've successfully created our external resource. In many cases the
		// creation process takes a little time to finish. We requeue explicitly
		// order to observe the external resource to determine whether it's
		// ready for use.
		//log.Debug("Successfully requested creation of external resource")
		record.Event(managed, event.Normal(reasonCreated, "Successfully requested creation of external resource"))
		managed.SetConditions(nddv1.ReconcileSuccess())
		return reconcile.Result{RequeueAfter: veryShortWait}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)

	}

	if observation.IsUpToDate {
		// We did not need to create, update, or delete our external resource.
		// Per the below issue nothing will notify us if and when the external
		// resource we manage changes, so we requeue a speculative reconcile
		// after the specified poll interval in order to observe it and react
		// accordingly.
		log.Debug("External resource is up to date", "requeue-after", time.Now().Add(r.pollInterval))
		managed.SetConditions(nddv1.ReconcileSuccess(), nddv1.Available(), nddv1.LeafRefValidationSuccess(), nddv1.ParentValidationSuccess())
		return reconcile.Result{RequeueAfter: r.pollInterval}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
	}

	if err := external.Update(externalCtx, managed, observation); err != nil {
		// We'll hit this condition if we can't update our external resource,
		log.Debug("Cannot update external resource")
		record.Event(managed, event.Warning(reasonCannotUpdate, err))
		managed.SetConditions(nddv1.ReconcileError(errors.Wrap(err, errReconcileUpdate)), nddv1.Unknown())
		return reconcile.Result{Requeue: true}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
	}

	// We've successfully updated our external resource. Per the below issue
	// nothing will notify us if and when the external resource we manage
	// changes, so we requeue a speculative reconcile after the specified poll
	// interval in order to observe it and react accordingly.
	//log.Debug("Successfully requested update of external resource", "requeue-after", time.Now().Add(veryShortWait))
	record.Event(managed, event.Normal(reasonUpdated, "Successfully requested update of external resource"))
	managed.SetConditions(nddv1.ReconcileSuccess(), nddv1.Updating())
	return reconcile.Result{RequeueAfter: veryShortWait}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
}

func FindExternalResourceNameDelta(oldSlice, newSlice []string) []string {
	// we run over the old Slice and see if all the elements exists
	//in the new Slice if not we need to add the element to the slice
	returnSlice := make([]string, 0)
	for _, oldName := range oldSlice {
		found := false
		for _, newName := range newSlice {
			if oldName == newName {
				found = true
				break
			}
		}
		if !found {
			returnSlice = append(returnSlice, oldName)
		}
	}
	return returnSlice
}

// A OperationType represents an operatio on a JSON resource
type FinalizerOperation string

// Condition Kinds.
const (
	// add
	FinalizerOperationAdd FinalizerOperation = "Add"
	// remove
	FinalizerOperationRemove FinalizerOperation = "Remove"
)

func (r *Reconciler) HandleExternalResourceFinalizer(ctx context.Context, operation FinalizerOperation, externalResourceName string, managed resource.Managed) error {
	gvk, err := gvkresource.String2Gvk(externalResourceName)
	if err != nil {
		r.log.Debug("Cannot get gvk", "error", err, "externalResourceName", externalResourceName, "gvk", gvk)
		return errors.Wrap(err, fmt.Sprintf("Cannot get gvk with externalResourceName %s, gvk: %v", externalResourceName, gvk))
	}

	emr := &unstructured.Unstructured{}
	emr.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   gvk.Group,
		Kind:    gvk.Kind,
		Version: gvk.Version,
	})

	key := types.NamespacedName{
		Namespace: gvk.NameSpace,
		Name:      gvk.Name,
	}
	if err := r.client.Get(ctx, key, emr); err != nil {
		r.log.Debug("Cannot get external resource", "error", err, "externalResourceName", externalResourceName, "gvk", gvk)
		return errors.Wrap(err, fmt.Sprintf("Cannot get external resource with externalResourceName %s, gvk: %v", externalResourceName, gvk))

	}
	switch operation {
	case FinalizerOperationAdd:
		if err := r.managed.AddFinalizerString(ctx, emr, managed.GetName()); err != nil {
			r.log.Debug("Cannot remove finalizer from external resource", "error", err, "externalResourceName", externalResourceName, "gvk", gvk)
			return errors.Wrap(err, fmt.Sprintf("Cannot remove finalizer from external resource with externalResourceName %s, gvk: %v", externalResourceName, gvk))
		}
	case FinalizerOperationRemove:
		if err := r.managed.RemoveFinalizerString(ctx, emr, managed.GetName()); err != nil {
			r.log.Debug("Cannot remove finalizer from external resource", "error", err, "externalResourceName", externalResourceName, "gvk", gvk)
			return errors.Wrap(err, fmt.Sprintf("Cannot remove finalizer from external resource with externalResourceName %s, gvk: %v", externalResourceName, gvk))
		}
	}
	return nil
}
