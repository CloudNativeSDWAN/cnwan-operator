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
	"fmt"
	"strings"
	"time"

	"github.com/CloudNativeSDWAN/cnwan-operator/internal/types"
	sr "github.com/CloudNativeSDWAN/cnwan-operator/pkg/servregistry"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/discovery/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	ktypes "k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// BaseReconciler is the base controller/reconciler upon which all other
// controllers will be based.
type BaseReconciler struct {
	client.Client
	Log                logr.Logger
	Scheme             *runtime.Scheme
	ServRegBroker      *sr.Broker
	AllowedAnnotations map[string]bool
	CurrentNsPolicy    types.ListPolicy
}

// NewBaseReconciler returns a new instance of a base reconciler to be used
// to create other reconcilers.
func NewBaseReconciler(cli client.Client, scheme *runtime.Scheme, broker *sr.Broker, anns []string, currNsPolicy types.ListPolicy) *BaseReconciler {
	allowedAnnotations := map[string]bool{}
	for _, ann := range anns {
		allowedAnnotations[ann] = true
	}

	return &BaseReconciler{
		Client:             cli,
		Log:                ctrl.Log.WithName("Controller"),
		Scheme:             scheme,
		ServRegBroker:      broker,
		AllowedAnnotations: allowedAnnotations,
		CurrentNsPolicy:    currNsPolicy,
	}
}

// shouldWatchNs returns true a namespace should be watched according to the
// provided labels and the list policy currently implemented.
func (b *BaseReconciler) shouldWatchNs(labels map[string]string) (watch bool) {
	switch b.CurrentNsPolicy {
	case types.AllowList:
		if _, exists := labels[types.AllowedKey]; exists {
			watch = true
		}
	case types.BlockList:
		if _, exists := labels[types.BlockedKey]; !exists {
			watch = true
		}
	}

	return
}

func (b *BaseReconciler) shouldWatchSrv(srv *corev1.Service) bool {
	nsrv := ktypes.NamespacedName{Namespace: srv.Namespace, Name: srv.Name}
	l := b.Log.WithValues("service", nsrv)
	if srv.Spec.Type != corev1.ServiceTypeLoadBalancer {
		return false
	}

	if len(srv.Status.LoadBalancer.Ingress) == 0 {
		return false
	}

	filteredAnnotations := b.filterAnnotations(srv.Annotations)
	if len(filteredAnnotations) == 0 {
		return false
	}

	var ns corev1.Namespace
	if err := b.Get(context.Background(), ktypes.NamespacedName{Name: srv.Namespace}, &ns); err != nil {
		l.Error(err, "error while getting parent namespace from service")
		return false
	}

	return b.shouldWatchNs(ns.Labels)
}

// filterAnnotations takes a map of annotations and returnes a new one
// stripped from the ones that should not be registered on the service
// registry.
func (b *BaseReconciler) filterAnnotations(annotations map[string]string) map[string]string {
	if len(annotations) == 0 {
		return map[string]string{}
	}
	if len(b.AllowedAnnotations) == 0 {
		return map[string]string{}
	}

	if _, exists := b.AllowedAnnotations["*/*"]; exists {
		return annotations
	}

	filtered := map[string]string{}
	for key, val := range annotations {

		// Check this key specifically
		if _, exists := b.AllowedAnnotations[key]; exists {
			filtered[key] = val
			continue
		}

		prefixName := strings.Split(key, "/")
		if len(prefixName) != 2 {
			// This key is not in prefix/name format
			continue
		}

		prefixWildcard := fmt.Sprintf("%s/*", prefixName[0])
		if _, exists := b.AllowedAnnotations[prefixWildcard]; exists {
			filtered[key] = val
			continue
		}

		wildcardName := fmt.Sprintf("*/%s", prefixName[1])
		if _, exists := b.AllowedAnnotations[wildcardName]; exists {
			filtered[key] = val
		}
	}

	return filtered
}

func (b *BaseReconciler) shouldWatchEpSlice(epslice *v1beta1.EndpointSlice) bool {
	srvName, exists := epslice.Labels[v1beta1.LabelServiceName]
	if !exists {
		return false
	}

	ctx, canc := context.WithTimeout(context.Background(), 30*time.Second)
	defer canc()

	var srv corev1.Service
	if err := b.Get(ctx, ktypes.NamespacedName{Name: srvName, Namespace: epslice.Namespace}, &srv); err != nil {
		b.Log.Error(err, "error while getting service from endpointslice", "endpointslice", epslice.Name)
		return false
	}
	if !b.shouldWatchSrv(&srv) {
		return false
	}

	enabled := srv.Labels[countPodsLabelKey]
	if strings.ToLower(enabled) != enableVal {
		return false
	}

	if epslice.AddressType != v1beta1.AddressTypeIPv4 {
		return false
	}

	return true
}

// NamespaceReconciler returns a namespace reconciler starting from this
// base reconciler.
func (b *BaseReconciler) NamespaceReconciler() *NamespaceReconciler {
	return &NamespaceReconciler{
		BaseReconciler: b,
	}
}

// ServiceReconciler returns a service reconciler starting from this
// base reconciler.
func (b *BaseReconciler) ServiceReconciler() *ServiceReconciler {
	return &ServiceReconciler{
		BaseReconciler: b,
	}
}

// EndpointSliceReconciler returns a service reconciler starting from this
// base reconciler.
func (b *BaseReconciler) EndpointSliceReconciler() *EndpointSliceReconciler {
	return &EndpointSliceReconciler{
		BaseReconciler: b,
	}
}
