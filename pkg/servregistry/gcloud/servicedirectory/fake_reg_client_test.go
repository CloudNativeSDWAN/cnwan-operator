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
	"errors"
	"strings"

	sd "cloud.google.com/go/servicedirectory/apiv1"
	gax "github.com/googleapis/gax-go"
	sdpb "google.golang.org/genproto/googleapis/cloud/servicedirectory/v1"
	iampb "google.golang.org/genproto/googleapis/iam/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

type fakeRegClient struct {
}

func getFakeHandler() *Handler {
	return &Handler{
		ProjectID:     "project",
		DefaultRegion: "us",
		Context:       context.Background(),
		Client:        &fakeRegClient{},
		Log:           zap.New(zap.UseDevMode(true)),
	}
}

func (f *fakeRegClient) GetNamespace(ctx context.Context, req *sdpb.GetNamespaceRequest, opts ...gax.CallOption) (*sdpb.Namespace, error) {
	split := strings.Split(req.Name, "/")
	name := split[len(split)-1]
	if name == "get-error" {
		return nil, errors.New("error")
	}

	if name == "get-not-found" {
		return nil, status.Error(codes.NotFound, codes.NotFound.String())
	}

	if name == "timeout-error" {
		return nil, status.Error(codes.DeadlineExceeded, codes.DeadlineExceeded.String())
	}

	return &sdpb.Namespace{Name: "one/two/three/four/five/ns"}, nil
}

func (f *fakeRegClient) ListNamespaces(ctx context.Context, req *sdpb.ListNamespacesRequest, opts ...gax.CallOption) *sd.NamespaceIterator {
	// Not mocked as currently not used by the operator
	return nil
}

func (f *fakeRegClient) CreateNamespace(ctx context.Context, req *sdpb.CreateNamespaceRequest, opts ...gax.CallOption) (*sdpb.Namespace, error) {
	split := strings.Split(req.NamespaceId, "/")
	name := split[len(split)-1]
	if name == "create-error" {
		return nil, errors.New("error")
	}

	if name == "create-exists" {
		return nil, status.Error(codes.AlreadyExists, codes.AlreadyExists.String())
	}

	if name == "timeout-error" {
		return nil, status.Error(codes.DeadlineExceeded, codes.DeadlineExceeded.String())
	}

	return &sdpb.Namespace{Name: "one/two/three/four/five/ns"}, nil
}

func (f *fakeRegClient) UpdateNamespace(ctx context.Context, req *sdpb.UpdateNamespaceRequest, opts ...gax.CallOption) (*sdpb.Namespace, error) {
	split := strings.Split(req.Namespace.Name, "/")
	name := split[len(split)-1]
	if name == "update-error" {
		return nil, errors.New("error")
	}

	if name == "update-not-found" {
		return nil, status.Error(codes.NotFound, codes.NotFound.String())
	}

	if name == "timeout-error" {
		return nil, status.Error(codes.DeadlineExceeded, codes.DeadlineExceeded.String())
	}

	return &sdpb.Namespace{Name: "one/two/three/four/five/ns"}, nil
}

func (f *fakeRegClient) DeleteNamespace(ctx context.Context, req *sdpb.DeleteNamespaceRequest, opts ...gax.CallOption) error {
	split := strings.Split(req.Name, "/")
	name := split[len(split)-1]
	if name == "delete-error" {
		return errors.New("error")
	}

	if name == "delete-not-found" {
		return status.Error(codes.NotFound, codes.NotFound.String())
	}

	if name == "timeout-error" {
		return status.Error(codes.DeadlineExceeded, codes.DeadlineExceeded.String())
	}

	return nil
}

func (f *fakeRegClient) GetService(ctx context.Context, req *sdpb.GetServiceRequest, opts ...gax.CallOption) (*sdpb.Service, error) {
	split := strings.Split(req.Name, "/")
	name := split[len(split)-1]
	if name == "get-error" {
		return nil, errors.New("error")
	}

	if name == "get-not-found" {
		return nil, status.Error(codes.NotFound, codes.NotFound.String())
	}

	if name == "timeout-error" {
		return nil, status.Error(codes.DeadlineExceeded, codes.DeadlineExceeded.String())
	}

	return &sdpb.Service{Name: "one/two/three/four/five/six/seven/" + req.Name}, nil
}

func (f *fakeRegClient) CreateService(ctx context.Context, req *sdpb.CreateServiceRequest, opts ...gax.CallOption) (*sdpb.Service, error) {
	split := strings.Split(req.ServiceId, "/")
	name := split[len(split)-1]
	if name == "create-error" {
		return nil, errors.New("error")
	}

	if name == "create-exists" {
		return nil, status.Error(codes.AlreadyExists, codes.AlreadyExists.String())
	}

	if name == "timeout-error" {
		return nil, status.Error(codes.DeadlineExceeded, codes.DeadlineExceeded.String())
	}

	return &sdpb.Service{Name: "one/two/three/four/five/six/seven/" + req.ServiceId}, nil
}

