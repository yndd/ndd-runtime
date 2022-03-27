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

	nddv1 "github.com/yndd/ndd-runtime/apis/common/v1"
	"github.com/yndd/ndd-yang/pkg/parser"

	"github.com/pkg/errors"
	"github.com/yndd/ndd-runtime/pkg/event"
	"github.com/yndd/ndd-runtime/pkg/logging"
	"github.com/yndd/ndd-runtime/pkg/meta"
	"github.com/yndd/ndd-runtime/pkg/resource"
	"k8s.io/apimachinery/pkg/runtime/schema"
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
	errGetManaged                = "cannot get managed resource"
	errUpdateManagedAfterCreate  = "cannot update managed resource. this may have resulted in a leaked external resource"
	errReconcileConnect          = "connect failed"
	errReconcileObserve          = "observe failed"
	errReconcileCreate           = "create failed"
	errReconcileUpdate           = "update failed"
	errReconcileDelete           = "delete failed"
	errReconcileGetSystemConfig  = "get system device config failed"
	errReconcileGetRunningConfig = "get running device config failed"
	errReconcileValidateResource = "resource validation failed"
	errReconcileValidateConfig   = "config validation failed"
	errUpdateManagedStatus       = "cannot update status of the managed resource"

	// Event reasons.
	reasonCannotConnect          event.Reason = "CannotConnectToProvider"
	reasonCannotInitialize       event.Reason = "CannotInitializeManagedResource"
	reasonCannotResolveRefs      event.Reason = "CannotResolveResourceReferences"
	reasonCannotObserve          event.Reason = "CannotObserveExternalResource"
	reasonCannotCreate           event.Reason = "CannotCreateExternalResource"
	reasonCannotDelete           event.Reason = "CannotDeleteExternalResource"
	reasonCannotPublish          event.Reason = "CannotPublishConnectionDetails"
	reasonCannotUnpublish        event.Reason = "CannotUnpublishConnectionDetails"
	reasonCannotUpdate           event.Reason = "CannotUpdateExternalResource"
	reasonCannotUpdateManaged    event.Reason = "CannotUpdateManagedResource"
	reasonCannotGetSystemConfig  event.Reason = "CannotGetSystemConfig"
	reasonCannotGetRunningConfig event.Reason = "CannotGetRunningConfig"
	reasonCannotGetValideTarget  event.Reason = "CannotGetValideTarget"
	reasonCannotValidateResource event.Reason = "CannotValidateResource"
	reasonCannotValidateConfig   event.Reason = "CannotValidateConfig"
	reasonValidateResourceFailed event.Reason = "reasonValidateResourceFailed"

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
		managed.SetConditions(nddv1.RootPathValidationUnknown())
		// if the target was not found it means the network node is not defined or not in a status
		// to handle the reconciliation. A reconcilation retry will be triggered
		if strings.Contains(fmt.Sprintf("%s", err), "not found") ||
			strings.Contains(fmt.Sprintf("%s", err), "not configured") ||
			strings.Contains(fmt.Sprintf("%s", err), "not ready") {
			//log.Debug("network node not found")
			record.Event(managed, event.Warning(reasonCannotGetValideTarget, err))
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

	systemCfg, err := external.GetSystemConfig(externalCtx, managed)
	if err != nil {
		log.Debug("Cannot get resourceList", "error", err)
		record.Event(managed, event.Warning(reasonCannotGetSystemConfig, err))
		managed.SetConditions(nddv1.ReconcileError(errors.Wrap(err, errReconcileGetSystemConfig)), nddv1.Unknown())
		return reconcile.Result{Requeue: true}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
	}

	if systemCfg == nil {
		// When systemCfg is nil -> proxy cache is Not Ready
		// When the cache is initializing we should not reconcile, so it is better to wait a reconciliation loop before retrying
		log.Debug("External resource cache is not ready", "requeue-after", time.Now().Add(shortWait))
		managed.SetConditions(nddv1.Unavailable())
		return reconcile.Result{RequeueAfter: shortWait}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
	}

	validateResourceObservation, err := r.validator.ValidateResource(ctx, managed, systemCfg)
	if err != nil {
		log.Debug("Cannot validate rootPaths", "error", err)
		record.Event(managed, event.Warning(reasonCannotValidateResource, err))
		managed.SetConditions(nddv1.ReconcileError(errors.Wrap(err, errReconcileValidateResource)), nddv1.Unknown())
		return reconcile.Result{Requeue: true}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
	}
	log.Debug("validateResourceObservation", "observation", validateResourceObservation)

	if validateResourceObservation.Exhausted {
		// When the device is exhausted we should not reconcile, so it is better to wait a reconciliation loop before retrying
		log.Debug("External resource cache is exhausted", "requeue-after", time.Now().Add(mediumWait))
		managed.SetConditions(nddv1.Unavailable())
		return reconcile.Result{RequeueAfter: mediumWait}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
	}

	// we need to exclude this check when a resource is deleted, otherwise the delete would never be executed
	if !meta.WasDeleted(managed) && validateResourceObservation.Exists && validateResourceObservation.Pending {
		//Action was not yet executed so there is no point in doing further reconciliation
		log.Debug("Action is not yet executed", "requeue-after", time.Now().Add(veryShortWait))
		managed.SetConditions(nddv1.Unavailable())
		return reconcile.Result{RequeueAfter: veryShortWait}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
	}

	// we need to exclude this check when a resource is deleted
	if !meta.WasDeleted(managed) && validateResourceObservation.Failed {
		// The resource was not successfully applied to the device, the spec should change to retry
		log.Debug("External resource cache failed", "requeue-after", time.Now().Add(shortWait))
		managed.SetConditions(nddv1.Failed(validateResourceObservation.Message))
		return reconcile.Result{RequeueAfter: shortWait}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
	}

	runningCfg, err := external.GetRunningConfig(externalCtx, managed)
	if err != nil {
		log.Debug("Cannot get running", "error", err)
		record.Event(managed, event.Warning(reasonCannotGetRunningConfig, err))
		managed.SetConditions(nddv1.ReconcileError(errors.Wrap(err, errReconcileValidateConfig)), nddv1.Unknown())
		return reconcile.Result{Requeue: true}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
	}

	if runningCfg == nil {
		// When runningCfg is nil -> proxy cache is Not Ready
		// When the cache is initializing we should not reconcile, so it is better to wait a reconciliation loop before retrying
		log.Debug("External resource cache is not ready", "requeue-after", time.Now().Add(shortWait))
		managed.SetConditions(nddv1.Unavailable())
		return reconcile.Result{RequeueAfter: shortWait}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
	}

	validateConfigObservation, err := r.validator.ValidateConfig(ctx, managed, systemCfg, runningCfg)
	if err != nil {
		log.Debug("Cannot validate config", "error", err)
		record.Event(managed, event.Warning(reasonCannotValidateConfig, err))
		managed.SetConditions(nddv1.ReconcileError(errors.Wrap(err, errReconcileValidateConfig)), nddv1.Unknown())
		return reconcile.Result{Requeue: true}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
	}
	log.Debug("validateConfigObservation", "observation", validateConfigObservation)

	if !validateConfigObservation.ValidateSucces {
		// The resource was not successfully applied to the device, the spec should change to retry
		log.Debug("resourceValidation failed", "requeue-after", time.Now().Add(shortWait))
		managed.SetConditions(nddv1.Failed(validateConfigObservation.Message))
		return reconcile.Result{RequeueAfter: shortWait}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
	}

	// TODO if changed -> delete the paths that are no longer needed
	managed.SetRootPaths(validateConfigObservation.RootPaths)

	if meta.WasDeleted(managed) {
		// delete triggered
		log = log.WithValues("deletion-timestamp", managed.GetDeletionTimestamp())
		managed.SetConditions(nddv1.Deleting())

		// We'll only reach this point if deletion policy is not orphan, so we
		// are safe to call external deletion if external resource exists or the
		// resource has data
		if validateResourceObservation.Exists {
			if err := external.Delete(externalCtx, managed, ExternalObservation{}); err != nil {
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

	//log.Debug("Observation", "observation", observation)
	if !validateResourceObservation.Exists {
		// transaction was false
		if err := external.Create(externalCtx, managed, ExternalObservation{}); err != nil {
			// We'll hit this condition if the grpc connection fails.
			// If this is the first time we encounter this
			// issue we'll be requeued implicitly when we update our status with
			// the new error condition. If not, we requeue explicitly, which will trigger backoff.
			log.Debug("Cannot create external resource", "error", err)
			record.Event(managed, event.Warning(reasonCannotCreate, err))
			managed.SetConditions(nddv1.ReconcileError(errors.Wrap(err, errReconcileCreate)), nddv1.Unknown())
			return reconcile.Result{RequeueAfter: veryShortWait}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
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

	observation, err := external.Observe(externalCtx, managed, runningCfg)
	log.Debug("Observation", "Observation", observation)
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
	// resource exists
	if !observation.HasData {
		// the resource got deleted, so we need to recreate the resource
		// transaction was true
		if err := external.Create(externalCtx, managed, observation); err != nil {
			// We'll hit this condition if the grpc connection fails.
			// If this is the first time we encounter this
			// issue we'll be requeued implicitly when we update our status with
			// the new error condition. If not, we requeue explicitly, which will trigger backoff.
			log.Debug("Cannot create external resource", "error", err)
			record.Event(managed, event.Warning(reasonCannotCreate, err))
			managed.SetConditions(nddv1.ReconcileError(errors.Wrap(err, errReconcileCreate)), nddv1.Unknown())
			return reconcile.Result{RequeueAfter: veryShortWait}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
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

	if !observation.IsUpToDate {
		if err := external.Update(externalCtx, managed, observation); err != nil {
			// We'll hit this condition if we can't update our external resource,
			log.Debug("Cannot update external resource")
			record.Event(managed, event.Warning(reasonCannotUpdate, err))
			managed.SetConditions(nddv1.ReconcileError(errors.Wrap(err, errReconcileUpdate)), nddv1.Unknown())
			return reconcile.Result{Requeue: true}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
		}
		managed.SetConditions(nddv1.Updating())

		// We've successfully updated our external resource. We will
		// reconcile short to observe the result of the update
		record.Event(managed, event.Normal(reasonUpdated, "Successfully requested update of external resource"))
		managed.SetConditions(nddv1.ReconcileSuccess(), nddv1.Updating())
		return reconcile.Result{RequeueAfter: veryShortWait}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
	}

	// The reosuce is up to date, we reconcile after the pollInterval to
	// regularly validate if the MR is still up to date
	log.Debug("External resource is up to date", "requeue-after", time.Now().Add(r.pollInterval))
	managed.SetConditions(nddv1.ReconcileSuccess(), nddv1.Available(), nddv1.RootPathValidationSuccess())
	return reconcile.Result{RequeueAfter: r.pollInterval}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)

}
