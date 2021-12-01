// Copyright © 2021 Cisco
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

package cloudmap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	sr "github.com/CloudNativeSDWAN/cnwan-operator/pkg/servregistry"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
)

const (
	defaultTimeout time.Duration = time.Minute
)

var (
	// pollOperationFrequency is the time betwenn two consecutive
	// pollOperationResult calls.
	pollOperationFrequency time.Duration = 2 * time.Second
)

// cloudMapIDs is used to hold ARNs and IDs and is only used internally
type cloudMapIDs struct {
	arn string
	id  string
}

// Handler is in charge of handling all the operations that need to be
// performed in AWS Cloud Map.
type Handler struct {
	Client cloudMapClientIface
	// TODO: on next versions the context will be provided explicitly for
	// each call.
	mainCtx context.Context
	log     logr.Logger
}

// NewHandler returns a new instance of the Handler.
func NewHandler(ctx context.Context, client *servicediscovery.Client, log logr.Logger) *Handler {
	return &Handler{client, ctx, log}
}

func (h *Handler) ExtractData(ns *corev1.Namespace, serv *corev1.Service) (*sr.Namespace, *sr.Service, []*sr.Endpoint, error) {
	// NOTE: on future versions, this function will be removed from service
	// registry and moved to the broker instead: it's not this package's job
	// to convert structs.
	if ns == nil {
		return nil, nil, nil, sr.ErrNsNotProvided
	}
	if serv == nil {
		return nil, nil, nil, sr.ErrServNotProvided
	}

	// Parse the namespace
	namespaceData := &sr.Namespace{
		Name:     ns.Name,
		Metadata: ns.Annotations,
	}
	if namespaceData.Metadata == nil {
		namespaceData.Metadata = map[string]string{}
	}

	// Parse the service
	// NOTE: we put metadata on the service in service directory,
	// not on the endpoints
	serviceData := &sr.Service{
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

	endpointsData := []*sr.Endpoint{}
	for _, port := range serv.Spec.Ports {
		for _, ip := range ips {

			// Create an hashed name for this
			toBeHashed := fmt.Sprintf("%s-%d", ip, port.Port)
			h := sha256.New()
			h.Write([]byte(toBeHashed))
			hash := hex.EncodeToString(h.Sum(nil))

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

	return namespaceData, serviceData, endpointsData, nil
}
