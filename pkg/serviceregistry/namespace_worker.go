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

	serego "github.com/CloudNativeSDWAN/serego/api/core/types"
	"github.com/rs/zerolog"
)

type namespaceWorkerData struct {
	worker    *namespaceWorker
	ctx       context.Context
	canc      context.CancelFunc
	lastEvent time.Time
}

type namespaceWorker struct {
	log        zerolog.Logger
	eventsChan chan *Event
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
			switch o := event.Object.(type) {
			case *serego.Namespace:
				// TODO: handle namespace event
				l.Info().Str("event type", string(event.EventType)).Str("name", o.Name).Msg("received namespace event")
			case *serego.Service:
				// TODO: handle service event
				l.Info().Str("event type", string(event.EventType)).Str("name", o.Namespace+"/"+o.Name).Msg("received service event")
			case *serego.Endpoint:
				// TODO: handle endpoint event
				l.Info().Str("event type", string(event.EventType)).Str("name", o.Namespace+"/"+o.Service+"/"+o.Name).Msg("received endpoint event")
			}
		}
	}
}
