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
	"fmt"
	"sync"
	"time"

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
	srvRecon       *ServiceReconciler
}

// +kubebuilder:rbac:groups=discovery.k8s.io,resources=endpointslice,verbs=get;list;watch;create;update;patch;delete

// Reconcile keeps track counts in the endpointslice length
func (r *EndpointSliceReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	l := r.Log.WithName("EndpointSliceReconciler").WithValues("endpointslice", req.NamespacedName)
	namespacedName := req.NamespacedName.String()
	data := func(nname string) *epsData {
		r.lock.Lock()
		defer r.lock.Unlock()
		defer delete(r.epsDataActions, nname)

		return r.epsDataActions[nname]
	}(namespacedName)
	if data == nil {
		l.Error(fmt.Errorf("no data exists"), "could not get endpointslice data for this endpointslice")
		return ctrl.Result{}, nil
	}

	srvname := ktypes.NamespacedName{Namespace: req.Namespace, Name: data.srv}
	wind, exists := r.srvWindows[srvname.String()]
	if !exists {
		r.srvWindows[srvname.String()] = &window{values: []*windowValue{}}
		wind = r.srvWindows[srvname.String()]
	}

	wind.lock.Lock()
	defer wind.lock.Unlock()

	countBeforeUpd := r.epsliceCounter.getSrvCount(srvname.Namespace, srvname.Name)
	r.epsliceCounter.putSrvCount(req.Namespace, srvname.Name, req.Name, data.count)
	newCount := r.epsliceCounter.getSrvCount(srvname.Namespace, srvname.Name)

	if len(wind.values) > 0 {
		if newCount > wind.getHighest() {
			l.Info("new count detected and is the highest in the window, updating service registry...", "highest", wind.getHighest(), "new-count", newCount)
			r.srvRecon.cacheSrvWatch[srvname.String()] = true
			r.srvRecon.Reconcile(ctrl.Request{NamespacedName: srvname})
		} else {
			l.Info("new count detected, but not highest in window: performing cooldown...", "highest", wind.getHighest(), "new-count", newCount)
		}
	} else {
		if newCount > countBeforeUpd {
			l.Info("new count detected and window is empty, updating service registry...", "old-val", countBeforeUpd, "new-val", newCount)
			r.srvRecon.cacheSrvWatch[srvname.String()] = true
			r.srvRecon.Reconcile(ctrl.Request{NamespacedName: srvname})
		} else {
			l.Info("new count detected, but not higher than current value, performing cooldown...", "old-val", countBeforeUpd, "new-val", newCount)
		}
	}

	wind.values = append(wind.values, &windowValue{
		epsliceName:  req.Name,
		epsliceCount: data.count,
		totalCount:   newCount,
		timer: time.AfterFunc(time.Minute, func() {
			r.exitWindow(srvname)
		}),
	})

	return ctrl.Result{}, nil
}

func (r *EndpointSliceReconciler) exitWindow(srv ktypes.NamespacedName) {
	l := r.Log.WithName("EndpointSliceReconciler").WithValues("Window", srv)

	wind := r.srvWindows[srv.String()]
	wind.lock.Lock()
	defer wind.lock.Unlock()

	if len(wind.values) == 0 {
		l.V(2).Info("window is empty, returning...")
		return
	}

	oldestVal := wind.values[0]
	oldestVal.timer.Stop()
	defer func() {
		wind.values = wind.values[1:]
	}()

	highestVal := r.epsliceCounter.getSrvCount(srv.Namespace, srv.Name)
	for i := 1; i < len(wind.values); i++ {
		if wind.values[i].totalCount > highestVal {
			highestVal = wind.values[i].totalCount
		}
	}

	if oldestVal.totalCount <= highestVal && len(wind.values) > 1 {
		l.Info("highest value isn't changed, returning...", "exiting-value", oldestVal.totalCount, "highest", highestVal)
		return
	}

	l.Info("updating service registry...", "highest-value", highestVal)
	r.srvRecon.cacheSrvWatch[srv.String()] = true
	r.srvRecon.Reconcile(ctrl.Request{NamespacedName: srv})
}

// SetServiceReconciler sets the service reconciler, so that the endpointslice
// reconciler can refer to it.
func (r *EndpointSliceReconciler) SetServiceReconciler(srvrecon *ServiceReconciler) *EndpointSliceReconciler {
	r.srvRecon = srvrecon
	return r
}

// SetupWithManager sets this reconciler with the manager.
func (r *EndpointSliceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.epsDataActions = map[string]*epsData{}
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
