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

	"github.com/go-logr/logr"
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

	opKey string
	opVal string
	lock  sync.Mutex
}

// NewBroker returns a new instance of service registry broker.
//
// An error is returned in case no service registry where to perform operations
// is provided.
func NewBroker(reg ServiceRegistry, opKey, opVal string) (*Broker, error) {
	// Validation and inits
	l := zap.New(zap.UseDevMode(true)).WithName("ServiceRegistryBroker")

	if reg == nil {
		return nil, ErrServRegNotProvided
	}

	if len(opKey) == 0 {
		opKey = defOpKey
	}
	if len(opVal) == 0 {
		opVal = defOpVal
	}

	return &Broker{
		log:   l,
		Reg:   reg,
		opKey: opKey,
		opVal: opVal,
	}, nil
}
