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

	sdpb "google.golang.org/genproto/googleapis/cloud/servicedirectory/v1beta1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// getNamespace tries to load the given namespace from service directory
// and performs some error checking.
// In case the service does not exist, both returned values are nil.
func (s *sdHandler) getNamespace(ctx context.Context, name string) (*sdpb.Namespace, error) {
	nsPath := s.getResourcePath(name)

	// Try to get it, first
	ns, err := s.client.GetNamespace(ctx, &sdpb.GetNamespaceRequest{
		Name: nsPath,
	})

	if err == nil {
		return ns, nil
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

// createNamespace creates a namespace in the base path with the given name.
// It is a convenient function that builds the resource before actually
// calling the appropriate service directory method.
func (s *sdHandler) createNamespace(ctx context.Context, name string) (*sdpb.Namespace, error) {
	resource := &sdpb.Namespace{
		Name: name,
		Labels: map[string]string{
			"owner": "cnwan-operator",
		},
	}

	req := &sdpb.CreateNamespaceRequest{
		Parent:      s.getResourcePath(),
		NamespaceId: name,
		Namespace:   resource,
	}

	return s.client.CreateNamespace(ctx, req)
}
