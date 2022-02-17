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
	"strings"
	"time"

	nddv1 "github.com/yndd/ndd-runtime/apis/common/v1"
	systemv1alpha1 "github.com/yndd/nddp-system/apis/system/v1alpha1"
	"github.com/yndd/nddp-system/pkg/gvkresource"

	"github.com/pkg/errors"
	"github.com/yndd/ndd-runtime/pkg/event"
	"github.com/yndd/ndd-runtime/pkg/logging"
	"github.com/yndd/ndd-runtime/pkg/meta"
	"github.com/yndd/ndd-runtime/pkg/resource"
	"github.com/yndd/ndd-runtime/pkg/tresource"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	// timers
	transactionFinalizerName = "finalizer.transaction.ndd.yndd.io"
	reconcileGracePeriod     = 30 * time.Second
	reconcileTimeout         = 1 * time.Minute
	shortWait                = 30 * time.Second
	mediumWait               = 1 * time.Minute
	veryShortWait            = 1 * time.Second
	longWait                 = 1 * time.Minute

	defaultpollInterval = 1 * time.Minute

	// errors
	errGetTransaction               = "cannot get transaction resource"
	errUpdateTransactionAfterCreate = "cannot update transaction resource. this may have resulted in a leaked external resource"
	errReconcileConnect             = "connect failed"
	errReconcileObserve             = "observe failed"
	errReconcileCreate              = "create failed"
	errReconcileDelete              = "delete failed"
	errUpdateTransactionStatus      = "cannot update status of the transaction resource"

	// Event reasons.
	reasonCannotConnect           event.Reason = "CannotConnectToProvider"
	reasonCannotInitialize        event.Reason = "CannotInitializeManagedResource"
	reasonCannotResolveRefs       event.Reason = "CannotResolveResourceReferences"
	reasonCannotObserve           event.Reason = "CannotObserveExternalResource"
	reasonCannotCreate            event.Reason = "CannotCreateExternalResource"
	reasonCannotDelete            event.Reason = "CannotDeleteExternalResource"
	reasonCannotPublish           event.Reason = "CannotPublishConnectionDetails"
	reasonCannotUnpublish         event.Reason = "CannotUnpublishConnectionDetails"
	reasonCannotUpdate            event.Reason = "CannotUpdateExternalResource"
	reasonCannotUpdateTransaction event.Reason = "CannotUpdateTransactionResource"
	reasonCannotGetResources      event.Reason = "CannotGetResources"
	reasonConvertGvk              event.Reason = "CannotConvertGvk"

	reasonDeleted event.Reason = "DeletedTransactionResource"
	reasonCreated event.Reason = "CreatedTransactionResource"
	reasonUpdated event.Reason = "UpdatedTransactionResource"
	reasonSuccess event.Reason = "TransactionSucceeded"
	reasonFailed  event.Reason = "TransactionFailed"
)

// ControllerName returns the recommended name for controllers that use this
// package to reconcile a particular kind of managed resource.
func ControllerName(kind string) string {
	return "transaction/" + strings.ToLower(kind)
}

// A Reconciler reconciles managed resources by creating and managing the
// lifecycle of an external resource, i.e. a resource in an external network
// device through an API. Each controller must watch the managed resource kind
// for which it is responsible.
type Reconciler struct {
	client         client.Client
	newTransaction func() tresource.Transaction

	pollInterval time.Duration
	timeout      time.Duration

	// The below structs embed the set of interfaces used to implement the
	// managed resource reconciler. We do this primarily for readability, so
	// that the reconciler logic reads r.external.Connect(),
	// r.managed.Delete(), etc.
	external    trExternal
	transaction trTransaction
	handler     trHandler

	log    logging.Logger
	record event.Recorder
}

type trTransaction struct {
	tresource.Finalizer
}

func defaultTRTransaction(m manager.Manager) trTransaction {
	return trTransaction{
		Finalizer: tresource.NewAPIFinalizer(m.GetClient(), transactionFinalizerName),
	}
}

type trHandler struct {
	Handler
}

func defaultTRHandler() trHandler {
	return trHandler{
		Handler: &NopHandler{},
	}
}

type trExternal struct {
	ExternalConnecter
}

