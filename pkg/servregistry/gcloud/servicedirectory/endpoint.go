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

package servicedirectory

import (
	sr "github.com/CloudNativeSDWAN/cnwan-operator/pkg/servregistry"
)

// GetEndp returns the endpoint if exists.
func (s *servDir) GetEndp(nsName, servName, endpName string) (*sr.Endpoint, error) {
	// TODO
	return nil, nil
}

// ListServ returns a list of services inside the provided namespace.
func (s *servDir) ListEndp(nsName, servName string) ([]*sr.Endpoint, error) {
	// TODO
	return nil, nil
}

// CreateEndp creates the endpoint.
func (s *servDir) CreateEndp(endp *sr.Endpoint) (*sr.Endpoint, error) {
	// TODO
	return nil, nil
}

// UpdateEndp updates the endpoint.
func (s *servDir) UpdateEndp(endp *sr.Endpoint) (*sr.Endpoint, error) {
	// TODO
	return nil, nil
}

// DeleteEndp deletes the endpoint.
func (s *servDir) DeleteEndp(nsName, servName, endpName string) error {
	// TODO
	return nil
}
