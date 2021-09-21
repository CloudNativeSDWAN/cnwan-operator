// Copyright © 2020 Cisco
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

package servregistry

import (
	"sync"
	"time"

	"github.com/go-logr/logr"
	cache "github.com/patrickmn/go-cache"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const (
	defOpKey = "owner"
	defOpVal = "cnwan-operator"
)

// This file contains the definition of the Broker struct.

// Broker is a structure that acts as an intermediary, setting up data - i.e.
// namespaces, services and endpoints - and performing checks before calling
// the appropriate functions of the service registry.
//
// Its functions are split on namespace.go, service.go and endpoint.go to
// make the package more readable.
type Broker struct {
	Reg ServiceRegistry
	log logr.Logger

	cache          *cache.Cache
	opMetaPair     MetadataPair
	persistentMeta []MetadataPair
	lock           sync.Mutex
}

// MetadataPair represents a key-value pair that is/will be registered in a
// service registry.
type MetadataPair struct {
	// Key of the metadata
	Key string
	// Value of the metadata
	Value string
}

// NewBroker returns a new instance of service registry broker.
//
// An error is returned in case no service registry where to perform operations
// is provided.
func NewBroker(reg ServiceRegistry, opMetaPair MetadataPair, persMeta ...MetadataPair) (*Broker, error) {
	// Validation and inits
	l := zap.New(zap.UseDevMode(true)).WithName("ServiceRegistryBroker")

	if reg == nil {
		return nil, ErrServRegNotProvided
	}

	if opMetaPair.Key == "" {
		opMetaPair.Key = defOpKey
	}
	if opMetaPair.Value == "" {
		opMetaPair.Value = defOpVal
	}

	return &Broker{
		log:            l,
		Reg:            reg,
		opMetaPair:     opMetaPair,
		persistentMeta: persMeta,
		cache:          cache.New(5*time.Minute, 10*time.Minute),
	}, nil
}
