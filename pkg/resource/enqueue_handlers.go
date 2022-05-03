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

package resource

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type adder interface {
	Add(item interface{})
}

// EnqueueRequestForTarget enqueues a reconcile.Request for a referenced
// Target.
type EnqueueRequestForTarget struct{}

// Create adds a NamespacedName for the supplied CreateEvent if its Object is a
// TargetReferencer.
func (e *EnqueueRequestForTarget) Create(evt event.CreateEvent, q workqueue.RateLimitingInterface) {
	addTarget(evt.Object, q)
}

// Update adds a NamespacedName for the supplied UpdateEvent if its Objects are
// a TargetReferencer.
func (e *EnqueueRequestForTarget) Update(evt event.UpdateEvent, q workqueue.RateLimitingInterface) {
	addTarget(evt.ObjectOld, q)
	addTarget(evt.ObjectNew, q)
}

// Delete adds a NamespacedName for the supplied DeleteEvent if its Object is a
// TargetReferencer.
func (e *EnqueueRequestForTarget) Delete(evt event.DeleteEvent, q workqueue.RateLimitingInterface) {
	addTarget(evt.Object, q)
}

// Generic adds a NamespacedName for the supplied GenericEvent if its Object is
// a TargetReferencer.
func (e *EnqueueRequestForTarget) Generic(evt event.GenericEvent, q workqueue.RateLimitingInterface) {
	addTarget(evt.Object, q)
}

func addTarget(obj runtime.Object, queue adder) {
	pcr, ok := obj.(RequiredTargetReferencer)
	if !ok {
		return
	}

	queue.Add(reconcile.Request{NamespacedName: types.NamespacedName{Name: pcr.GetTargetReference().Name}})
}
