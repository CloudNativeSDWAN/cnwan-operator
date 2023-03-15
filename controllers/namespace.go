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
	nsCtrlName string = "namespace-event-handler"
)

type namespaceEventHandler struct {
	log zerolog.Logger
}

func NewNamespaceController(mgr manager.Manager, log zerolog.Logger) (controller.Controller, error) {
	nsHandler := &namespaceEventHandler{log}

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

// Update handles update events.
func (n *namespaceEventHandler) Update(ue event.UpdateEvent, wq workqueue.RateLimitingInterface) {
	l := n.log.With().Str("event-handler", "Update").Logger()
	defer wq.Done(ue.ObjectNew)

	// TODO
	_ = l
}

// Delete handles delete events.
func (n *namespaceEventHandler) Delete(de event.DeleteEvent, wq workqueue.RateLimitingInterface) {
	defer wq.Done(de.Object)

	namespace, ok := de.Object.(*corev1.Namespace)
	if !ok {
		return
	}

	// TODO
	_ = namespace
}

// Create handles create events.
func (n *namespaceEventHandler) Create(ce event.CreateEvent, wq workqueue.RateLimitingInterface) {
	l := n.log.With().Str("event-handler", "Create").Logger()
	defer wq.Done(ce.Object)

	namespace, ok := ce.Object.(*corev1.Namespace)
	if !ok {
		return
	}

	// TODO
	_, _ = l, namespace
}

// Generic handles generic events.
func (n *namespaceEventHandler) Generic(ge event.GenericEvent, wq workqueue.RateLimitingInterface) {
	// We don't really know what to do with generic events.
	// We will just ignore this.
	wq.Done(ge.Object)
}
