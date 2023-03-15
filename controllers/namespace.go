// Copyright (c) 2023 Cisco Systems, Inc. and its affiliates
// All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"

	"github.com/rs/zerolog"
	corev1 "k8s.io/api/core/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	nsCtrlName         string = "namespace-event-handler"
	watchLabel         string = "operator.cnwan.io/watch"
	watchEnabledLabel  string = "enabled"
	watchDisabledLabel string = "disabled"
)

type namespaceEventHandler struct {
	// client
	log zerolog.Logger
	NamespaceControllerOptions
}

type NamespaceControllerOptions struct {
	WatchAllByDefault  bool
	ServiceAnnotations []string
}

func NewNamespaceController(mgr manager.Manager, opts *NamespaceControllerOptions, log zerolog.Logger) (controller.Controller, error) {
	if mgr == nil {
		return nil, ErrorInvalidManager
	}
	if opts == nil {
		return nil, ErrorInvalidNamespaceOptions
	}

	nsHandler := &namespaceEventHandler{log: log}
	c, err := controller.New(nsCtrlName, mgr, controller.Options{
		Reconciler: reconcile.Func(func(c context.Context, r reconcile.Request) (reconcile.Result, error) {
			return reconcile.Result{}, nil
		}),
	})

	if err != nil {
		return nil, err
	}

	err = c.Watch(&source.Kind{Type: &corev1.Namespace{}}, nsHandler)
	if err != nil {
		return nil, err
	}

	return c, nil
}

// Create handles create events.
func (n *namespaceEventHandler) Create(ce event.CreateEvent, wq workqueue.RateLimitingInterface) {
	defer wq.Done(ce.Object)

	namespace, ok := ce.Object.(*corev1.Namespace)
	if !ok {
		return
	}

	if !shouldWatchLabel(namespace.Labels, n.WatchAllByDefault) {
		return
	}

	// TODO: send event to listener that a new namespace has been created.
}

// Update handles update events.
func (n *namespaceEventHandler) Update(ue event.UpdateEvent, wq workqueue.RateLimitingInterface) {
	defer wq.Done(ue.ObjectNew)

	curr, currok := ue.ObjectNew.(*corev1.Namespace)
	old, oldok := ue.ObjectOld.(*corev1.Namespace)
	if !currok || !oldok {
		return
	}

	watchedBefore := shouldWatchLabel(curr.Labels, n.WatchAllByDefault)
	watchNow := shouldWatchLabel(old.Labels, n.WatchAllByDefault)

	switch {
	case !watchedBefore && !watchNow:
		return
	case !watchedBefore && watchNow:
		// TODO: send create ns, send create for all services inside it
	case watchedBefore && !watchNow:
		// TODO: send delete for all services inside it, send delete ns
	}
}

// Delete handles delete events.
func (n *namespaceEventHandler) Delete(de event.DeleteEvent, wq workqueue.RateLimitingInterface) {
	defer wq.Done(de.Object)

	namespace, ok := de.Object.(*corev1.Namespace)
	if !ok {
		return
	}

	if !shouldWatchLabel(namespace.Labels, n.WatchAllByDefault) {
		return
	}

	// TODO: send event to listener that a namespace has been deleted.
}

// Generic handles generic events.
func (n *namespaceEventHandler) Generic(ge event.GenericEvent, wq workqueue.RateLimitingInterface) {
	// We don't really know what to do with generic events.
	// We will just ignore this.
	wq.Done(ge.Object)
}

func shouldWatchLabel(labels map[string]string, watchAllByDefault bool) bool {
	switch labels[watchLabel] {
	case watchEnabledLabel:
		return true
	case watchDisabledLabel:
		return false
	default:
		return watchAllByDefault
	}
}
