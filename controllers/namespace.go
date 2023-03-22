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
	"time"

	"github.com/CloudNativeSDWAN/cnwan-operator/pkg/serviceregistry"
	serego "github.com/CloudNativeSDWAN/serego/api/core/types"
	"github.com/rs/zerolog"
	corev1 "k8s.io/api/core/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

type ControllerOptions struct {
	WatchNamespacesByDefault bool
	ServiceAnnotations       []string
	EventsChan               chan *serviceregistry.Event
}

type namespaceEventHandler struct {
	client client.Client
	log    zerolog.Logger
	*ControllerOptions
}

func NewNamespaceController(mgr manager.Manager, opts *ControllerOptions, log zerolog.Logger) (controller.Controller, error) {
	if mgr == nil {
		return nil, ErrorInvalidManager
	}
	if opts == nil {
		return nil, ErrorInvalidControllerOptions
	}

	nsHandler := &namespaceEventHandler{
		client:            mgr.GetClient(),
		log:               log,
		ControllerOptions: opts,
	}
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
	// The namespace is created once an appropriate service appears.
	wq.Done(ce.Object)
}

// Update handles update events.
func (n *namespaceEventHandler) Update(ue event.UpdateEvent, wq workqueue.RateLimitingInterface) {
	defer wq.Done(ue.ObjectNew)

	curr, currok := ue.ObjectNew.(*corev1.Namespace)
	old, oldok := ue.ObjectOld.(*corev1.Namespace)
	if !currok || !oldok {
		return
	}

	watchNow := checkNsLabels(curr.Labels, n.WatchNamespacesByDefault)
	watchedBefore := checkNsLabels(old.Labels, n.WatchNamespacesByDefault)

	switch {
	case watchedBefore && watchNow:
		return
	case watchedBefore && !watchNow:
		n.handleUpdateEvent(curr, serviceregistry.EventDelete)
	case !watchedBefore && watchNow:
		n.handleUpdateEvent(curr, serviceregistry.EventCreate)
	}
}

// Delete handles delete events.
func (n *namespaceEventHandler) Delete(de event.DeleteEvent, wq workqueue.RateLimitingInterface) {
	defer wq.Done(de.Object)

	namespace, ok := de.Object.(*corev1.Namespace)
	if !ok {
		return
	}

	if !checkNsLabels(namespace.Labels, n.WatchNamespacesByDefault) {
		return
	}

	n.handleUpdateEvent(namespace, serviceregistry.EventDelete)
}

func (n *namespaceEventHandler) handleUpdateEvent(namespace *corev1.Namespace, eventType serviceregistry.EventType) {
	ctx, canc := context.WithTimeout(context.Background(), 10*time.Second)
	defer canc()

	services := corev1.ServiceList{}
	if err := n.client.List(ctx, &services, &client.ListOptions{
		Namespace: namespace.Name,
	}); err != nil {
		n.log.Err(err).Str("namespace", namespace.Name).
			Msg("cannot retrieve list of services inside namespace")
		return
	}

	// Inline function definitions for sending the namespace and service
	sendNsEvent := func() {
		n.EventsChan <- &serviceregistry.Event{
			EventType: eventType,
			Object: &serego.Namespace{
				Name: namespace.Name,
			},
		}
	}
	switch eventType {
	case serviceregistry.EventCreate:
		sendNsEvent()
	case serviceregistry.EventDelete:
		defer sendNsEvent()
	}

	sendServiceEvent := func(name string) {
		n.EventsChan <- &serviceregistry.Event{
			EventType: eventType,
			Object: &serego.Service{
				Namespace: namespace.Name,
				Name:      name,
			},
		}
	}

	for _, service := range services.Items {
		checkedService := checkService(&service, n.ServiceAnnotations)
		if !checkedService.passed {
			continue
		}

		func() {
			// using an anonymous function, so we can defer events
			// if needed.
			switch eventType {
			case serviceregistry.EventCreate:
				sendServiceEvent(service.Name)
			case serviceregistry.EventDelete:
				defer sendServiceEvent(service.Name)
			}

			for _, endpoint := range checkedService.endpoints {
				n.EventsChan <- &serviceregistry.Event{
					EventType: serviceregistry.EventDelete,
					Object:    endpoint,
				}
			}
		}()
	}
}

// Generic handles generic events.
func (n *namespaceEventHandler) Generic(ge event.GenericEvent, wq workqueue.RateLimitingInterface) {
	// We don't really know what to do with generic events.
	// We will just ignore this.
	wq.Done(ge.Object)
}
