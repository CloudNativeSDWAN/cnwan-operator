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

	"github.com/CloudNativeSDWAN/cnwan-operator/types"
	sdpb "google.golang.org/genproto/googleapis/cloud/servicedirectory/v1beta1"
	"google.golang.org/genproto/protobuf/field_mask"
)

type endpointAction string

var (
	endpointNone   endpointAction = "none"
	endpointDelete endpointAction = "delete"
	endpointUpdate endpointAction = "update"
)

// createEndpoint creates an endpoint in service directory under the provided
// namespace and service. It also takes care of initializing some default
// metadata values.
func (s *sdHandler) createEndpoint(ctx context.Context, nsName, servName string, snapshot types.EndpointSnapshot) (*sdpb.Endpoint, error) {
	parentPath := s.getResourcePath(nsName, servName)

	if snapshot.Metadata == nil {
		snapshot.Metadata = map[string]string{}
	}
	snapshot.Metadata["owner"] = "cnwan-operator"

	resource := &sdpb.Endpoint{
		Name:     snapshot.Name,
		Metadata: snapshot.Metadata,
		Address:  snapshot.Address,
		Port:     snapshot.Port,
	}

	req := &sdpb.CreateEndpointRequest{
		Parent:     parentPath,
		EndpointId: snapshot.Name,
		Endpoint:   resource,
	}

	return s.client.CreateEndpoint(ctx, req)
}

// updateEndpoint updates the provided endpoint in service directory, taking
// care of initializing some default metadata values.
func (s *sdHandler) updateEndpoint(ctx context.Context, endp *sdpb.Endpoint) (*sdpb.Endpoint, error) {
	if endp.Metadata == nil {
		endp.Metadata = map[string]string{}
	}
	endp.Metadata["owner"] = "cnwan-operator"

	req := &sdpb.UpdateEndpointRequest{
		Endpoint: endp,
		UpdateMask: &field_mask.FieldMask{
			Paths: []string{"metadata"},
		},
	}

	return s.client.UpdateEndpoint(ctx, req)
}

// getEndpointAction checks if the endpoint is owned by the operator and,
// if so, returns the action that must be performed as second value.
func (s *sdHandler) getEndpointAction(sdEndp *sdpb.Endpoint, snapEndpoints map[string]types.EndpointSnapshot) (bool, endpointAction) {
	splitName := strings.Split(sdEndp.Name, "/")
	sdName := splitName[len(splitName)-1]

	if sdEndp.Metadata != nil && sdEndp.Metadata["owner"] != "cnwan-operator" {
		// This is not managed by us, better not touch it.
		return false, endpointNone
	}

	// Check if the endpoint still exists in current snapshot
	endpSnap, exists := snapEndpoints[sdName]
	if !exists {
		// This endpoint does not exist in the current snapshot anymore,
		// we gotta delete it.
		return true, endpointDelete
	}

	// Check if the endpoint is different
	if s.deepEqualMetadata(sdEndp.Metadata, endpSnap.Metadata) {
		return true, endpointUpdate
	}

	return true, endpointNone
}
