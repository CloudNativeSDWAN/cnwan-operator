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
	"context"
	"strings"
	"time"

	sr "github.com/CloudNativeSDWAN/cnwan-operator/pkg/servregistry"
	"google.golang.org/api/iterator"
	sdpb "google.golang.org/genproto/googleapis/cloud/servicedirectory/v1"
	"google.golang.org/genproto/protobuf/field_mask"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetEndp returns the endpoint if exists.
func (s *servDir) GetEndp(nsName, servName, endpName string) (*sr.Endpoint, error) {
	// -- Init
	if err := s.checkNames(&nsName, &servName, &endpName); err != nil {
		return nil, err
	}
	l := s.log.WithName("GetEndp").WithValues("ns-name", nsName, "serv-name", servName, "endp-name", endpName)
	endpPath := s.getResourcePath(servDirPath{namespace: nsName, service: servName, endpoint: endpName})
	ctx, canc := context.WithTimeout(s.context, s.timeout)
	defer canc()

	sdEndp, err := s.client.GetEndpoint(ctx, &sdpb.GetEndpointRequest{Name: endpPath})
	if err == nil {
		endp := &sr.Endpoint{
			Name:     endpName,
			NsName:   nsName,
			ServName: servName,
			Metadata: sdEndp.Annotations,
		}
		if endp.Metadata == nil {
			endp.Metadata = map[string]string{}
		}

		return endp, nil
	}

	// What is the error?
	if err == context.DeadlineExceeded {
		l.Error(err, "timeout expired while waiting for service directory to reply", "timeout-seconds", s.timeout.Seconds())
		return nil, sr.ErrTimeOutExpired
	}

	if status.Code(err) == codes.NotFound {
		return nil, sr.ErrNotFound
	}

	// Any other error
	return nil, err
}

// ListServ returns a list of services inside the provided namespace.
func (s *servDir) ListEndp(nsName, servName string) (endpList []*sr.Endpoint, err error) {
	// -- Init
	if err := s.checkNames(&nsName, &servName, nil); err != nil {
		return nil, err
	}
	l := s.log.WithName("ListEndp").WithValues("ns-name", nsName, "serv-name", servName)
	ctx, canc := context.WithTimeout(s.context, time.Minute)
	defer canc()

	req := &sdpb.ListEndpointsRequest{
		Parent: s.getResourcePath(servDirPath{namespace: nsName, service: servName}),
	}

	iter := s.client.ListEndpoints(ctx, req)
	if iter == nil {
		l.V(0).Info("returned list is nil")
		return
	}
	for {
		nextEndp, iterErr := iter.Next()
		if iterErr != nil {

			if iterErr == context.DeadlineExceeded {
				l.Error(err, "timeout expired while waiting for service directory to reply", "timeout-seconds", s.timeout.Seconds())
				return nil, sr.ErrTimeOutExpired
			}

			if iterErr != iterator.Done {
				l.Error(iterErr, "error while loading endpoints")
				return nil, iterErr
			}

			break
		}

		// Create the list
		splitName := strings.Split(nextEndp.Name, "/")
		endp := &sr.Endpoint{
			Name:     splitName[len(splitName)-1],
			ServName: servName,
			NsName:   nsName,
			Metadata: nextEndp.Annotations,
		}
		if endp.Metadata == nil {
			endp.Metadata = map[string]string{}
		}

		endpList = append(endpList, endp)
	}

	return
}

