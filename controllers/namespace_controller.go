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

// NamespaceReconciler reconciles a Namespace object
type NamespaceReconciler struct {
	*BaseReconciler

	cacheNsWatch map[string]bool
	lock         sync.Mutex
}

// +kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=namespaces/status,verbs=get;update;patch

// Reconcile checks the changes in a service and reflects those changes in the service registry
func (r *NamespaceReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	l := r.Log.WithName("NamespaceReconciler").WithValues("namespace", req.NamespacedName)

	shouldWatch := func() bool {
		r.lock.Lock()
		defer r.lock.Unlock()
		defer delete(r.cacheNsWatch, req.NamespacedName.String())
		val, exists := r.cacheNsWatch[req.NamespacedName.String()]
		if !exists {
			val = false
		}

		return val
	}()

	var servList corev1.ServiceList
	if err := r.List(ctx, &servList, &client.ListOptions{Namespace: req.Name}); err != nil {
		l.Error(err, "error while getting services")
		return ctrl.Result{}, err
	}

	// First, check the services
	for _, serv := range servList.Items {
		if shouldWatch {
			serv.Annotations = r.filterAnnotations(serv.Annotations)
			nsData, servData, endpList, err := r.ServRegBroker.Reg.ExtractData(&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: req.Name, Namespace: req.Namespace}}, &serv)
			if err != nil {
				l.WithValues("serv-name", servData.Name).Error(err, "error while extracting data from the namespace and service")
				return ctrl.Result{}, nil
			}

			nsData.Metadata = map[string]string{}
			if strings.ToLower(serv.Labels[countPodsLabelKey]) == enableVal {
				servData.Metadata[r.CountPodKey] = fmt.Sprintf("%d", r.epsliceCounter.getSrvCount(serv.Namespace, serv.Name))
			}

			if _, err := r.ServRegBroker.ManageNs(nsData); err != nil {
				l.WithValues("ns-name", nsData.Name).Error(err, "error while processing namespace change")
				return ctrl.Result{}, nil
			}
			if len(servData.Metadata) > 0 && len(endpList) > 0 {
				if _, err := r.ServRegBroker.ManageServ(servData); err != nil {
					l.WithValues("serv-name", servData.Name).Error(err, "error while updating service")
					continue
				}
				if _, err := r.ServRegBroker.ManageServEndps(servData.NsName, servData.Name, endpList); err != nil {
					l.WithValues("serv-name", servData.Name).Error(err, "an error occurred while processing service's endpoints")
					continue
				}
			}
		} else {
			if err := r.ServRegBroker.RemoveServ(serv.Namespace, serv.Name, true); err != nil {
				l.Error(err, "error while deleting service")
			}
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager ...
func (r *NamespaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.cacheNsWatch = map[string]bool{}
	predicates := predicate.Funcs{
		CreateFunc: r.createPredicate,
		UpdateFunc: r.updatePredicate,
		DeleteFunc: r.deletePredicate,
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Namespace{}).
		WithEventFilter(predicates).
		Complete(r)
}

func (r *NamespaceReconciler) createPredicate(ev event.CreateEvent) bool {
	if !r.shouldWatchNs(ev.Meta.GetLabels()) {
		return false
	}

	namespacedName := ktypes.NamespacedName{Namespace: ev.Meta.GetNamespace(), Name: ev.Meta.GetName()}.String()
	r.lock.Lock()
	defer r.lock.Unlock()
	r.cacheNsWatch[namespacedName] = true
	return true
}

func (r *NamespaceReconciler) updatePredicate(ev event.UpdateEvent) bool {
	wasWatched := r.shouldWatchNs(ev.MetaOld.GetLabels())
	isWatched := r.shouldWatchNs(ev.MetaNew.GetLabels())

	if isWatched == wasWatched {
		return false
	}

	namespacedName := ktypes.NamespacedName{Namespace: ev.MetaNew.GetNamespace(), Name: ev.MetaNew.GetName()}.String()
	r.lock.Lock()
	defer r.lock.Unlock()
	r.cacheNsWatch[namespacedName] = isWatched
	return true
}

func (r *NamespaceReconciler) deletePredicate(ev event.DeleteEvent) bool {
	if !r.shouldWatchNs(ev.Meta.GetLabels()) {
		return false
	}

	namespacedName := ktypes.NamespacedName{Namespace: ev.Meta.GetNamespace(), Name: ev.Meta.GetName()}.String()
	r.lock.Lock()
	defer r.lock.Unlock()
	r.cacheNsWatch[namespacedName] = false
	return true
}
