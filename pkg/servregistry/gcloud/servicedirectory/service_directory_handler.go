// Copyright © 2020, 2021 Cisco
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

package servicedirectory

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	sr "github.com/CloudNativeSDWAN/cnwan-operator/pkg/servregistry"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
)

const (
	defTimeout time.Duration = 30 * time.Second
)

// Handler is a wrapper for Service Directory that exposes its methods in a
// sort of "universal" way through the ServiceRegistry interface.
type Handler struct {
	// ProjectID where ServiceDirectory is enabled.
	ProjectID string
	// DefaultRegion where namespaces, services and endpoints are going to be
	// registered to.
	DefaultRegion string
	// Log to use.
	Log logr.Logger
	// Context to use for requests.
	// TODO: remove this in favor of explicit context for each call?
	Context context.Context
	// Client to wrap around.
	Client regClient
}

func (s *Handler) ExtractData(ns *corev1.Namespace, serv *corev1.Service) (namespaceData *sr.Namespace, serviceData *sr.Service, endpointsData []*sr.Endpoint, err error) {
	if ns == nil {
		err = sr.ErrNsNotProvided
		return
	}
	if serv == nil {
		err = sr.ErrServNotProvided
		return
	}

	// Parse the namespace
	namespaceData = &sr.Namespace{
		Name:     ns.Name,
		Metadata: ns.Annotations,
	}
	if namespaceData.Metadata == nil {
		namespaceData.Metadata = map[string]string{}
	}

	// Parse the service
	// NOTE: we put metadata on the service in service directory,
	// not on the endpoints
	serviceData = &sr.Service{
		Name:     serv.Name,
		NsName:   ns.Name,
		Metadata: serv.Annotations,
	}
	if serviceData.Metadata == nil {
		serviceData.Metadata = map[string]string{}
	}

	// Get the endpoints from the service
	// First, build the ips
	ips := []string{}
	ips = append(ips, serv.Spec.ExternalIPs...)

	// Get data from load balancers
	for _, ing := range serv.Status.LoadBalancer.Ingress {
		ips = append(ips, ing.IP)
	}

	for _, port := range serv.Spec.Ports {
		for _, ip := range ips {

			// Create an hashed name for this
			toBeHashed := fmt.Sprintf("%s-%d", ip, port.Port)
			h := sha256.New()
			h.Write([]byte(toBeHashed))
			hash := fmt.Sprintf("%x", h.Sum(nil))

			// Only take the first 10 characters of the hashed name
			name := fmt.Sprintf("%s-%s", serv.Name, hash[:10])
			endpointsData = append(endpointsData, &sr.Endpoint{
				Name:     name,
				NsName:   namespaceData.Name,
				ServName: serviceData.Name,
				Address:  ip,
				Port:     port.Port,
				Metadata: map[string]string{},
			})
		}
	}

	return
}
