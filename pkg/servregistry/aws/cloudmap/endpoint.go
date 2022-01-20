// Copyright © 2021 Cisco
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

package cloudmap

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	sr "github.com/CloudNativeSDWAN/cnwan-operator/pkg/servregistry"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery/types"
)

func (h *Handler) listOrGetEndpoint(nsName, servName string, epName *string) ([]*sr.Endpoint, error) {
	ctx, canc := context.WithTimeout(h.mainCtx, defaultTimeout)
	defer canc()

	instList, err := h.Client.DiscoverInstances(ctx, &servicediscovery.DiscoverInstancesInput{
		NamespaceName: aws.String(nsName),
		ServiceName:   aws.String(servName),
	})
	if err != nil {
		var oe *types.NamespaceNotFound
		if errors.As(err, &oe) {
			return nil, sr.ErrNotFound
		}

		var os *types.ServiceNotFound
		if errors.As(err, &os) {
			return nil, sr.ErrNotFound
		}
		return nil, err
	}

	endpList := []*sr.Endpoint{}
	for _, inst := range instList.Instances {
		if epName != nil && aws.ToString(inst.InstanceId) != *epName {
			continue
		}

		metadata := inst.Attributes
		ipv4 := metadata["AWS_INSTANCE_IPV4"]
		if ipv4 == "" {
			h.log.WithName("ListEndpoints").Info("skipping instance with no address", "name", inst.InstanceId)
			continue
		}

		strPort := inst.Attributes["AWS_INSTANCE_PORT"]
		if strPort == "" {
			h.log.WithName("ListEndpoints").Info("skipping instance with no port", "name", inst.InstanceId)
			continue
		}
		port, _ := strconv.ParseInt(strPort, 10, 32)

		delete(metadata, "AWS_INSTANCE_PORT")
		delete(metadata, "AWS_INSTANCE_IPV4")

		ep := &sr.Endpoint{
			NsName:   nsName,
			ServName: servName,
			Name:     aws.ToString(inst.InstanceId),
			Address:  ipv4,
			Port:     int32(port),
			Metadata: metadata,
		}

		if epName != nil {
			return []*sr.Endpoint{ep}, nil
		}

		endpList = append(endpList, ep)
	}

	return endpList, nil
}

// GetEndp returns the endpoint if exists.
func (h *Handler) GetEndp(nsName, servName, endpName string) (*sr.Endpoint, error) {
	if nsName == "" {
		return nil, sr.ErrNsNameNotProvided
	}
	if servName == "" {
		return nil, sr.ErrServNameNotProvided
	}

	list, err := h.listOrGetEndpoint(nsName, servName, &endpName)
	if err != nil {

		return nil, err
	}
	if len(list) == 0 {
		return nil, sr.ErrNotFound
	}

	return list[0], nil
}

// ListServ returns a list of services inside the provided namespace.
func (h *Handler) ListEndp(nsName, servName string) (endpList []*sr.Endpoint, err error) {
	if nsName == "" {
		return nil, sr.ErrNsNameNotProvided
	}
	if servName == "" {
		return nil, sr.ErrServNameNotProvided
	}

	return h.listOrGetEndpoint(nsName, servName, nil)
}

// CreateEndp creates the endpoint.
func (h *Handler) CreateEndp(endp *sr.Endpoint) (*sr.Endpoint, error) {
	if endp == nil {
		return nil, sr.ErrEndpNotProvided
	}
	if endp.NsName == "" {
		return nil, sr.ErrNsNameNotProvided
	}
	if endp.ServName == "" {
		return nil, sr.ErrServNameNotProvided
	}
	if endp.Name == "" {
		return nil, sr.ErrEndpNameNotProvided
	}
	if endp.Address == "" {
		return nil, fmt.Errorf("no address provided")
	}
	if endp.Port <= 0 {
		return nil, fmt.Errorf("invalid port provided")
	}

	_, nsID, err := h.listOrGetNamespace(&endp.NsName)
	if err != nil {
		return nil, err
	}
	if len(nsID) == 0 {
		return nil, sr.ErrNotFound
	}

	_, servID, err := h.listOrGetService(endp.NsName, &endp.ServName)
	if err != nil {
		return nil, err
	}
	if len(servID) == 0 {
		return nil, sr.ErrNotFound
	}

	attributes := endp.Metadata
	attributes["AWS_INSTANCE_PORT"] = fmt.Sprintf("%d", endp.Port)
	attributes["AWS_INSTANCE_IPV4"] = endp.Address

	ctx, canc := context.WithTimeout(h.mainCtx, defaultTimeout)
	defer canc()
	out, err := h.Client.RegisterInstance(ctx, &servicediscovery.RegisterInstanceInput{
		ServiceId:  aws.String(servID[0].id),
		InstanceId: aws.String(endp.Name),
		Attributes: attributes,
	})
	if err != nil {
		var oe *types.ServiceNotFound
		if errors.As(err, &oe) {
			return nil, sr.ErrNotFound
		}

		// any other error
		return nil, err
	}

	l := h.log.WithName("CreateInstance")

	l.Info("waiting for operation to complete...")
	if err := h.pollOperationStatus(aws.ToString(out.OperationId)); err != nil {
		l.Info("operation completed with error")
		return nil, err
	}
	l.Info("operation completed successfully")

	return endp, nil
}

// UpdateEndp updates the endpoint.
func (h *Handler) UpdateEndp(endp *sr.Endpoint) (*sr.Endpoint, error) {
	return h.CreateEndp(endp)
}

// DeleteEndp deletes the endpoint.
func (h *Handler) DeleteEndp(nsName, servName, endpName string) error {
	if nsName == "" {
		return sr.ErrNsNameNotProvided
	}
	if servName == "" {
		return sr.ErrServNameNotProvided
	}
	if endpName == "" {
		return sr.ErrEndpNameNotProvided
	}

	_, nsID, err := h.listOrGetNamespace(&nsName)
	if err != nil {
		return err
	}
	if len(nsID) == 0 {
		return sr.ErrNotFound
	}

	_, servID, err := h.listOrGetService(nsName, &servName)
	if err != nil {
		return err
	}
	if len(servID) == 0 {
		return sr.ErrNotFound
	}

	ctx, canc := context.WithTimeout(h.mainCtx, defaultTimeout)
	defer canc()

	out, err := h.Client.DeregisterInstance(ctx, &servicediscovery.DeregisterInstanceInput{
		ServiceId:  aws.String(servID[0].id),
		InstanceId: aws.String(endpName),
	})
	if err != nil {
		var oe *types.ServiceNotFound
		if errors.As(err, &oe) {
			return sr.ErrNotFound
		}

		var oi *types.InstanceNotFound
		if errors.As(err, &oi) {
			return sr.ErrNotFound
		}

		// any other error
		return err
	}

	l := h.log.WithName("DeleteInstance")

	l.Info("waiting for operation to complete...")
	if err := h.pollOperationStatus(aws.ToString(out.OperationId)); err != nil {
		l.Info("operation completed with error")
		return err
	}
	l.Info("operation completed successfully")

	return nil
}
