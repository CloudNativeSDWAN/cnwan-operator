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
	"fmt"
	"strconv"
	"testing"

	sr "github.com/CloudNativeSDWAN/cnwan-operator/pkg/servregistry"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery/types"
	"github.com/stretchr/testify/assert"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestListOrGetEndpoint(t *testing.T) {
	// prepare the testing environment
	discoverInstances := func(ctx context.Context, params *servicediscovery.DiscoverInstancesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.DiscoverInstancesOutput, error) {
		if aws.ToString(params.NamespaceName) == "ns-0" {
			return nil, &types.NamespaceNotFound{}
		}

		if aws.ToString(params.ServiceName) == "serv-0" {
			return nil, &types.ServiceNotFound{}
		}

		if aws.ToString(params.ServiceName) == "serv-1" {
			return nil, fmt.Errorf("whatever-error")
		}

		if aws.ToString(params.ServiceName) == "serv-2" {
			return &servicediscovery.DiscoverInstancesOutput{Instances: []types.HttpInstanceSummary{
				{
					Attributes:    map[string]string{},
					InstanceId:    aws.String("endp-1"),
					NamespaceName: aws.String("ns-1"),
					ServiceName:   aws.String("serv-2"),
				},
				{
					Attributes:    map[string]string{"AWS_INSTANCE_IPV4": "10.10.10.10"},
					InstanceId:    aws.String("endp-2"),
					NamespaceName: aws.String("ns-1"),
					ServiceName:   aws.String("serv-2"),
				},
				{
					Attributes:    map[string]string{"AWS_INSTANCE_IPV4": "10.10.10.10", "AWS_INSTANCE_PORT": "8080", "key": "value"},
					InstanceId:    aws.String("endp-3"),
					NamespaceName: aws.String("ns-1"),
					ServiceName:   aws.String("serv-2"),
				},
				{
					Attributes:    map[string]string{"AWS_INSTANCE_IPV4": "11.11.11.11", "AWS_INSTANCE_PORT": "8080", "key": "value"},
					InstanceId:    aws.String("endp-4"),
					NamespaceName: aws.String("ns-1"),
					ServiceName:   aws.String("serv-2"),
				},
			}}, nil
		}

		return &servicediscovery.DiscoverInstancesOutput{Instances: []types.HttpInstanceSummary{}}, nil
	}

	a := assert.New(t)
	cases := []struct {
		nsName   string
		servName string
		epName   *string
		cli      cloudMapClientIface
		expRes   []*sr.Endpoint
		expErr   error
	}{
		{
			nsName:   "ns-0",
			servName: "serv-0",
			cli: &fakeCloudMapClient{
				_DiscoverInstances: func(ctx context.Context, params *servicediscovery.DiscoverInstancesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.DiscoverInstancesOutput, error) {
					// check that it really did receive expected values
					if aws.ToString(params.NamespaceName) != "ns-0" || aws.ToString(params.ServiceName) != "serv-0" {
						return nil, fmt.Errorf("unexpected-values")
					}

					return discoverInstances(ctx, params, optFns...)
				},
			},
			expErr: sr.ErrNotFound,
		},
		{
			nsName:   "ns-1",
			servName: "serv-0",
			cli:      &fakeCloudMapClient{_DiscoverInstances: discoverInstances},
			expErr:   sr.ErrNotFound,
		},
		{
			nsName:   "ns-1",
			servName: "serv-1",
			cli:      &fakeCloudMapClient{_DiscoverInstances: discoverInstances},
			expErr:   fmt.Errorf("whatever-error"),
		},
		{
			nsName:   "ns-1",
			servName: "serv-2",
			cli:      &fakeCloudMapClient{_DiscoverInstances: discoverInstances},
			expRes: []*sr.Endpoint{
				{Name: "endp-3", NsName: "ns-1", ServName: "serv-2", Port: 8080, Address: "10.10.10.10", Metadata: map[string]string{"key": "value"}},
				{Name: "endp-4", NsName: "ns-1", ServName: "serv-2", Port: 8080, Address: "11.11.11.11", Metadata: map[string]string{"key": "value"}},
			},
		},
		{
			nsName:   "ns-1",
			servName: "serv-2",
			epName:   aws.String("endp-5"),
			cli:      &fakeCloudMapClient{_DiscoverInstances: discoverInstances},
			expRes:   []*sr.Endpoint{},
		},
		{
			nsName:   "ns-1",
			servName: "serv-2",
			epName:   aws.String("endp-3"),
			cli:      &fakeCloudMapClient{_DiscoverInstances: discoverInstances},
			expRes: []*sr.Endpoint{
				{Name: "endp-3", NsName: "ns-1", ServName: "serv-2", Port: 8080, Address: "10.10.10.10", Metadata: map[string]string{"key": "value"}},
			},
		},
	}

	for i, c := range cases {
		h := &Handler{Client: c.cli, mainCtx: context.Background(), log: ctrl.Log.WithName("test")}

		res, err := h.listOrGetEndpoint(c.nsName, c.servName, c.epName)
		if !a.Equal(c.expRes, res) || !a.Equal(c.expErr, err) {
			a.FailNow("case failed", "case", i)
		}
	}
}