func (f *fakeRegClient) UpdateService(ctx context.Context, req *sdpb.UpdateServiceRequest, opts ...gax.CallOption) (*sdpb.Service, error) {
	split := strings.Split(req.Service.Name, "/")
	name := split[len(split)-1]
	if name == "update-error" {
		return nil, errors.New("error")
	}

	if name == "update-not-found" {
		return nil, status.Error(codes.NotFound, codes.NotFound.String())
	}

	if name == "timeout-error" {
		return nil, status.Error(codes.DeadlineExceeded, codes.DeadlineExceeded.String())
	}

	return &sdpb.Service{Name: "one/two/three/four/five/ns"}, nil
}

func (f *fakeRegClient) DeleteService(ctx context.Context, req *sdpb.DeleteServiceRequest, opts ...gax.CallOption) error {
	split := strings.Split(req.Name, "/")
	name := split[len(split)-1]
	if name == "delete-error" {
		return errors.New("error")
	}

	if name == "delete-not-found" {
		return status.Error(codes.NotFound, codes.NotFound.String())
	}

	if name == "timeout-error" {
		return status.Error(codes.DeadlineExceeded, codes.DeadlineExceeded.String())
	}

	return nil
}

func (f *fakeRegClient) GetEndpoint(ctx context.Context, req *sdpb.GetEndpointRequest, opts ...gax.CallOption) (*sdpb.Endpoint, error) {
	split := strings.Split(req.Name, "/")
	name := split[len(split)-1]
	if name == "get-error" {
		return nil, errors.New("error")
	}

	if name == "get-not-found" {
		return nil, status.Error(codes.NotFound, codes.NotFound.String())
	}

	if name == "timeout-error" {
		return nil, status.Error(codes.DeadlineExceeded, codes.DeadlineExceeded.String())
	}

	return &sdpb.Endpoint{Name: "one/two/three/four/five/six/seven/eight/nine/" + req.Name}, nil
}

func (f *fakeRegClient) CreateEndpoint(ctx context.Context, req *sdpb.CreateEndpointRequest, opts ...gax.CallOption) (*sdpb.Endpoint, error) {
	split := strings.Split(req.EndpointId, "/")
	name := split[len(split)-1]
	if name == "create-error" {
		return nil, errors.New("error")
	}

	if name == "create-exists" {
		return nil, status.Error(codes.AlreadyExists, codes.AlreadyExists.String())
	}

	if name == "timeout-error" {
		return nil, status.Error(codes.DeadlineExceeded, codes.DeadlineExceeded.String())
	}

	return &sdpb.Endpoint{Name: "one/two/three/four/five/six/seven/eight/nine/" + req.EndpointId}, nil
}

func (f *fakeRegClient) UpdateEndpoint(ctx context.Context, req *sdpb.UpdateEndpointRequest, opts ...gax.CallOption) (*sdpb.Endpoint, error) {
	split := strings.Split(req.Endpoint.Name, "/")
	name := split[len(split)-1]
	if name == "update-error" {
		return nil, errors.New("error")
	}

	if name == "update-not-found" {
		return nil, status.Error(codes.NotFound, codes.NotFound.String())
	}

	if name == "timeout-error" {
		return nil, status.Error(codes.DeadlineExceeded, codes.DeadlineExceeded.String())
	}

	return &sdpb.Endpoint{Name: "one/two/three/four/five/six/seven/eight/nine/" + req.Endpoint.Name}, nil
}

func (f *fakeRegClient) DeleteEndpoint(ctx context.Context, req *sdpb.DeleteEndpointRequest, opts ...gax.CallOption) error {
	split := strings.Split(req.Name, "/")
	name := split[len(split)-1]
	if name == "delete-error" {
		return errors.New("error")
	}

	if name == "delete-not-found" {
		return status.Error(codes.NotFound, codes.NotFound.String())
	}

	if name == "timeout-error" {
		return status.Error(codes.DeadlineExceeded, codes.DeadlineExceeded.String())
	}

	return nil
}

func (f *fakeRegClient) Close() error { return nil }

func (f *fakeRegClient) GetIamPolicy(ctx context.Context, req *iampb.GetIamPolicyRequest, opts ...gax.CallOption) (*iampb.Policy, error) {
	return nil, nil
}

func (f *fakeRegClient) ListEndpoints(ctx context.Context, req *sdpb.ListEndpointsRequest, opts ...gax.CallOption) *sd.EndpointIterator {
	return nil
}

func (f *fakeRegClient) ListServices(ctx context.Context, req *sdpb.ListServicesRequest, opts ...gax.CallOption) *sd.ServiceIterator {
	return nil
}
func (f *fakeRegClient) SetIamPolicy(ctx context.Context, req *iampb.SetIamPolicyRequest, opts ...gax.CallOption) (*iampb.Policy, error) {
	return nil, nil
}
func (f *fakeRegClient) TestIamPermissions(ctx context.Context, req *iampb.TestIamPermissionsRequest, opts ...gax.CallOption) (*iampb.TestIamPermissionsResponse, error) {
	return nil, nil
}
