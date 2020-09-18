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

// GetServ returns the service if exists.
func (s *servDir) GetServ(nsName, servName string) (*sr.Service, error) {
	// TODO
	return nil, nil
}

// ListServ returns a list of services inside the provided namespace.
func (s *servDir) ListServ(nsName string) ([]*sr.Service, error) {
	// TODO
	return nil, nil
}

// CreateServ creates the service.
func (s *servDir) CreateServ(serv *sr.Service) (*sr.Service, error) {
	// TODO
	return nil, nil
}

// UpdateServ updates the service.
func (s *servDir) UpdateServ(serv *sr.Service) (*sr.Service, error) {
	// TODO
	return nil, nil
}

// DeleteServ deletes the service.
func (s *servDir) DeleteServ(nsName, servName string) error {
	// TODO
	return nil
}
