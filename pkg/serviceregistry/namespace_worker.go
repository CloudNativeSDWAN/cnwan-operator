// Copyright © 2023 Cisco
//
// SPDX-License-Identifier: Apache-2.0
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

package serviceregistry

import (
	"context"
	"time"

	serego "github.com/CloudNativeSDWAN/serego/api/core"
	stypes "github.com/CloudNativeSDWAN/serego/api/core/types"
	serrors "github.com/CloudNativeSDWAN/serego/api/errors"
	"github.com/CloudNativeSDWAN/serego/api/options/register"
	"github.com/rs/zerolog"
)

type namespaceWorkerData struct {
	worker    *namespaceWorker
	ctx       context.Context
	canc      context.CancelFunc
	lastEvent time.Time
}

type namespaceWorker struct {
	nsop           *serego.NamespaceOperation
	log            zerolog.Logger
	eventsChan     chan *Event
	persistentMeta map[string]string
}

func (n *namespaceWorker) handleNamespacedEvents(ctx context.Context) error {
	l := n.log.With().Logger()
	l.Info().Msg("worker waiting for events for this namespace...")

	for {
		select {
		case <-ctx.Done():
			l.Info().Msg("received stop from manager: exiting...")
			return nil
		case event := <-n.eventsChan:

			switch event.EventType {
			case EventCreate, EventUpdate:
				n.handleCreateUpdate(ctx, event)
			case EventDelete:
				switch obj := event.Object.(type) {
				case *stypes.Namespace:
					n.handleDeleteNamespace(ctx, obj)
				case *stypes.Service:
					n.handleDeleteService(ctx, obj)
				case *stypes.Endpoint:
					n.handleDeleteEndpoint(ctx, obj)
				}
			}
		}
	}
}

func (n *namespaceWorker) handleCreateUpdate(mainCtx context.Context, event *Event) {
	ctx, canc := context.WithTimeout(mainCtx, time.Minute)
	defer canc()

	switch obj := event.Object.(type) {

	case *stypes.Namespace:
		l := n.log.With().Logger()
		l.Info().Msg("registering namespace...")
		if err := n.nsop.
			Register(ctx, register.WithMetadata(n.persistentMeta)); err != nil {
			l.Err(err).Msg("could not registrer namespace")
		} else {
			l.Info().Msg("namespace correctly registered")
		}

	case *stypes.Service:
		l := n.log.With().Str("service-name", obj.Name).Logger()
		l.Info().Msg("registering service...")
		if err := n.nsop.Service(obj.Name).
			Register(ctx, register.WithMetadata(n.persistentMeta)); err != nil {
			l.Err(err).Msg("could not registrer service")
		} else {
			l.Info().Msg("service correctly registered")
		}

	case *stypes.Endpoint:
		l := n.log.With().
			Str("service-name", obj.Service).
			Str("endpoint-name", obj.Name).
			Logger()
		l.Info().Msg("registering endpoint...")
		if err := n.nsop.Service(obj.Service).Endpoint(obj.Name).Register(ctx,
			register.WithAddress(obj.Address),
			register.WithPort(obj.Port),
			register.WithMetadata(obj.Metadata),
			register.WithMetadata(n.persistentMeta)); err != nil {
			l.Err(err).Msg("could not registrer endpoint")
		} else {
			l.Info().Msg("endpoint correctly registered")
		}

	}
}

func (n *namespaceWorker) handleDeleteEndpoint(mainCtx context.Context, endpoint *stypes.Endpoint) {
	ctx, canc := context.WithTimeout(mainCtx, time.Minute)
	defer canc()

	l := n.log.With().
		Str("service", endpoint.Service).
		Str("endpoint", endpoint.Name).
		Logger()

	eop := n.nsop.Service(endpoint.Service).Endpoint(endpoint.Name)

	ep, err := eop.Get(ctx)
	if err != nil {
		l.Warn().AnErr("error", err).Msg("cannot check if endpoint exists: it " +
			"might be already deleted")
		return
	}

	if !isOwnedByOperator(ep.Metadata) {
		l.Info().Str("reason", "not managed by CNWAN-Operator").
			Msg("skipping endpoint deletion")
		return
	}

	l.Info().Msg("deleting endpoint...")
	err = eop.Deregister(ctx)
	if err != nil {
		l.Err(err).Msg("cannot delete endpoint")
		return
	}

	l.Info().Msg("endpoint successfully deleted")
}

func (n *namespaceWorker) handleDeleteService(mainCtx context.Context, service *stypes.Service) {
	ctx, canc := context.WithTimeout(mainCtx, time.Minute)
	defer canc()

	l := n.log.With().Str("service", service.Name).Logger()

	sop := n.nsop.Service(service.Name)

	srv, err := sop.Get(ctx)
	if err != nil {
		l.Warn().AnErr("error", err).Msg("cannot check if service exists: it " +
			"might be already deleted")
		return
	}

	if !isOwnedByOperator(srv.Metadata) {
		l.Info().Str("reason", "not managed by CNWAN-Operator").
			Msg("skipping service deletion")
		return
	}

	_, _, err = sop.Endpoint(serego.Any).List().Next(ctx)
	switch {
	case err != nil && !serrors.IsIteratorDone(err):
		l.Err(err).Msg("cannot check if service is empty")
		return
	case err == nil:
		l.Info().Str("reason", "not empty").
			Msg("skipping service deletion")
		return
	}

	l.Info().Msg("deleting service...")
	err = sop.Deregister(ctx)
	if err != nil {
		l.Err(err).Msg("cannot delete service")
		return
	}

	l.Info().Msg("service successfully deleted")
}

func (n *namespaceWorker) handleDeleteNamespace(mainCtx context.Context, namespace *stypes.Namespace) {
	ctx, canc := context.WithTimeout(mainCtx, time.Minute)
	defer canc()

	l := n.log.With().Str("namespace", namespace.Name).Logger()

	ns, err := n.nsop.Get(ctx)
	if err != nil {
		l.Err(err).Msg("cannot check if namespace exists: won't be deleted")
		return
	}

	if !isOwnedByOperator(ns.Metadata) {
		l.Info().Str("reason", "not managed by CNWAN-Operator").
			Msg("skipping service deletion")
		return
	}

	_, _, err = n.nsop.Service(serego.Any).List().Next(ctx)
	switch {
	case err != nil && !serrors.IsIteratorDone(err):
		l.Err(err).Msg("cannot check if namespace is empty")
		return
	case err == nil:
		l.Info().Str("reason", "not empty").
			Msg("skipping namespace deletion")
		return
	}

	l.Info().Msg("deleting namespace...")
	err = n.nsop.Deregister(ctx)
	if err != nil {
		l.Err(err).Msg("cannot delete namespace")
		return
	}

	l.Info().Msg("namespace successfully deleted")
}
