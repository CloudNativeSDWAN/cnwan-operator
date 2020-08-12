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

	"github.com/CloudNativeSDWAN/cnwan-operator/types"
	sdpb "google.golang.org/genproto/googleapis/cloud/servicedirectory/v1beta1"
	"google.golang.org/genproto/protobuf/field_mask"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// getService tries to load the given service from service directory and
// performs some error checking.
// In case the service does not exist, both returned values are nil.
func (s *sdHandler) getService(ctx context.Context, nsName, servName string) (*sdpb.Service, error) {
	servPath := s.getResourcePath(nsName, servName)

	serv, err := s.client.GetService(ctx, &sdpb.GetServiceRequest{
		Name: servPath,
	})

	if err == nil {
		return serv, nil
	}

	// If it doesn't exist we're returning nil.
	// This will make the code that uses this functions free from having
	// to worry about importing and checking errors, thus cluttering the
	// code too much
	if status.Code(err) == codes.NotFound {
		return nil, nil
	}

	// Every other error
	return nil, err
}

// createService is a convenient method for creating a service, taking care
// of setting some default values in the service's metadata.
func (s *sdHandler) createService(ctx context.Context, snapshot types.ServiceSnapshot) (*sdpb.Service, error) {
	parentPath := s.getResourcePath(snapshot.Namespace)

	if snapshot.Metadata == nil {
		snapshot.Metadata = map[string]string{}
	}
	snapshot.Metadata["owner"] = "cnwan-operator"

	serv := &sdpb.Service{
		Name:     snapshot.Name,
		Metadata: snapshot.Metadata,
	}
	req := &sdpb.CreateServiceRequest{
		Parent:    parentPath,
		ServiceId: snapshot.Name,
		Service:   serv,
	}

	return s.client.CreateService(ctx, req)
}

// updateService is a convenient function for updating a service, taking care
// of setting some default values in the service's metadata
func (s *sdHandler) updateService(ctx context.Context, serv *sdpb.Service) (*sdpb.Service, error) {
	if serv.Metadata == nil {
		serv.Metadata = map[string]string{}
	}
	serv.Metadata["owner"] = "cnwan-operator"

	req := &sdpb.UpdateServiceRequest{
		Service: serv,
		UpdateMask: &field_mask.FieldMask{
			Paths: []string{"metadata"},
		},
	}

	return s.client.UpdateService(ctx, req)
}
