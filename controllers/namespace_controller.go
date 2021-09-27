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

	"github.com/CloudNativeSDWAN/cnwan-operator/internal/utils"
	sr "github.com/CloudNativeSDWAN/cnwan-operator/pkg/servregistry"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	monitorLabel string = "operator.cnwan.io/monitor"
)

// NamespaceReconciler reconciles a Namespace object
type NamespaceReconciler struct {
	client.Client
	Log                        logr.Logger
	Scheme                     *runtime.Scheme
	MonitorNamespacesByDefault bool
	nsLastConf                 map[string]bool
	lock                       sync.Mutex
	ServRegBroker              *sr.Broker
}

// +kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=namespaces/status,verbs=get;update;patch

// Reconcile checks the changes in a service and reflects those changes in the service registry
func (r *NamespaceReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
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
		deleted = true
	}

	if r.ServRegBroker == nil {
		l.Error(fmt.Errorf("%s", "service registry broker is nil"), "cannot handle namespace")
		return ctrl.Result{}, nil
	}

	if deleted {
		// If this namespace was deleted, there is no point in loading
		// services: you won't find anything there.
		// So let's save ourselves some computation and just go straight to
		// business then.
		if err := r.ServRegBroker.RemoveNs(ns.Name, true); err != nil {
			l.Error(err, "error while deleting service")
		}

		r.lock.Lock()
		defer r.lock.Unlock()

		delete(r.nsLastConf, ns.Name)
		return ctrl.Result{}, nil
	}

	change, nsIsMonitored := func() (bool, bool) {
		var currentlyMonitored bool
		switch strings.ToLower(ns.Labels[monitorLabel]) {
		case "true":
			currentlyMonitored = true
		case "false":
			currentlyMonitored = false
		default:
			currentlyMonitored = r.MonitorNamespacesByDefault
		}

		r.lock.Lock()
		defer r.lock.Unlock()
		previouslyMonitored, existed := r.nsLastConf[ns.Name]
		if !existed {
			previouslyMonitored = r.MonitorNamespacesByDefault
		}

		changed := currentlyMonitored != previouslyMonitored
		r.nsLastConf[ns.Name] = currentlyMonitored
		return changed, currentlyMonitored
	}()
	if !change {
		return ctrl.Result{}, nil
	}

	var servList corev1.ServiceList
	if err := r.List(ctx, &servList, &client.ListOptions{Namespace: ns.Name}); err != nil {
		l.Error(err, "error while getting services")
		return ctrl.Result{}, err
	}

	// First, check the services
	for _, serv := range servList.Items {
		if !nsIsMonitored {
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

	if !nsIsMonitored {
		if err := r.ServRegBroker.RemoveNs(ns.Name, true); err != nil {
			l.Error(err, "error while deleting service")
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager ...
func (r *NamespaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.nsLastConf = map[string]bool{}

	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Namespace{}).
		Complete(r)
}
