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
	"sync"

	"github.com/CloudNativeSDWAN/cnwan-operator/internal/types"
	"github.com/CloudNativeSDWAN/cnwan-operator/internal/utils"
	sr "github.com/CloudNativeSDWAN/cnwan-operator/pkg/servregistry"
	"github.com/go-logr/logr"
	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ktypes "k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// NamespaceReconciler reconciles a Namespace object
type NamespaceReconciler struct {
	client.Client
	Log           logr.Logger
	Scheme        *runtime.Scheme
	nsLastConf    map[string]types.ListPolicy
	cacheNsWatch  map[string]bool
	lock          sync.Mutex
	ServRegBroker *sr.Broker
	*Utils
}

// +kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=namespaces/status,verbs=get;update;patch

// Reconcile checks the changes in a service and reflects those changes in the service registry
func (r *NamespaceReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	// TODO: re-format this to only do the least amount of work now that we
	// have predicates.
	ctx := context.Background()
	l := r.Log.WithValues("namespace", req.NamespacedName)

	// Get the namespace
	var ns corev1.Namespace
	deleted := false

	err := r.Get(ctx, req.NamespacedName, &ns)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			l.Error(err, "unable to fetch the namespace")
			// we'll ignore not-found errors, since they can't be fixed by an immediate
			// requeue (we'll need to wait for a new notification), and we can get them
			// on deleted requests.
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}

		l.V(0).Info("namespace was deleted")
		ns.Name = req.Name
		ns.Namespace = req.Namespace
		deleted = false
	}

	if r.ServRegBroker == nil {
		l.Error(fmt.Errorf("%s", "service registry broker is nil"), "cannot handle namespace")
		return ctrl.Result{}, nil
	}

	change, nsList := func(nsData corev1.Namespace) (bool, types.ListPolicy) {
		r.lock.Lock()
		defer r.lock.Unlock()

		currPolicy := types.ListPolicy(viper.GetString(types.NamespaceListPolicy))
		previousList, existed := r.nsLastConf[nsData.Name]
		var nsList types.ListPolicy

		if currPolicy == types.AllowList {
			// Defaults for an allowlist
			nsList = types.BlockList
			if len(previousList) == 0 {
				previousList = types.BlockList
			}

			if _, exists := nsData.Labels[types.AllowedKey]; exists {
				nsList = types.AllowList
			}
		}
		if currPolicy == types.BlockList {
			// Defaults for a blocklist
			nsList = types.AllowList
			if len(previousList) == 0 {
				previousList = types.AllowList
			}

			if _, exists := nsData.Labels[types.BlockedKey]; exists {
				nsList = types.BlockList
			}
		}

		// Update
		r.nsLastConf[ns.Name] = nsList

		if !existed {
			// If we have no data about previous configuration,
			// then we don't need to do anything, as this namespace
			// will be created as services are created.
			// We update the conf even if it is not the case.
			return false, currPolicy
		}

		return nsList != previousList, nsList
	}(ns)

	if !change {
		// Nothing to do here
		return ctrl.Result{}, nil
	}

	// Change is needed
	if nsList == types.AllowList {
		l.V(0).Info("namespace needs to be allowed")
	} else {
		l.V(0).Info("namespace needs to be blocked")
	}

	var servList corev1.ServiceList

	if err := r.List(ctx, &servList, &client.ListOptions{Namespace: ns.Name}); err != nil {
		l.Error(err, "error while getting services")
		return ctrl.Result{}, err
	}

	// First, check the services
	for _, serv := range servList.Items {
		if nsList == types.BlockList || deleted {
			if err := r.ServRegBroker.RemoveServ(serv.Namespace, serv.Name, true); err != nil {
				l.Error(err, "error while deleting service")
			}
		} else {

			// Get the data in our simpler format
			// Note: as of now, we are not copying any annotations from a namespace
			serv.Annotations = utils.FilterAnnotations(serv.Annotations)
			nsData, servData, endpList, err := r.ServRegBroker.Reg.ExtractData(&ns, &serv)
			if err != nil {
				l.WithValues("serv-name", servData.Name).Error(err, "error while extracting data from the namespace and service")
				return ctrl.Result{}, nil
			}
			nsData.Metadata = map[string]string{}

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
		}
	}

	if nsList == types.BlockList || deleted {
		if err := r.ServRegBroker.RemoveNs(ns.Name, true); err != nil {
			l.Error(err, "error while deleting service")
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager ...
func (r *NamespaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.cacheNsWatch = map[string]bool{}
	r.nsLastConf = map[string]types.ListPolicy{}
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
	if !r.ShouldWatchNs(ev.Meta.GetLabels()) {
		return false
	}

	namespacedName := ktypes.NamespacedName{Namespace: ev.Meta.GetNamespace(), Name: ev.Meta.GetName()}.String()
	r.lock.Lock()
	defer r.lock.Unlock()
	r.cacheNsWatch[namespacedName] = true
	return true
}

func (r *NamespaceReconciler) updatePredicate(ev event.UpdateEvent) bool {
	wasWatched := r.ShouldWatchNs(ev.MetaOld.GetLabels())
	isWatched := r.ShouldWatchNs(ev.MetaNew.GetLabels())

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
	if !r.ShouldWatchNs(ev.Meta.GetLabels()) {
		return false
	}

	namespacedName := ktypes.NamespacedName{Namespace: ev.Meta.GetNamespace(), Name: ev.Meta.GetName()}.String()
	r.lock.Lock()
	defer r.lock.Unlock()
	r.cacheNsWatch[namespacedName] = false
	return true
}