// CreateEndp creates the endpoint.
func (s *servDir) CreateEndp(endp *sr.Endpoint) (*sr.Endpoint, error) {
	// -- Init
	if endp == nil {
		return nil, sr.ErrEndpNotProvided
	}
	if err := s.checkNames(&endp.NsName, &endp.ServName, &endp.Name); err != nil {
		return nil, err
	}
	l := s.log.WithName("CreateEndp").WithValues("ns-name", endp.NsName, "serv-name", endp.ServName, "endp-name", endp.Name, "metadata", endp.Metadata)
	ctx, canc := context.WithTimeout(s.context, s.timeout)
	defer canc()

	endpToCreate := &sdpb.Endpoint{
		Name:        endp.Name,
		Annotations: endp.Metadata,
		Address:     endp.Address,
		Port:        endp.Port,
	}

	req := &sdpb.CreateEndpointRequest{
		Parent:     s.getResourcePath(servDirPath{namespace: endp.NsName, service: endp.ServName}),
		EndpointId: endp.Name,
		Endpoint:   endpToCreate,
	}

	_, err := s.client.CreateEndpoint(ctx, req)
	if err == nil {
		// If it is successful, then it makes no point in parsing the returned
		// service from service directory, because it will look like just the
		// same as the service we want to create, apart from having prefixes
		// in the name, which is something we want to abstract to someone
		// using this.
		return endp, nil
	}

	// What is the error?
	if err == context.DeadlineExceeded {
		l.Error(err, "timeout expired while waiting for service directory to reply", "timeout-seconds", s.timeout.Seconds())
		return nil, sr.ErrTimeOutExpired
	}

	if status.Code(err) == codes.AlreadyExists {
		return nil, sr.ErrAlreadyExists
	}

	// Any other error
	return nil, err
}

// UpdateEndp updates the endpoint.
func (s *servDir) UpdateEndp(endp *sr.Endpoint) (*sr.Endpoint, error) {
	// -- Init
	if endp == nil {
		return nil, sr.ErrEndpNotProvided
	}
	if err := s.checkNames(&endp.NsName, &endp.ServName, &endp.Name); err != nil {
		return nil, err
	}
	l := s.log.WithName("UpdateEndp").WithValues("ns-name", endp.NsName, "serv-name", endp.ServName, "endp-name", endp.Name, "metadata", endp.Metadata)
	endpPath := s.getResourcePath(servDirPath{namespace: endp.NsName, service: endp.ServName, endpoint: endp.Name})
	ctx, canc := context.WithTimeout(s.context, s.timeout)
	defer canc()

	endpToUpd := &sdpb.Endpoint{
		Name:        endpPath,
		Annotations: endp.Metadata,
		Address:     endp.Address,
		Port:        endp.Port,
	}

	req := &sdpb.UpdateEndpointRequest{
		Endpoint: endpToUpd,
		UpdateMask: &field_mask.FieldMask{
			Paths: []string{"metadata", "port", "address"},
		},
	}

	_, err := s.client.UpdateEndpoint(ctx, req)
	if err == nil {
		return endp, nil
	}

	// What is the error?
	if err == context.DeadlineExceeded {
		l.Error(err, "timeout expired while waiting for service directory to reply", "timeout-seconds", s.timeout.Seconds())
		return nil, sr.ErrTimeOutExpired
	}

	if status.Code(err) == codes.NotFound {
		return nil, sr.ErrNotFound
	}

	// Any other error
	return nil, err
}

// DeleteEndp deletes the endpoint.
func (s *servDir) DeleteEndp(nsName, servName, endpName string) error {
	// -- Init
	if err := s.checkNames(&nsName, &servName, &endpName); err != nil {
		return err
	}
	l := s.log.WithName("DeleteEndp").WithValues("ns-name", nsName, "serv-name", servName, "endp-name", endpName)
	ctx, canc := context.WithTimeout(s.context, s.timeout)
	defer canc()

	req := &sdpb.DeleteEndpointRequest{
		Name: s.getResourcePath(servDirPath{namespace: nsName, service: servName, endpoint: endpName}),
	}

	err := s.client.DeleteEndpoint(ctx, req)
	if err == nil {
		return nil
	}

	// What is the error?
	if err == context.DeadlineExceeded {
		l.Error(err, "timeout expired while waiting for service directory to reply", "timeout-seconds", s.timeout.Seconds())
		return sr.ErrTimeOutExpired
	}

	if status.Code(err) == codes.NotFound {
		return sr.ErrNotFound
	}

	// Any other error
	return err
}
