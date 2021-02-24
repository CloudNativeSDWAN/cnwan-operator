// Copyright © 2021 Cisco
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// All rights reserved.

package controllers

import (
	"context"
	"sync"

	"k8s.io/api/discovery/v1beta1"
	discoveryv1beta1 "k8s.io/api/discovery/v1beta1"
	ktypes "k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const (
	countPodsLabelKey string = "operator.cnwan.io/pods-count"
	enableVal         string = "enabled"
	disableVal        string = "disabled"
)

type epsData struct {
	count int
	srv   string
}

// EndpointSliceReconciler reconciles a EndpointSlice object
type EndpointSliceReconciler struct {
	*BaseReconciler

	lock           sync.Mutex
	epsDataActions map[string]*epsData
	srvCounts      map[string]map[string]int
}

// +kubebuilder:rbac:groups=discovery.k8s.io,resources=endpointslice,verbs=get;list;watch;create;update;patch;delete

// Reconcile keeps track counts in the endpointslice length
func (r *EndpointSliceReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithName("EndpointSliceReconciler").WithValues("endpointslice", req.NamespacedName)

	// TODO: implement me

	return ctrl.Result{}, nil
}

// SetupWithManager sets this reconciler with the manager.
func (r *EndpointSliceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.epsDataActions = map[string]*epsData{}
	r.srvCounts = map[string]map[string]int{}
	predicates := predicate.Funcs{
		CreateFunc: r.createPredicate,
		UpdateFunc: r.updatePredicate,
		DeleteFunc: r.deletePredicate,
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&discoveryv1beta1.EndpointSlice{}).
		WithEventFilter(predicates).
		Complete(r)
}

func (r *EndpointSliceReconciler) createPredicate(ev event.CreateEvent) bool {
	epslice, ok := ev.Object.(*v1beta1.EndpointSlice)
	if !ok {
		return false
	}

	if !r.shouldWatchEpSlice(epslice) {
		return false
	}

	namespacedName := ktypes.NamespacedName{Namespace: epslice.Namespace, Name: epslice.Name}.String()
	epCount := 0
	for _, ep := range epslice.Endpoints {
		if ep.Conditions.Ready != nil && !*ep.Conditions.Ready {
			r.Log.WithValues("name", namespacedName).Info("found some in not ready", "len", len(ep.Addresses))
			continue
		}
		epCount += len(ep.Addresses)
	}

	r.Log.WithValues("name", namespacedName).Info("calculated len for this epslice", "len", epCount)

	r.lock.Lock()
	defer r.lock.Unlock()
	r.epsDataActions[namespacedName] = &epsData{
		srv:   epslice.Labels[v1beta1.LabelServiceName],
		count: epCount,
	}

	return true
}

func (r *EndpointSliceReconciler) updatePredicate(ev event.UpdateEvent) bool {
	evNew := event.CreateEvent{
		Meta:   ev.MetaNew,
		Object: ev.ObjectNew,
	}
	return r.createPredicate(evNew)
}

func (r *EndpointSliceReconciler) deletePredicate(ev event.DeleteEvent) bool {
	epslice, ok := ev.Object.(*v1beta1.EndpointSlice)
	if !ok {
		return false
	}

	if !r.shouldWatchEpSlice(epslice) {
		return false
	}

	namespacedName := ktypes.NamespacedName{Namespace: epslice.Namespace, Name: epslice.Name}.String()
	r.lock.Lock()
	defer r.lock.Unlock()
	r.epsDataActions[namespacedName] = &epsData{
		srv:   epslice.Labels[v1beta1.LabelServiceName],
		count: 0,
	}

	return true
}
