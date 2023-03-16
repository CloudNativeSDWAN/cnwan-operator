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
	"crypto/sha256"
	"encoding/hex"
	"fmt"

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
	servCtrlName string = "service-event-handler"
)

type serviceEventHandler struct {
	// client
	log zerolog.Logger
	ControllerOptions
}

func NewServiceController(mgr manager.Manager, opts *ControllerOptions, log zerolog.Logger) (controller.Controller, error) {
	if mgr == nil {
		return nil, ErrorInvalidManager
	}
	if opts == nil {
		return nil, ErrorInvalidControllerOptions
	}

	servHandler := &namespaceEventHandler{log: log}
	c, err := controller.New(nsCtrlName, mgr, controller.Options{
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

	if service.Spec.Type != corev1.ServiceTypeLoadBalancer {
		return
	}

	annotations := filterAnnotations(service.Annotations, s.ServiceAnnotations)
	if len(annotations) == 0 {
		return
	}

	ips, err := getIPsFromService(service)
	if err != nil {
		l.Err(err).Msg("error occurred while getting ips from service")
		return
	}
	if len(ips) == 0 {
		return
	}

	for _, port := range service.Spec.Ports {
		for _, ip := range ips {

			// Create an hashed name for this
			toBeHashed := fmt.Sprintf("%s:%d", ip, port.Port)
			h := sha256.New()
			h.Write([]byte(toBeHashed))
			hash := hex.EncodeToString(h.Sum(nil))

			// Only take the first 10 characters of the hashed name
			name := fmt.Sprintf("%s:%s", service.Name, hash[:10])
			_ = name
			// TODO: define the endpoint
		}
	}

	// TODO: check the namespace
	// TODO: send event to listener that a new service has been created.
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

	currAnnotations := filterAnnotations(curr.Annotations, s.ServiceAnnotations)
	oldAnnotations := filterAnnotations(old.Annotations, s.ServiceAnnotations)

	currIps, currErr := getIPsFromService(curr)
	oldIps, oldErr := getIPsFromService(old)
	if currErr != nil || oldErr != nil {
		err := currErr
		if err != nil {
			err = oldErr
		}
		l.Err(oldErr).Msg("error occurred while getting ips from service")
		return
	}

	// -----------------------------------------------
	// Determine if this event should be skipped
	// -----------------------------------------------

	switch {
	case curr.Spec.Type != corev1.ServiceTypeLoadBalancer &&
		old.Spec.Type != corev1.ServiceTypeLoadBalancer:
		// Wasn't a LoadBalancer and still isn't
	case len(oldAnnotations) == 0 && len(currAnnotations) == 0:
		// Didn't and still hasn't required annotations.
	case len(currIps) == 0 && len(oldIps) == 0:
		// Wasn't DNS and still isn't
		// TODO: case Namespace is not being watched:
		return
	}

	// -----------------------------------------------
	// Determine if we should remove this
	// -----------------------------------------------

	mustBeRemoved := func() (remove bool, reason string) {
		switch {
		case len(currIps) == 0:
			remove, reason = true, "no ips found"
		case curr.Spec.Type != corev1.ServiceTypeLoadBalancer:
			remove, reason = true, "not a LoadBalancer"
		case len(currAnnotations) == 0:
			remove, reason = true, "no valid annotations"
		case len(currIps) == 0:
			remove, reason = true, "no valid hostnames/ips found"
		}

		return
	}

	if remove, reason := mustBeRemoved(); remove {
		l.Info().Str("reason", reason).Msg("sending delete...")
		// TODO
		return
	}

	// TODO
}

// Delete handles delete events.
func (s *serviceEventHandler) Delete(de event.DeleteEvent, wq workqueue.RateLimitingInterface) {
	l := s.log.With().Str("name", "delete-event-handler").Logger()
	defer wq.Done(de.Object)

	service, ok := de.Object.(*corev1.Service)
	if !ok {
		return
	}

	if service.Spec.Type != corev1.ServiceTypeLoadBalancer {
		return
	}

	annotations := filterAnnotations(service.Annotations, s.ServiceAnnotations)
	if len(annotations) == 0 {
		return
	}

	ips, err := getIPsFromService(service)
	if err != nil {
		l.Err(err).Msg("error occurred while getting ips from service")
		return
	}
	if len(ips) == 0 {
		return
	}

	// TODO: check the namespace
	// TODO: send event to listener that a service has been deleted.
}

// Generic handles generic events.
func (s *serviceEventHandler) Generic(ge event.GenericEvent, wq workqueue.RateLimitingInterface) {
	// We don't really know what to do with generic events.
	// We will just ignore this.
	wq.Done(ge.Object)
}