func TestGetEndp(t *testing.T) {
	// prepare the testing environment
	discoverInstances := func(ctx context.Context, params *servicediscovery.DiscoverInstancesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.DiscoverInstancesOutput, error) {
		if aws.ToString(params.NamespaceName) == "ns-0" {
			return nil, &types.NamespaceNotFound{}
		}

		if aws.ToString(params.ServiceName) == "serv-0" {
			return nil, &types.ServiceNotFound{}
		}

		if aws.ToString(params.ServiceName) == "serv-1" {
			return nil, fmt.Errorf("whatever-error")
		}

		if aws.ToString(params.ServiceName) == "serv-2" {
			return &servicediscovery.DiscoverInstancesOutput{Instances: []types.HttpInstanceSummary{
				{
					Attributes:    map[string]string{},
					InstanceId:    aws.String("endp-1"),
					NamespaceName: aws.String("ns-1"),
					ServiceName:   aws.String("serv-2"),
				},
				{
					Attributes:    map[string]string{"AWS_INSTANCE_IPV4": "10.10.10.10"},
					InstanceId:    aws.String("endp-2"),
					NamespaceName: aws.String("ns-1"),
					ServiceName:   aws.String("serv-2"),
				},
				{
					Attributes:    map[string]string{"AWS_INSTANCE_IPV4": "10.10.10.10", "AWS_INSTANCE_PORT": "8080", "key": "value"},
					InstanceId:    aws.String("endp-3"),
					NamespaceName: aws.String("ns-1"),
					ServiceName:   aws.String("serv-2"),
				},
				{
					Attributes:    map[string]string{"AWS_INSTANCE_IPV4": "11.11.11.11", "AWS_INSTANCE_PORT": "8080", "key": "value"},
					InstanceId:    aws.String("endp-4"),
					NamespaceName: aws.String("ns-1"),
					ServiceName:   aws.String("serv-2"),
				},
			}}, nil
		}

		return &servicediscovery.DiscoverInstancesOutput{Instances: []types.HttpInstanceSummary{}}, nil
	}

	a := assert.New(t)
	cases := []struct {
		nsName   string
		servName string
		endpName string
		cli      cloudMapClientIface
		expRes   *sr.Endpoint
		expErr   error
	}{
		{
			expErr: sr.ErrNsNameNotProvided,
		},
		{
			nsName: "ns-0",
			expErr: sr.ErrServNameNotProvided,
		},
		{
			nsName:   "ns-0",
			servName: "serv-0",
			cli: &fakeCloudMapClient{
				_DiscoverInstances: func(ctx context.Context, params *servicediscovery.DiscoverInstancesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.DiscoverInstancesOutput, error) {
					// check that it really did receive expected values
					if aws.ToString(params.NamespaceName) != "ns-0" || aws.ToString(params.ServiceName) != "serv-0" {
						return nil, fmt.Errorf("unexpected-values")
					}

					return discoverInstances(ctx, params, optFns...)
				},
			},
			expErr: sr.ErrNotFound,
		},
		{
			nsName:   "ns-1",
			servName: "serv-0",
			cli:      &fakeCloudMapClient{_DiscoverInstances: discoverInstances},
			expErr:   sr.ErrNotFound,
		},
		{
			nsName:   "ns-1",
			servName: "serv-2",
			endpName: "endp-5",
			cli:      &fakeCloudMapClient{_DiscoverInstances: discoverInstances},
			expErr:   sr.ErrNotFound,
		},
		{
			nsName:   "ns-1",
			servName: "serv-2",
			endpName: "endp-3",
			cli:      &fakeCloudMapClient{_DiscoverInstances: discoverInstances},
			expRes:   &sr.Endpoint{Name: "endp-3", NsName: "ns-1", ServName: "serv-2", Port: 8080, Address: "10.10.10.10", Metadata: map[string]string{"key": "value"}},
		},
	}

	for i, c := range cases {
		h := &Handler{Client: c.cli, mainCtx: context.Background(), log: ctrl.Log.WithName("test")}

		res, err := h.GetEndp(c.nsName, c.servName, c.endpName)
		if !a.Equal(c.expRes, res) || !a.Equal(c.expErr, err) {
			a.FailNow("case failed", "case", i)
		}
	}
}

