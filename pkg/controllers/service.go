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
	"k8s.io/apimachinery/pkg/types"
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
	servCtrlName string = "service-event-handler"
)

type serviceEventHandler struct {
	client client.Client
	log    zerolog.Logger
	*ControllerOptions
}

func NewServiceController(mgr manager.Manager, opts *ControllerOptions, log zerolog.Logger) (controller.Controller, error) {
	if mgr == nil {
		return nil, ErrorInvalidManager
	}
	if opts == nil {
		return nil, ErrorInvalidControllerOptions
	}

	servHandler := &serviceEventHandler{
		client:            mgr.GetClient(),
		log:               log,
		ControllerOptions: opts,
	}
	c, err := controller.New(servCtrlName, mgr, controller.Options{
		Reconciler: reconcile.Func(func(c context.Context, r reconcile.Request) (reconcile.Result, error) {
			return reconcile.Result{}, nil
		}),
	})

	if err != nil {
		return nil, err
	}

	err = c.Watch(&source.Kind{Type: &corev1.Service{}}, servHandler)
	if err != nil {
		return nil, err
	}

	return c, nil
}

// Create handles create events.
func (s *serviceEventHandler) Create(ce event.CreateEvent, wq workqueue.RateLimitingInterface) {
	l := s.log.With().Str("name", "create-event-handler").Logger()
	defer wq.Done(ce.Object)

	service, ok := ce.Object.(*corev1.Service)
	if !ok {
		return
	}

	l = l.With().Str("name", types.NamespacedName{
		Namespace: service.Namespace,
		Name:      service.Name,
	}.String()).Logger()

	watchNs, err := s.checkParentNamespace(service)
	if !watchNs {
		if err != nil {
			l.Err(err).Msg("cannot check parent namespace")
		}

		return
	}

	checkedService := checkService(service, s.ServiceAnnotations)
	if !checkedService.passed {
		if checkedService.err != nil {
			l.Err(err).Msg("cannot check service")
		}

		return
	}

	// Send an event to create the namespace. NOTE: we do this because we have
	// no idea whether the namespace controller sent this before us. Se we
	// disabled the namespace controller from sending Create events, and we let
	// the service controller do that.
	s.EventsChan <- &serviceregistry.Event{
		EventType: serviceregistry.EventCreate,
		Object: &serego.Namespace{
			Name: service.Namespace,
		},
	}

	s.EventsChan <- &serviceregistry.Event{
		EventType: serviceregistry.EventCreate,
		Object: &serego.Service{
			Name:      service.Name,
			Namespace: service.Namespace,
		},
	}

	for _, ep := range checkedService.endpoints {
		s.EventsChan <- &serviceregistry.Event{
			EventType: serviceregistry.EventCreate,
			Object:    ep,
		}
	}
}

