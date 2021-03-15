// Copyright © 2020, 2021 Cisco
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
	"fmt"
	"reflect"
	"strings"
	"sync"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ktypes "k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// ServiceReconciler reconciles a Service object
type ServiceReconciler struct {
	*BaseReconciler

	cacheSrvWatch map[string]bool
	lock          sync.Mutex
}

// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services/status,verbs=get;update;patch

// Reconcile checks the changes in a service and reflects those changes in the service registry
func (r *ServiceReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	l := r.Log.WithName("ServiceReconciler").WithValues("service", req.NamespacedName)

	shouldWatch := func() bool {
		r.lock.Lock()
		defer r.lock.Unlock()
		defer delete(r.cacheSrvWatch, req.NamespacedName.String())
		val, exists := r.cacheSrvWatch[req.NamespacedName.String()]
		if !exists {
			val = false
		}

		return val
	}()

	if !shouldWatch {
		// No need to load anything if we just need to delete.
		if err := r.ServRegBroker.RemoveServ(req.Namespace, req.Name, true); err != nil {
			l.Error(err, "an error occurred while processing service deletion")
		}

		return ctrl.Result{}, nil
	}

	// Get the service
	var service corev1.Service
	err := r.Get(ctx, req.NamespacedName, &service)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			l.Error(err, "unable to fetch the service")
			// we'll ignore not-found errors, since they can't be fixed by an immediate
			// requeue (we'll need to wait for a new notification), and we can get them
			// on deleted requests.
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}

		return ctrl.Result{}, err
	}

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: req.Namespace, Annotations: map[string]string{}}}
	service.Annotations = r.filterAnnotations(service.Annotations)
	nsData, servData, endpList, err := r.ServRegBroker.Reg.ExtractData(ns, &service)
	if err != nil {
		l.Error(err, "error while getting data from the namespace and service")
		return ctrl.Result{}, nil
	}

	nsData.Metadata = map[string]string{}
	if strings.ToLower(service.Labels[countPodsLabelKey]) == enableVal {
		servData.Metadata[r.CountPodKey] = fmt.Sprintf("%d", r.epsliceCounter.getSrvCount(service.Namespace, service.Name))
	}

	if _, err := r.ServRegBroker.ManageNs(nsData); err != nil {
		l.WithValues("ns-name", nsData.Name).Error(err, "an error occurred while processing the namespace")
		return ctrl.Result{}, nil
	}
	if _, err := r.ServRegBroker.ManageServ(servData); err != nil {
		l.WithValues("serv-name", nsData.Name).Error(err, "an error occurred while processing the service")
		return ctrl.Result{}, nil
	}
	if _, err := r.ServRegBroker.ManageServEndps(nsData.Name, servData.Name, endpList); err != nil {
		l.WithValues("serv-name", nsData.Name).Error(err, "an error occurred while processing service's endpoints")
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

// SetupWithManager ...
func (r *ServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.cacheSrvWatch = map[string]bool{}
	predicates := predicate.Funcs{
		CreateFunc: r.createPredicate,
		UpdateFunc: r.updatePredicate,
		DeleteFunc: r.deletePredicate,
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Service{}).
		WithEventFilter(predicates).
		Complete(r)
}

func (r *ServiceReconciler) createPredicate(ev event.CreateEvent) bool {

	srv, ok := ev.Object.(*corev1.Service)
	if !ok {
		return false
	}
	if !r.shouldWatchSrv(srv) {
		return false
	}

	nsrv := ktypes.NamespacedName{Namespace: srv.Namespace, Name: srv.Name}.String()
	r.lock.Lock()
	defer r.lock.Unlock()
	r.cacheSrvWatch[nsrv] = true
	return true
}

func (r *ServiceReconciler) updatePredicate(ev event.UpdateEvent) bool {
	old, ok := ev.ObjectOld.(*corev1.Service)
	if !ok {
		return false
	}
	curr, ok := ev.ObjectNew.(*corev1.Service)
	if !ok {
		return false
	}

	wasWatched, shouldWatch := r.shouldWatchSrv(old), r.shouldWatchSrv(curr)
	nsrv := ktypes.NamespacedName{Namespace: curr.Namespace, Name: curr.Name}.String()
	watchAction := false
	switch {
	case !wasWatched && !shouldWatch:
		return false
	case !wasWatched && shouldWatch:
		watchAction = true
	case wasWatched && !shouldWatch:
		watchAction = false
	default:
		changeOccurred := func() bool {
			if strings.ToLower(curr.Labels[countPodsLabelKey]) != strings.ToLower(old.Labels[countPodsLabelKey]) {
				return true
			}

			if !reflect.DeepEqual(r.filterAnnotations(old.Annotations), r.filterAnnotations(curr.Annotations)) {
				return true
			}

			if !reflect.DeepEqual(old.Status.LoadBalancer.Ingress, curr.Status.LoadBalancer.Ingress) {
				return true
			}

			return false
		}()
		if !changeOccurred {
			// Nothing relevant to us changed
			return false
		}

		watchAction = true
	}

	r.lock.Lock()
	defer r.lock.Unlock()
	r.cacheSrvWatch[nsrv] = watchAction
	return true
}

func (r *ServiceReconciler) deletePredicate(ev event.DeleteEvent) bool {
	srv, ok := ev.Object.(*corev1.Service)
	if !ok {
		return false
	}
	if !r.shouldWatchSrv(srv) {
		return false
	}

	nsrv := ktypes.NamespacedName{Namespace: srv.Namespace, Name: srv.Name}.String()
	r.lock.Lock()
	defer r.lock.Unlock()
	r.cacheSrvWatch[nsrv] = false
	return true
}
