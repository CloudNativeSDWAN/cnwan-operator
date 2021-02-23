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

	discoveryv1beta1 "k8s.io/api/discovery/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// EndpointSliceReconciler reconciles a EndpointSlice object
type EndpointSliceReconciler struct {
	*BaseReconciler
}

// +kubebuilder:rbac:groups=discovery.k8s.io,resources=endpointslice,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=discovery.k8s.io,resources=endpointslice/status,verbs=get;update;patch

// Reconcile keeps track counts in the endpointslice length
func (r *EndpointSliceReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithName("EndpointSliceReconciler").WithValues("endpointslice", req.NamespacedName)

	// TODO: implement me

	return ctrl.Result{}, nil
}

// SetupWithManager sets this reconciler with the manager.
func (r *EndpointSliceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&discoveryv1beta1.EndpointSlice{}).
		Complete(r)
}
