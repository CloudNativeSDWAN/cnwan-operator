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

	sd "cloud.google.com/go/servicedirectory/apiv1beta1"
	gax "github.com/googleapis/gax-go"
	sdpb "google.golang.org/genproto/googleapis/cloud/servicedirectory/v1beta1"
	iampb "google.golang.org/genproto/googleapis/iam/v1"
)

type regClient interface {
	Close() error
	CreateEndpoint(ctx context.Context, req *sdpb.CreateEndpointRequest, opts ...gax.CallOption) (*sdpb.Endpoint, error)
	CreateNamespace(ctx context.Context, req *sdpb.CreateNamespaceRequest, opts ...gax.CallOption) (*sdpb.Namespace, error)
	CreateService(ctx context.Context, req *sdpb.CreateServiceRequest, opts ...gax.CallOption) (*sdpb.Service, error)
	DeleteEndpoint(ctx context.Context, req *sdpb.DeleteEndpointRequest, opts ...gax.CallOption) error
	DeleteNamespace(ctx context.Context, req *sdpb.DeleteNamespaceRequest, opts ...gax.CallOption) error
	DeleteService(ctx context.Context, req *sdpb.DeleteServiceRequest, opts ...gax.CallOption) error
	GetEndpoint(ctx context.Context, req *sdpb.GetEndpointRequest, opts ...gax.CallOption) (*sdpb.Endpoint, error)
	GetIamPolicy(ctx context.Context, req *iampb.GetIamPolicyRequest, opts ...gax.CallOption) (*iampb.Policy, error)
	GetNamespace(ctx context.Context, req *sdpb.GetNamespaceRequest, opts ...gax.CallOption) (*sdpb.Namespace, error)
	GetService(ctx context.Context, req *sdpb.GetServiceRequest, opts ...gax.CallOption) (*sdpb.Service, error)
	ListEndpoints(ctx context.Context, req *sdpb.ListEndpointsRequest, opts ...gax.CallOption) *sd.EndpointIterator
	ListNamespaces(ctx context.Context, req *sdpb.ListNamespacesRequest, opts ...gax.CallOption) *sd.NamespaceIterator
	ListServices(ctx context.Context, req *sdpb.ListServicesRequest, opts ...gax.CallOption) *sd.ServiceIterator
	SetIamPolicy(ctx context.Context, req *iampb.SetIamPolicyRequest, opts ...gax.CallOption) (*iampb.Policy, error)
	TestIamPermissions(ctx context.Context, req *iampb.TestIamPermissionsRequest, opts ...gax.CallOption) (*iampb.TestIamPermissionsResponse, error)
	UpdateEndpoint(ctx context.Context, req *sdpb.UpdateEndpointRequest, opts ...gax.CallOption) (*sdpb.Endpoint, error)
	UpdateNamespace(ctx context.Context, req *sdpb.UpdateNamespaceRequest, opts ...gax.CallOption) (*sdpb.Namespace, error)
	UpdateService(ctx context.Context, req *sdpb.UpdateServiceRequest, opts ...gax.CallOption) (*sdpb.Service, error)
}