// Update handles update events.
func (s *serviceEventHandler) Update(ue event.UpdateEvent, wq workqueue.RateLimitingInterface) {
	l := s.log.With().Str("name", "update-event-handler").Logger()
	defer wq.Done(ue.ObjectNew)

	curr, currok := ue.ObjectNew.(*corev1.Service)
	old, oldok := ue.ObjectOld.(*corev1.Service)
	if !currok || !oldok {
		return
	}

	l = l.With().Str("name", types.NamespacedName{
		Namespace: curr.Namespace,
		Name:      curr.Name,
	}.String()).Logger()

	watchNs, err := s.checkParentNamespace(curr)
	if !watchNs {
		if err != nil {
			l.Err(err).Msg("cannot check parent namespace")
		}

		return
	}

	currChecked := checkService(curr, s.ServiceAnnotations)
	oldChecked := checkService(old, s.ServiceAnnotations)

	if currChecked.err != nil || oldChecked.err != nil {
		err := currChecked.err
		if err != nil {
			err = oldChecked.err
		}
		l.Err(err).Msg("error occurred while getting ips from service")
		return
	}

	// Easiest cases
	switch {
	case !oldChecked.passed && !currChecked.passed:
		return
	case oldChecked.passed && !currChecked.passed:
		l.Info().Str("reason", currChecked.reason).Msg("sending delete...")

		for _, ep := range oldChecked.endpoints {
			s.EventsChan <- &serviceregistry.Event{
				EventType: serviceregistry.EventDelete,
				Object:    ep,
			}
		}

		s.EventsChan <- &serviceregistry.Event{
			EventType: serviceregistry.EventDelete,
			Object: &serego.Service{
				Name:      old.Name,
				Namespace: old.Namespace,
			},
		}

		return
	case !oldChecked.passed && currChecked.passed:
		l.Info().Msg("sending create...")
		s.EventsChan <- &serviceregistry.Event{
			EventType: serviceregistry.EventCreate,
			Object: &serego.Service{
				Name:      curr.Name,
				Namespace: curr.Namespace,
			},
		}

		for _, ep := range currChecked.endpoints {
			s.EventsChan <- &serviceregistry.Event{
				EventType: serviceregistry.EventCreate,
				Object:    ep,
			}
		}

		return
	}

	// Check what is changed
	oldEndpoints := getEndpointsMapFromSlice(oldChecked.endpoints)
	currEndpoints := getEndpointsMapFromSlice(currChecked.endpoints)

	// Check what must be removed, updated or created
	for _, ep := range oldEndpoints {
		currEp := currEndpoints[ep.Name]

		if currEp == nil {
			s.EventsChan <- &serviceregistry.Event{
				EventType: serviceregistry.EventDelete,
				Object:    ep,
			}
		} else {
			s.EventsChan <- &serviceregistry.Event{
				EventType: serviceregistry.EventUpdate,
				Object:    currEp,
			}
		}
	}

	for _, ep := range currEndpoints {
		if _, exists := currEndpoints[ep.Name]; !exists {
			s.EventsChan <- &serviceregistry.Event{
				EventType: serviceregistry.EventCreate,
				Object:    ep,
			}
		}
	}
}

// Delete handles delete events.
func (s *serviceEventHandler) Delete(de event.DeleteEvent, wq workqueue.RateLimitingInterface) {
	l := s.log.With().Str("handler", "service-delete-event-handler").Logger()
	defer wq.Done(de.Object)

	service, ok := de.Object.(*corev1.Service)
	if !ok {
		return
	}

	l = l.With().Str("name", types.NamespacedName{
		Namespace: service.Namespace,
		Name:      service.Name,
	}.String()).Logger()

	watchNs, err := s.checkParentNamespace(service)
	if !watchNs {
		if err != nil {
			l.Err(err).Msg("cannot check parent namespace")
		}

		return
	}

	checkedService := checkService(service, s.ServiceAnnotations)

	for _, ep := range checkedService.endpoints {
		s.EventsChan <- &serviceregistry.Event{
			EventType: serviceregistry.EventDelete,
			Object:    ep,
		}
	}

	s.EventsChan <- &serviceregistry.Event{
		EventType: serviceregistry.EventDelete,
		Object: &serego.Service{
			Name:      service.Name,
			Namespace: service.Namespace,
		},
	}
}

func (s *serviceEventHandler) checkParentNamespace(service *corev1.Service) (bool, error) {
	ctx, canc := context.WithTimeout(context.Background(), 10*time.Second)
	defer canc()

	var namespace corev1.Namespace
	if err := s.client.
		Get(ctx, types.NamespacedName{Name: service.Namespace}, &namespace); err != nil {
		return false, err
	}

	return checkNsLabels(namespace.Labels, s.WatchNamespacesByDefault), nil
}

// Generic handles generic events.
func (s *serviceEventHandler) Generic(ge event.GenericEvent, wq workqueue.RateLimitingInterface) {
	// We don't really know what to do with generic events.
	// We will just ignore this.
	wq.Done(ge.Object)
}
