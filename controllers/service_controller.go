// Copyright © 2020 Cisco
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

	sr "github.com/CloudNativeSDWAN/cnwan-operator/pkg/servregistry"
	"github.com/CloudNativeSDWAN/cnwan-operator/utils"

	cnwan_types "github.com/CloudNativeSDWAN/cnwan-operator/types"
	"github.com/go-logr/logr"
	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ServiceReconciler reconciles a Service object
type ServiceReconciler struct {
	client.Client
	Log           logr.Logger
	Scheme        *runtime.Scheme
	ServRegBroker *sr.Broker
}

// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services/status,verbs=get;update;patch

// Reconcile checks the changes in a service and reflects those changes in the service registry
func (r *ServiceReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	l := r.Log.WithValues("service", req.NamespacedName)
	deleted := false

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

		l.V(0).Info("service was deleted")
		service.Name = req.Name
		service.Namespace = req.Namespace
		deleted = true
	}

	// Get the namespace
	var ns corev1.Namespace
	if err := r.Get(ctx, types.NamespacedName{Name: service.Namespace}, &ns); err != nil {
		l.Error(err, "error occurred while trying to get namespace")
		return ctrl.Result{}, err
	}

	nsListPolicy := cnwan_types.ListPolicy(viper.GetString(cnwan_types.NamespaceListPolicy))

	if nsListPolicy == cnwan_types.AllowList {
		if _, exists := ns.Labels[cnwan_types.AllowedKey]; !exists {
			l.V(1).Info("ignoring service as namespace is not in the allow list")
			return ctrl.Result{}, nil
		}
	}

	if nsListPolicy == cnwan_types.BlockList {
		if _, exists := ns.Labels[cnwan_types.BlockedKey]; exists {
			l.V(1).Info("ignoring service as namespace is in the block list")
			return ctrl.Result{}, nil
		}
	}

	if r.ServRegBroker == nil {
		l.Error(fmt.Errorf("%s", "service registry broker is nil"), "cannot handle namespace")
		return ctrl.Result{}, nil
	}

	// Get the data in our simpler format
	// Note: as of now, we are not copying any annotations from a namespace
	service.Annotations = utils.FilterAnnotations(service.Annotations)
	nsData, servData, endpList, err := r.ServRegBroker.Reg.ExtractData(&ns, &service)
	if err != nil {
		l.Error(err, "error while getting data from the namespace and service")
		return ctrl.Result{}, nil
	}

	// We don't support metadata on namespaces right now
	nsData.Metadata = map[string]string{}

	if !deleted && len(endpList) > 0 && len(servData.Metadata) > 0 {
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

	if err := r.ServRegBroker.RemoveServ(ns.Name, service.Name, true); err != nil {
		l.WithValues("serv-name", nsData.Name).Error(err, "an error occurred while processing service deletion")
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

// SetupWithManager ...
func (r *ServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Service{}).
		Complete(r)
}
