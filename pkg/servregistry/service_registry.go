// Copyright © 2020 Cisco
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

package servregistry

import (
	corev1 "k8s.io/api/core/v1"
)

// ServiceRegistry is an interface containing functions that are implemented
// by a service registry.
type ServiceRegistry interface {
	// GetNs returns the namespace if exists.
	GetNs(name string) (*Namespace, error)
	// ListNs returns a list of all namespaces.
	ListNs() ([]*Namespace, error)
	// CreateNs creates the namespace.
	CreateNs(ns *Namespace) (*Namespace, error)
	// UpdateNs updates the namespace.
	UpdateNs(ns *Namespace) (*Namespace, error)
	// DeleteNs deletes the namespace.
	DeleteNs(name string) error
	// GetServ returns the service if exists.
	GetServ(nsName, servName string) (*Service, error)
	// ListServ returns a list of services inside the provided namespace.
	ListServ(nsName string) ([]*Service, error)
	// CreateServ creates the service.
	CreateServ(serv *Service) (*Service, error)
	// UpdateServ updates the service.
	UpdateServ(serv *Service) (*Service, error)
	// DeleteServ deletes the service.
	DeleteServ(nsName, servName string) error
	// GetEndp returns the endpoint if exists.
	GetEndp(nsName, servName, endpName string) (*Endpoint, error)
	// ListEndp returns a list of endpoints belonging to the provided namespace and service.
	ListEndp(nsName, servName string) ([]*Endpoint, error)
	// CreateEndp creates the endpoint.
	CreateEndp(endp *Endpoint) (*Endpoint, error)
	// UpdateEndp updates the endpoint.
	UpdateEndp(endp *Endpoint) (*Endpoint, error)
	// DeleteEndp deletes the endpoint.
	DeleteEndp(nsName, servName, endpName string) error
	// ExtractData extracts relevant data from the provided Kubernetes namespace and service
	// and returns a namespace, service and an array of endpoints with data relevant to this
	// specific service registry.
	ExtractData(ns *corev1.Namespace, serv *corev1.Service) (*Namespace, *Service, []*Endpoint, error)
}