func defaultTRExternal() trExternal {
	return trExternal{
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

// WithFinalizer specifies how the Reconciler should add and remove
// finalizers to and from the managed resource.
func WithFinalizer(f tresource.Finalizer) ReconcilerOption {
	return func(r *Reconciler) {
		r.transaction.Finalizer = f
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

func WithHandler(h Handler) ReconcilerOption {
	return func(r *Reconciler) {
		r.handler.Handler = h
	}
}

// NewReconciler returns a Reconciler that reconciles managed resources of the
// supplied ManagedKind with resources in an external network device.
// It panics if asked to reconcile a managed resource kind that is
// not registered with the supplied manager's runtime.Scheme. The returned
// Reconciler reconciles with a dummy, no-op 'external system' by default;
// callers should supply an ExternalConnector that returns an ExternalClient
// capable of managing resources in a real system.
func NewReconciler(m manager.Manager, of tresource.TransactionKind, o ...ReconcilerOption) *Reconciler {
	nt := func() tresource.Transaction {
		return resource.MustCreateObject(schema.GroupVersionKind(of), m.GetScheme()).(tresource.Transaction)
	}

	// Panic early if we've been asked to reconcile a resource kind that has not
	// been registered with our controller manager's scheme.
	_ = nt()

	r := &Reconciler{
		client:         m.GetClient(),
		newTransaction: nt,
		pollInterval:   defaultpollInterval,
		timeout:        reconcileTimeout,
		transaction:    defaultTRTransaction(m),
		external:       defaultTRExternal(),
		handler:        defaultTRHandler(),
		log:            logging.NewNopLogger(),
		record:         event.NewNopRecorder(),
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

	tr := r.newTransaction()
	if err := r.client.Get(ctx, req.NamespacedName, tr); err != nil {
		// There's no need to requeue if we no longer exist. Otherwise we'll be
		// requeued implicitly because we return an error.
		log.Debug("Cannot get transaction resource", "error", err)
		return reconcile.Result{}, errors.Wrap(resource.IgnoreNotFound(err), errGetTransaction)
	}

	record := r.record.WithAnnotations("external-name", meta.GetExternalName(tr))
	log = log.WithValues(
		"uid", tr.GetUID(),
		"version", tr.GetResourceVersion(),
	)

	if err := r.transaction.AddFinalizer(ctx, tr); err != nil {
		// If this is the first time we encounter this issue we'll be requeued
		// implicitly when we update our status with the new error condition. If
		// not, we requeue explicitly, which will trigger backoff.
		log.Debug("Cannot add finalizer", "error", err)
		tr.SetConditions(nddv1.ReconcileError(err), nddv1.Unknown())
		return reconcile.Result{Requeue: true}, errors.Wrap(r.client.Status().Update(ctx, tr), errUpdateTransactionStatus)
	}

	resources, err := r.handler.GetResources(ctx, tr)
	if err != nil {
		record.Event(tr, event.Warning(reasonCannotGetResources, err))
		tr.SetConditions(nddv1.ReconcileError(errors.Wrap(err, errReconcileConnect)), nddv1.Unavailable())
		return reconcile.Result{Requeue: true}, errors.Wrap(r.client.Status().Update(ctx, tr), errUpdateTransactionStatus)
	}

	log.Debug("Transaction", "resources", resources)

	// validate if the transaction is complete in the cache
	// we walk over the system resource cache and if the transaction macthes we delete the entries from the k8s api resourcelist
	// if the transaction is complete we should have no entries any longer in the k8s resourceList
	allDeviceTransactionsCompleted := true
	reconcileError := false
	success := true
	for deviceName, deviceCrs := range resources {
		// add devicename to the logger
		log.WithValues("device", deviceName)

		external, err := r.external.Connect(externalCtx, tr, deviceName)
		if err != nil {
			record.Event(tr, event.Warning(reasonCannotConnect, err))
			tr.SetDeviceConditions(deviceName, nddv1.ReconcileError(errors.Wrap(err, errReconcileConnect)), nddv1.Unavailable())
			// we stop now and continue to the next device if it is available
			allDeviceTransactionsCompleted = false
			reconcileError = true
			continue
		}

		defer external.Close()

		observation, err := external.Observe(externalCtx, tr)
		log.Debug("Observation", "Observation", observation)
		if err != nil {
			// We'll usually hit this case if our Provider credentials are invalid
			// or insufficient for observing the external resource type we're
			// concerned with. If this is the first time we encounter this issue
			// we'll be requeued implicitly when we update our status with the new
			// error condition. If not, we requeue explicitly, which will
			// trigger backoff.
			log.Debug("Cannot observe cache", "error", err)
			record.Event(tr, event.Warning(reasonCannotObserve, err))
			tr.SetDeviceConditions(deviceName, nddv1.ReconcileError(errors.Wrap(err, errReconcileConnect)), nddv1.Unknown())
			// we stop now and continue to the next device if it is available
			allDeviceTransactionsCompleted = false
			reconcileError = true
			continue
		}

		if observation.Exhausted {
			// When the cache is initializing we should not reconcile, so it is better to wait a reconciliation loop before retrying
			log.Debug("External resource cache is exhausted")
			tr.SetDeviceConditions(deviceName, nddv1.Unavailable())
			// we stop now and continue to the next device if it is available
			allDeviceTransactionsCompleted = false
			continue
		}

		if !observation.Ready {
			// When the cache is initializing we should not reconcile, so it is better to wait a reconciliation loop before retrying
			log.Debug("External resource cache is not ready")
			tr.SetDeviceConditions(deviceName, nddv1.Unavailable())
			// we stop now and continue to the next device if it is available
			allDeviceTransactionsCompleted = false
			continue
		}

		if observation.Pending {
			//Action was not yet executed so there is no point in doing further reconciliation
			// we dont update the status
			// we stop now and continue to the next device if it is available
			allDeviceTransactionsCompleted = false
			continue
		}

		// validate if all resources of the transaction are present in the cache
		complete, gvkList, err := r.validateResourcesPerTransactionInCache(tr, observation.GvkResourceList, deviceName, deviceCrs)
		if err != nil {
			allDeviceTransactionsCompleted = false
			continue
		}
		log.Debug("transaction complete", "complete", complete)
		if !complete {
			allDeviceTransactionsCompleted = false
			continue
		}

		if meta.WasDeleted(tr) {
			// delete triggered
			log.WithValues("deletion-timestamp", tr.GetDeletionTimestamp())
			tr.SetDeviceConditions(deviceName, nddv1.Deleting())

			// Delete the transaction from the cache
			if observation.Exists {
				if err := external.Delete(externalCtx, tr, gvkList); err != nil {
					// We'll hit this condition if we can't delete our external
					// resource, for example if our provider credentials don't have
					// access to delete it. If this is the first time we encounter
					// this issue we'll be requeued implicitly when we update our
					// status with the new error condition. If not, we want requeue
					// explicitly, which will trigger backoff.
					log.Debug("Cannot delete external resource", "error", err)
					record.Event(tr, event.Warning(reasonCannotDelete, err))
					tr.SetDeviceConditions(deviceName, nddv1.ReconcileError(errors.Wrap(err, errReconcileConnect)), nddv1.Unknown())
					// we stop now and continue to the next device if it is available
					allDeviceTransactionsCompleted = false
					reconcileError = true
					continue
				}

				// We've successfully requested deletion of our external resource.
				// We queue another reconcile after a short wait rather than
				// immediately finalizing our delete in order to verify that the
				// external resource was actually deleted. If it no longer exists
				// we'll skip this block on the next reconcile and proceed to
				// unpublish and finalize. If it still exists we'll re-enter this
				// block and try again.
				// log.Debug("Successfully requested deletion of external resource")
				record.Event(tr, event.Normal(reasonDeleted, "Successfully requested deletion of external resource"))
				tr.SetDeviceConditions(deviceName, nddv1.ReconcileSuccess())
			}
			continue
			// we are done for this device
		}
		//log.Debug("Observation", "observation", observation)
		if !observation.Exists {
			if err := external.Create(externalCtx, tr, gvkList); err != nil {
				// We'll hit this condition if the grpc connection fails.
				// If this is the first time we encounter this
				// issue we'll be requeued implicitly when we update our status with
				// the new error condition. If not, we requeue explicitly, which will trigger backoff.
				log.Debug("Cannot create external resource", "error", err)
				record.Event(tr, event.Warning(reasonCannotCreate, err))
				tr.SetDeviceConditions(deviceName, nddv1.ReconcileError(errors.Wrap(err, errReconcileConnect)), nddv1.Unknown())
				// we stop now and continue to the next device if it is available
				allDeviceTransactionsCompleted = false
				reconcileError = true
				continue
			}

			// We've successfully created our external resource. In many cases the
			// creation process takes a little time to finish. We requeue explicitly
			// order to observe the external resource to determine whether it's
			// ready for use.
			//log.Debug("Successfully requested creation of external resource")
			record.Event(tr, event.Normal(reasonCreated, "Successfully requested creation of external resource"))
			tr.SetDeviceConditions(deviceName, nddv1.ReconcileSuccess())
			continue
		}
		// Failed -> stop
		if !observation.Success {
			// The resource was not successfully applied to the device, the spec should change to retry
			log.Debug("Device transaction failed")
			record.Event(tr, event.Normal(reasonFailed, "Device Transaction failed"))
			tr.SetDeviceConditions(deviceName, nddv1.ReconcileSuccess(), nddv1.Failed())
			continue
		}
		// success -> stop
		//log.Debug("Successfully requested update of external resource", "requeue-after", time.Now().Add(veryShortWait))
		record.Event(tr, event.Normal(reasonSuccess, "Device Transaction succeeded"))
		log.Debug("Device transaction succeeded")
		tr.SetDeviceConditions(deviceName, nddv1.ReconcileSuccess(), nddv1.Available())
		continue
	}

	if reconcileError {
		tr.SetConditions(nddv1.ReconcileError(errors.Wrap(err, errReconcileConnect)), nddv1.Unavailable())
		return reconcile.Result{RequeueAfter: shortWait}, errors.Wrap(r.client.Status().Update(ctx, tr), errUpdateTransactionStatus)
	}

	if !allDeviceTransactionsCompleted {
		tr.SetConditions(nddv1.ReconcileSuccess(), nddv1.Pending())
		return reconcile.Result{RequeueAfter: veryShortWait}, errors.Wrap(r.client.Status().Update(ctx, tr), errUpdateTransactionStatus)
	}

	if meta.WasDeleted(tr) {
		if err := r.transaction.RemoveFinalizer(ctx, tr); err != nil {
			// If this is the first time we encounter this issue we'll be
			// requeued implicitly when we update our status with the new error
			// condition. If not, we requeue explicitly, which will trigger
			// backoff.
			log.Debug("Cannot remove managed resource finalizer", "error", err)
			tr.SetConditions(nddv1.ReconcileError(err), nddv1.Unknown())
			return reconcile.Result{Requeue: true}, errors.Wrap(r.client.Status().Update(ctx, tr), errUpdateTransactionStatus)
		}
		// We've successfully removed our finalizer. If we assume we were the only
		// controller that added a finalizer to this resource then it should no
		// longer exist and thus there is no point trying to update its status.
		// log.Debug("Successfully deleted managed resource")
		return reconcile.Result{Requeue: false}, nil
	}

	if !success {
		tr.SetConditions(nddv1.ReconcileSuccess(), nddv1.Failed())
		return reconcile.Result{}, errors.Wrap(r.client.Status().Update(ctx, tr), errUpdateTransactionStatus)
	}
	tr.SetConditions(nddv1.ReconcileSuccess(), nddv1.Available())
	return reconcile.Result{}, errors.Wrap(r.client.Status().Update(ctx, tr), errUpdateTransactionStatus)
}

func (r *Reconciler) validateResourcesPerTransactionInCache(tr tresource.Transaction, gvkresourceList []*systemv1alpha1.Gvk, deviceName string, deviceCrs map[string]map[string]interface{}) (bool, []string, error) {
	// initialize the gvkList per device
	gvkList := make([]string, 0)

	// if the gvkresourceList is empty there should not be a transaction so we return incomplete
	if len(gvkresourceList) == 0 {
		return false, gvkList, nil
	}

	for _, gvkResource := range gvkresourceList {
		gvk, err := gvkresource.String2Gvk(gvkResource.Name)
		if err != nil {
			r.log.Debug("Gvk Resource Failed", "error", err)
			return false, gvkList, err
		}
		r.log.Debug("Transaction information",
			"device", deviceName,
			"gvkResource.Transaction", gvkResource.Transaction,
			"transactionName", tr.GetName(),
			"gvkResource.Transactiongeneration", gvkResource.Transactiongeneration,
			"transactionOwnerGeneration", tr.GetOwnerGeneration(),
			"gvk.NameSpace", gvk.NameSpace,
			"transactionNamespace", tr.GetNamespace(),
		)
		if gvkResource.Transaction == tr.GetName() && gvkResource.Transactiongeneration == tr.GetOwnerGeneration() && gvk.NameSpace == tr.GetNamespace() {
			// check the kind
			if deviceCrNames, ok := deviceCrs[gvk.Kind]; ok {
				delete(deviceCrNames, gvk.Name)
				gvkList = append(gvkList, gvkResource.Name)
			}
		}
	}

	// when the transaction list is complete all the deviceCrPerKind entries should be 0
	for _, deviceCrPerKind := range deviceCrs {
		if len(deviceCrPerKind) != 0 {
			return false, gvkList, nil
		}
	}
	return true, gvkList, nil
}
