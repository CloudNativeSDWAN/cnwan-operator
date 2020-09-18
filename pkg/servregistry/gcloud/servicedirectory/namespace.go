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

// GetNs returns the namespace if exists.
func (s *servDir) GetNs(name string) (*sr.Namespace, error) {
	// TODO
	return nil, nil
}

// ListNs returns a list of all namespaces.
func (s *servDir) ListNs() ([]*sr.Namespace, error) {
	// TODO
	return nil, nil
}

// CreateNs creates the namespace.
func (s *servDir) CreateNs(ns *sr.Namespace) (*sr.Namespace, error) {
	// TODO
	return nil, nil
}

// UpdateNs updates the namespace.
func (s *servDir) UpdateNs(ns *sr.Namespace) (*sr.Namespace, error) {
	// TODO
	return nil, nil
}

// DeleteNs deletes the namespace.
func (s *servDir) DeleteNs(name string) error {
	// TODO
	return nil
}