func TestListEndp(t *testing.T) {
	// prepare the testing environment
	discoverInstances := func(ctx context.Context, params *servicediscovery.DiscoverInstancesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.DiscoverInstancesOutput, error) {
		if aws.ToString(params.NamespaceName) == "ns-0" {
			return nil, &types.NamespaceNotFound{}
		}

		if aws.ToString(params.ServiceName) == "serv-0" {
			return nil, &types.ServiceNotFound{}
		}

		if aws.ToString(params.ServiceName) == "serv-1" {
			return nil, fmt.Errorf("whatever-error")
		}

		if aws.ToString(params.ServiceName) == "serv-2" {
			return &servicediscovery.DiscoverInstancesOutput{Instances: []types.HttpInstanceSummary{
				{
					Attributes:    map[string]string{},
					InstanceId:    aws.String("endp-1"),
					NamespaceName: aws.String("ns-1"),
					ServiceName:   aws.String("serv-2"),
				},
				{
					Attributes:    map[string]string{"AWS_INSTANCE_IPV4": "10.10.10.10"},
					InstanceId:    aws.String("endp-2"),
					NamespaceName: aws.String("ns-1"),
					ServiceName:   aws.String("serv-2"),
				},
				{
					Attributes:    map[string]string{"AWS_INSTANCE_IPV4": "10.10.10.10", "AWS_INSTANCE_PORT": "8080", "key": "value"},
					InstanceId:    aws.String("endp-3"),
					NamespaceName: aws.String("ns-1"),
					ServiceName:   aws.String("serv-2"),
				},
				{
					Attributes:    map[string]string{"AWS_INSTANCE_IPV4": "11.11.11.11", "AWS_INSTANCE_PORT": "8080", "key": "value"},
					InstanceId:    aws.String("endp-4"),
					NamespaceName: aws.String("ns-1"),
					ServiceName:   aws.String("serv-2"),
				},
			}}, nil
		}

		return &servicediscovery.DiscoverInstancesOutput{Instances: []types.HttpInstanceSummary{}}, nil
	}

	a := assert.New(t)
	cases := []struct {
		nsName   string
		servName string
		cli      cloudMapClientIface
		expRes   []*sr.Endpoint
		expErr   error
	}{
		{
			expErr: sr.ErrNsNameNotProvided,
		},
		{
			nsName: "ns-0",
			expErr: sr.ErrServNameNotProvided,
		},
		{
			nsName:   "ns-0",
			servName: "serv-0",
			cli: &fakeCloudMapClient{
				_DiscoverInstances: func(ctx context.Context, params *servicediscovery.DiscoverInstancesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.DiscoverInstancesOutput, error) {
					// check that it really did receive expected values
					if aws.ToString(params.NamespaceName) != "ns-0" || aws.ToString(params.ServiceName) != "serv-0" {
						return nil, fmt.Errorf("unexpected-values")
					}

					return discoverInstances(ctx, params, optFns...)
				},
			},
			expErr: sr.ErrNotFound,
		},
		{
			nsName:   "ns-1",
			servName: "serv-0",
			cli:      &fakeCloudMapClient{_DiscoverInstances: discoverInstances},
			expErr:   sr.ErrNotFound,
		},
		{
			nsName:   "ns-1",
			servName: "serv-1",
			cli:      &fakeCloudMapClient{_DiscoverInstances: discoverInstances},
			expErr:   fmt.Errorf("whatever-error"),
		},
	}

	for i, c := range cases {
		h := &Handler{Client: c.cli, mainCtx: context.Background(), log: ctrl.Log.WithName("test")}

		res, err := h.ListEndp(c.nsName, c.servName)
		if !a.Equal(c.expRes, res) || !a.Equal(c.expErr, err) {
			a.FailNow("case failed", "case", i)
		}
	}
}

