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
	"sync"
	"time"

	serego "github.com/CloudNativeSDWAN/serego/api/core"
	"github.com/rs/zerolog"
)

type EventType string

const (
	EventCreate EventType = "create"
	EventUpdate EventType = "update"
	EventDelete EventType = "delete"
)

const (
	maximumIdleDuration = 5 * time.Minute
)

type Event struct {
	EventType
	Object interface{}
}

type EventHandler struct {
	seregoClient   *serego.ServiceRegistry
	workers        map[string]*namespaceWorkerData
	waitGroup      sync.WaitGroup
	log            zerolog.Logger
	persistentMeta map[string]string
}

func NewEventHandler(seregoClient *serego.ServiceRegistry, persistentMeta map[string]string, log zerolog.Logger) *EventHandler {
	return &EventHandler{
		seregoClient:   seregoClient,
		workers:        map[string]*namespaceWorkerData{},
		waitGroup:      sync.WaitGroup{},
		log:            log,
		persistentMeta: persistentMeta,
	}
}

func (e *EventHandler) WatchForEvents(mainCtx context.Context, eventsChannel chan *Event) error {
	l := e.log.With().Logger()
	l.Info().Msg("watching for events from the cluster...")

	cleanUpTicker := time.NewTicker(time.Minute)
	for {
		select {

		case <-mainCtx.Done():
			l := e.log.With().Str("from", "event handler").Logger()
			l.Info().Msg("cancel requested")
			if len(e.workers) == 0 {
				return nil
			}

			l.Info().Msg("propagating cancel to all namespace workers...")

			for _, nsWorker := range e.workers {
				nsWorker.canc()
			}

			l.Debug().Msg("waiting for all namespace workers to finish...")
			e.waitGroup.Wait()
			l.Info().Msg("all namespace workers exited: goodbye!")
			return nil

		case event := <-eventsChannel:
			l := e.log.With().Str("from", "event-dispatcher").Logger()
			namespaceName := getNamespaceNameFromEventObject(event)

			if namespaceName == "" {
				l.Warn().Msg("could not find namespace name: skipping...")
				continue
			}

			nsWorker := e.getOrCreateNamespaceWorker(mainCtx, namespaceName)

			l.Info().Msg("dispatching event to namespace worker...")

			nsWorker.worker.eventsChan <- event
			nsWorker.lastEvent = time.Now()

		case <-cleanUpTicker.C:
			l := e.log.With().Str("from", "worker-manager").Logger()

			now := time.Now()
			toRemove := []string{}

			for name, worker := range e.workers {
				if now.Sub(worker.lastEvent) > maximumIdleDuration {
					l.Info().Str("namespace", name).
						Msg("worker exceeded maximum idle time: signaling stop...")
					worker.canc()
					toRemove = append(toRemove, name)
				}
			}

			for _, workerToRemove := range toRemove {
				delete(e.workers, workerToRemove)
			}
		}
	}
}

func (e *EventHandler) getOrCreateNamespaceWorker(mainCtx context.Context, name string) *namespaceWorkerData {
	l := e.log.With().Str("namespace", name).Logger()

	nsWorker, exists := e.workers[name]
	if exists {
		l.Debug().Msg("worker already running")
		return nsWorker
	}

	l.Debug().Msg("creating namespace worker...")
	data := &namespaceWorkerData{
		worker: &namespaceWorker{
			nsop:           e.seregoClient.Namespace(name),
			log:            e.log.With().Str("worker", name+"-event-handler").Logger(),
			eventsChan:     make(chan *Event, 25),
			persistentMeta: e.persistentMeta,
		},
	}
	data.ctx, data.canc = context.WithCancel(mainCtx)
	e.workers[name] = data

	// Add it to the wait group so we can successfully wait for it to finish
	e.waitGroup.Add(1)
	go func() {
		defer e.waitGroup.Done()
		data.worker.handleNamespacedEvents(data.ctx)
	}()

	return data
}