func TestCreateEndp(t *testing.T) {
	// prepare the testing environment
	listNamespaces := func(ctx context.Context, params *servicediscovery.ListNamespacesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListNamespacesOutput, error) {
		return &servicediscovery.ListNamespacesOutput{Namespaces: []types.NamespaceSummary{
			{Arn: aws.String("ns-arn-1"), Id: aws.String("ns-id-1"), Name: aws.String("ns-1")},
			{Arn: aws.String("ns-arn-2"), Id: aws.String("ns-id-2"), Name: aws.String("ns-2")},
			{Arn: aws.String("ns-arn-3"), Id: aws.String("ns-id-3"), Name: aws.String("ns-3")},
		}}, nil
	}
	listServices := func(ctx context.Context, params *servicediscovery.ListServicesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListServicesOutput, error) {
		switch params.Filters[0].Values[0] {
		case "ns-id-1":
			return &servicediscovery.ListServicesOutput{Services: []types.ServiceSummary{
				{Arn: aws.String("serv-arn-1"), Id: aws.String("serv-id-1"), Name: aws.String("serv-1")},
				{Arn: aws.String("serv-arn-2"), Id: aws.String("serv-id-2"), Name: aws.String("serv-2")},
			}}, nil
		case "ns-id-2":
			return &servicediscovery.ListServicesOutput{Services: []types.ServiceSummary{
				{Arn: aws.String("serv-arn-3"), Id: aws.String("serv-id-3"), Name: aws.String("serv-3")},
			}}, nil
		default:
			return &servicediscovery.ListServicesOutput{Services: []types.ServiceSummary{}}, nil
		}
	}
	listTagsForResource := func(ctx context.Context, params *servicediscovery.ListTagsForResourceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListTagsForResourceOutput, error) {
		switch aws.ToString(params.ResourceARN) {
		case "ns-arn-1", "ns-arn-2", "ns-arn-3", "serv-arn-1", "serv-arn-2", "serv-arn-3":
			return &servicediscovery.ListTagsForResourceOutput{Tags: []types.Tag{{Key: aws.String("key"), Value: aws.String("value")}}}, nil
		default:
			return nil, fmt.Errorf("whatever-error")
		}
	}

	endp := &sr.Endpoint{NsName: "ns-1", ServName: "serv-1", Name: "endp-1", Address: "10.10.10.10", Port: 80, Metadata: map[string]string{"key": "val"}}
	a := assert.New(t)
	cases := []struct {
		endp   *sr.Endpoint
		cli    cloudMapClientIface
		expRes *sr.Endpoint
		expErr error
	}{
		{
			expErr: sr.ErrEndpNotProvided,
		},
		{
			endp:   &sr.Endpoint{},
			expErr: sr.ErrNsNameNotProvided,
		},
		{
			endp:   &sr.Endpoint{NsName: "ns-1"},
			expErr: sr.ErrServNameNotProvided,
		},
		{
			endp:   &sr.Endpoint{NsName: "ns-1", ServName: "serv-1"},
			expErr: sr.ErrEndpNameNotProvided,
		},
		{
			endp:   &sr.Endpoint{NsName: "ns-1", ServName: "serv-1", Name: "endp-1"},
			expErr: fmt.Errorf("no address provided"),
		},
		{
			endp:   &sr.Endpoint{NsName: "ns-1", ServName: "serv-1", Name: "endp-1", Address: "10.10.10.10"},
			expErr: fmt.Errorf("invalid port provided"),
		},
		{
			endp: endp,
			cli: &fakeCloudMapClient{
				_ListNamespaces: func(ctx context.Context, params *servicediscovery.ListNamespacesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListNamespacesOutput, error) {
					return nil, fmt.Errorf("whatever-error")
				},
			},
			expErr: fmt.Errorf("whatever-error"),
		},
		{
			endp:   &sr.Endpoint{NsName: "ns-4", ServName: "serv-1", Name: "endp-1", Address: "10.10.10.10", Port: 80},
			cli:    &fakeCloudMapClient{_ListNamespaces: listNamespaces, _ListTagsForResource: listTagsForResource, _ListServices: listServices},
			expErr: sr.ErrNotFound,
		},
		{
			endp: endp,
			cli: &fakeCloudMapClient{_ListNamespaces: listNamespaces, _ListTagsForResource: listTagsForResource,
				_ListServices: func(ctx context.Context, params *servicediscovery.ListServicesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListServicesOutput, error) {
					return nil, fmt.Errorf("whatever-error")
				}},
			expErr: fmt.Errorf("whatever-error"),
		},
		{
			endp:   &sr.Endpoint{NsName: "ns-1", ServName: "serv-4", Name: "endp-1", Address: "10.10.10.10", Port: 80},
			cli:    &fakeCloudMapClient{_ListNamespaces: listNamespaces, _ListTagsForResource: listTagsForResource, _ListServices: listServices},
			expErr: sr.ErrNotFound,
		},
		{
			endp: endp,
			cli: &fakeCloudMapClient{_ListNamespaces: listNamespaces, _ListTagsForResource: listTagsForResource, _ListServices: listServices,
				_RegisterInstance: func(ctx context.Context, params *servicediscovery.RegisterInstanceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.RegisterInstanceOutput, error) {
					return nil, fmt.Errorf("whatever-error")
				},
			},
			expErr: fmt.Errorf("whatever-error"),
		},
		{
			endp: endp,
			cli: &fakeCloudMapClient{_ListNamespaces: listNamespaces, _ListTagsForResource: listTagsForResource, _ListServices: listServices,
				_RegisterInstance: func(ctx context.Context, params *servicediscovery.RegisterInstanceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.RegisterInstanceOutput, error) {
					return nil, &types.ServiceNotFound{}
				},
			},
			expErr: sr.ErrNotFound,
		},
		{
			endp: endp,
			cli: &fakeCloudMapClient{_ListNamespaces: listNamespaces, _ListTagsForResource: listTagsForResource, _ListServices: listServices,
				_RegisterInstance: func(ctx context.Context, params *servicediscovery.RegisterInstanceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.RegisterInstanceOutput, error) {
					if aws.ToString(params.InstanceId) != endp.Name {
						return nil, fmt.Errorf("provided endpoint name is not correct")
					}
					if aws.ToString(params.ServiceId) != "serv-id-1" {
						return nil, fmt.Errorf("provided service id is not correct")
					}
					if val := params.Attributes["AWS_INSTANCE_PORT"]; val != strconv.Itoa(int(endp.Port)) {
						return nil, fmt.Errorf("provided endpoint port is not correct")
					}
					if val := params.Attributes["AWS_INSTANCE_IPV4"]; val != "10.10.10.10" {
						return nil, fmt.Errorf("provided endpoint address is not correct")
					}
					if val := params.Attributes["key"]; val != "val" {
						return nil, fmt.Errorf("provided endpoint metadata is not correct")
					}

					return &servicediscovery.RegisterInstanceOutput{OperationId: aws.String("op-id-1")}, nil
				},
				_GetOperation: func(ctx context.Context, params *servicediscovery.GetOperationInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.GetOperationOutput, error) {
					return &servicediscovery.GetOperationOutput{Operation: &types.Operation{Status: types.OperationStatusSuccess}}, nil
				},
			},
			expRes: endp,
		},
		{
			endp: endp,
			cli: &fakeCloudMapClient{_ListNamespaces: listNamespaces, _ListTagsForResource: listTagsForResource, _ListServices: listServices,
				_RegisterInstance: func(ctx context.Context, params *servicediscovery.RegisterInstanceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.RegisterInstanceOutput, error) {
					return &servicediscovery.RegisterInstanceOutput{OperationId: aws.String("op-id-1")}, nil
				},
				_GetOperation: func(ctx context.Context, params *servicediscovery.GetOperationInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.GetOperationOutput, error) {
					return &servicediscovery.GetOperationOutput{Operation: &types.Operation{Status: types.OperationStatusFail, ErrorMessage: aws.String("whatever-error")}}, nil
				},
			},
			expErr: fmt.Errorf("whatever-error"),
		},
	}

	for i, c := range cases {
		h := &Handler{Client: c.cli, mainCtx: context.Background(), log: ctrl.Log.WithName("test")}

		res, err := h.CreateEndp(c.endp)
		if !a.Equal(c.expRes, res) || !a.Equal(c.expErr, err) {
			a.FailNow("case failed", "case", i)
		}
	}
}

func TestDeleteEndp(t *testing.T) {
	// prepare the testing environment
	listNamespaces := func(ctx context.Context, params *servicediscovery.ListNamespacesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListNamespacesOutput, error) {
		return &servicediscovery.ListNamespacesOutput{Namespaces: []types.NamespaceSummary{
			{Arn: aws.String("ns-arn-1"), Id: aws.String("ns-id-1"), Name: aws.String("ns-1")},
			{Arn: aws.String("ns-arn-2"), Id: aws.String("ns-id-2"), Name: aws.String("ns-2")},
			{Arn: aws.String("ns-arn-3"), Id: aws.String("ns-id-3"), Name: aws.String("ns-3")},
		}}, nil
	}
	listServices := func(ctx context.Context, params *servicediscovery.ListServicesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListServicesOutput, error) {
		switch params.Filters[0].Values[0] {
		case "ns-id-1":
			return &servicediscovery.ListServicesOutput{Services: []types.ServiceSummary{
				{Arn: aws.String("serv-arn-1"), Id: aws.String("serv-id-1"), Name: aws.String("serv-1")},
				{Arn: aws.String("serv-arn-2"), Id: aws.String("serv-id-2"), Name: aws.String("serv-2")},
			}}, nil
		case "ns-id-2":
			return &servicediscovery.ListServicesOutput{Services: []types.ServiceSummary{
				{Arn: aws.String("serv-arn-3"), Id: aws.String("serv-id-3"), Name: aws.String("serv-3")},
			}}, nil
		default:
			return &servicediscovery.ListServicesOutput{Services: []types.ServiceSummary{}}, nil
		}
	}
	listTagsForResource := func(ctx context.Context, params *servicediscovery.ListTagsForResourceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListTagsForResourceOutput, error) {
		switch aws.ToString(params.ResourceARN) {
		case "ns-arn-1", "ns-arn-2", "ns-arn-3", "serv-arn-1", "serv-arn-2", "serv-arn-3":
			return &servicediscovery.ListTagsForResourceOutput{Tags: []types.Tag{{Key: aws.String("key"), Value: aws.String("value")}}}, nil
		default:
			return nil, fmt.Errorf("whatever-error")
		}
	}

	endp := &sr.Endpoint{NsName: "ns-1", ServName: "serv-1", Name: "endp-1", Address: "10.10.10.10", Port: 80, Metadata: map[string]string{"key": "val"}}
	a := assert.New(t)
	cases := []struct {
		nsName   string
		servName string
		endpName string
		cli      cloudMapClientIface
		expErr   error
	}{
		{
			expErr: sr.ErrNsNameNotProvided,
		},
		{
			nsName: "ns-1",
			expErr: sr.ErrServNameNotProvided,
		},
		{
			nsName:   "ns-1",
			servName: "serv-1",
			expErr:   sr.ErrEndpNameNotProvided,
		},
		{
			nsName:   "ns-1",
			servName: "serv-1",
			endpName: "endp-1",
			cli: &fakeCloudMapClient{
				_ListNamespaces: func(ctx context.Context, params *servicediscovery.ListNamespacesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListNamespacesOutput, error) {
					return nil, fmt.Errorf("whatever-error")
				},
			},
			expErr: fmt.Errorf("whatever-error"),
		},
		{
			nsName:   "ns-4",
			servName: "serv-1",
			endpName: "endp-1",
			cli:      &fakeCloudMapClient{_ListNamespaces: listNamespaces, _ListTagsForResource: listTagsForResource, _ListServices: listServices},
			expErr:   sr.ErrNotFound,
		},
		{
			nsName:   "ns-1",
			servName: "serv-1",
			endpName: "endp-1",
			cli: &fakeCloudMapClient{_ListNamespaces: listNamespaces, _ListTagsForResource: listTagsForResource,
				_ListServices: func(ctx context.Context, params *servicediscovery.ListServicesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListServicesOutput, error) {
					return nil, fmt.Errorf("whatever-error")
				}},
			expErr: fmt.Errorf("whatever-error"),
		},
		{
			nsName:   "ns-1",
			servName: "serv-5",
			endpName: "endp-1",
			cli:      &fakeCloudMapClient{_ListNamespaces: listNamespaces, _ListTagsForResource: listTagsForResource, _ListServices: listServices},
			expErr:   sr.ErrNotFound,
		},
		{
			nsName:   "ns-1",
			servName: "serv-1",
			endpName: "endp-1",
			cli: &fakeCloudMapClient{_ListNamespaces: listNamespaces, _ListTagsForResource: listTagsForResource, _ListServices: listServices,
				_DeregisterInstance: func(ctx context.Context, params *servicediscovery.DeregisterInstanceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.DeregisterInstanceOutput, error) {
					return nil, fmt.Errorf("whatever-error")
				},
			},
			expErr: fmt.Errorf("whatever-error"),
		},
		{
			nsName:   "ns-1",
			servName: "serv-1",
			endpName: "endp-1",
			cli: &fakeCloudMapClient{_ListNamespaces: listNamespaces, _ListTagsForResource: listTagsForResource, _ListServices: listServices,
				_DeregisterInstance: func(ctx context.Context, params *servicediscovery.DeregisterInstanceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.DeregisterInstanceOutput, error) {
					return nil, &types.ServiceNotFound{}
				},
			},
			expErr: sr.ErrNotFound,
		},
		{
			nsName:   "ns-1",
			servName: "serv-1",
			endpName: "endp-1",
			cli: &fakeCloudMapClient{_ListNamespaces: listNamespaces, _ListTagsForResource: listTagsForResource, _ListServices: listServices,
				_DeregisterInstance: func(ctx context.Context, params *servicediscovery.DeregisterInstanceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.DeregisterInstanceOutput, error) {
					return nil, &types.InstanceNotFound{}
				},
			},
			expErr: sr.ErrNotFound,
		},
		{
			nsName:   "ns-1",
			servName: "serv-1",
			endpName: "endp-1",
			cli: &fakeCloudMapClient{_ListNamespaces: listNamespaces, _ListTagsForResource: listTagsForResource, _ListServices: listServices,
				_DeregisterInstance: func(ctx context.Context, params *servicediscovery.DeregisterInstanceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.DeregisterInstanceOutput, error) {
					if aws.ToString(params.InstanceId) != endp.Name {
						return nil, fmt.Errorf("provided endpoint name is not correct")
					}
					if aws.ToString(params.ServiceId) != "serv-id-1" {
						return nil, fmt.Errorf("provided service id is not correct")
					}

					return &servicediscovery.DeregisterInstanceOutput{OperationId: aws.String("op-id-1")}, nil
				},
				_GetOperation: func(ctx context.Context, params *servicediscovery.GetOperationInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.GetOperationOutput, error) {
					return &servicediscovery.GetOperationOutput{Operation: &types.Operation{Status: types.OperationStatusSuccess}}, nil
				},
			},
		},
		{
			nsName:   "ns-1",
			servName: "serv-1",
			endpName: "endp-1",
			cli: &fakeCloudMapClient{_ListNamespaces: listNamespaces, _ListTagsForResource: listTagsForResource, _ListServices: listServices,
				_DeregisterInstance: func(ctx context.Context, params *servicediscovery.DeregisterInstanceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.DeregisterInstanceOutput, error) {
					return &servicediscovery.DeregisterInstanceOutput{OperationId: aws.String("op-id-1")}, nil
				},
				_GetOperation: func(ctx context.Context, params *servicediscovery.GetOperationInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.GetOperationOutput, error) {
					return &servicediscovery.GetOperationOutput{Operation: &types.Operation{Status: types.OperationStatusFail, ErrorMessage: aws.String("whatever-error")}}, nil
				},
			},
			expErr: fmt.Errorf("whatever-error"),
		},
	}

	for i, c := range cases {
		h := &Handler{Client: c.cli, mainCtx: context.Background(), log: ctrl.Log.WithName("test")}

		if !a.Equal(c.expErr, h.DeleteEndp(c.nsName, c.servName, c.endpName)) {
			a.FailNow("case failed", "case", i)
		}
	}
}
