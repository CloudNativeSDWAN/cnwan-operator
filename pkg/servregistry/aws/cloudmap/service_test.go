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
	"fmt"
	"testing"

	sr "github.com/CloudNativeSDWAN/cnwan-operator/pkg/servregistry"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery/types"
	"github.com/stretchr/testify/assert"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestListOrGetService(t *testing.T) {
	listNamespaces := func(ctx context.Context, params *servicediscovery.ListNamespacesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListNamespacesOutput, error) {
		return &servicediscovery.ListNamespacesOutput{
			Namespaces: []types.NamespaceSummary{
				{Arn: aws.String("ns-arn-1"), Id: aws.String("ns-id-1"), Name: aws.String("ns-1")},
				{Arn: aws.String("ns-arn-2"), Id: aws.String("ns-id-2"), Name: aws.String("ns-2")},
				{Arn: aws.String("ns-arn-3"), Id: aws.String("ns-id-3"), Name: aws.String("ns-3")},
			},
		}, nil
	}
	listTagsForResource := func(ctx context.Context, params *servicediscovery.ListTagsForResourceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListTagsForResourceOutput, error) {
		switch aws.ToString(params.ResourceARN) {
		case "ns-arn-1", "ns-arn-2", "ns-arn-3", "serv-arn-1", "serv-arn-2", "serv-arn-3":
			return &servicediscovery.ListTagsForResourceOutput{Tags: []types.Tag{{Key: aws.String("key"), Value: aws.String("value")}}}, nil
		default:
			return nil, fmt.Errorf("whatever-error")
		}
	}

	cases := []struct {
		name   *string
		cli    cloudMapClientIface
		expRes []*sr.Service
		expIDs []*cloudMapIDs
		expErr error
	}{
		{
			cli: &fakeCloudMapClient{
				_ListNamespaces: func(ctx context.Context, params *servicediscovery.ListNamespacesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListNamespacesOutput, error) {
					return nil, fmt.Errorf("whatever-error")
				},
				_ListTagsForResource: listTagsForResource,
			},
			expErr: fmt.Errorf("whatever-error"),
		},
		{
			cli: &fakeCloudMapClient{
				_ListNamespaces: func(ctx context.Context, params *servicediscovery.ListNamespacesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListNamespacesOutput, error) {
					return &servicediscovery.ListNamespacesOutput{Namespaces: []types.NamespaceSummary{}}, nil
				},
				_ListTagsForResource: listTagsForResource,
			},
			expErr: sr.ErrNotFound,
		},
		{
			cli: &fakeCloudMapClient{
				_ListNamespaces:      listNamespaces,
				_ListTagsForResource: listTagsForResource,
				_ListServices: func(ctx context.Context, params *servicediscovery.ListServicesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListServicesOutput, error) {
					return nil, fmt.Errorf("whatever-error")
				},
			},
			expErr: fmt.Errorf("whatever-error"),
		},
		{
			cli: &fakeCloudMapClient{
				_ListNamespaces: listNamespaces,
				_ListServices: func(ctx context.Context, params *servicediscovery.ListServicesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListServicesOutput, error) {
					if len(params.Filters) != 1 {
						return nil, fmt.Errorf("wrong number of filters")
					}
					ffilter := params.Filters[0]
					if ffilter.Name != types.ServiceFilterNameNamespaceId || ffilter.Condition != types.FilterConditionEq || ffilter.Values[0] != "ns-id-1" {
						return nil, fmt.Errorf("wrong filters passed")
					}

					return &servicediscovery.ListServicesOutput{
						Services: []types.ServiceSummary{
							{Arn: aws.String("arn-1"), Id: aws.String("id-1"), Name: aws.String("serv-1")},
							{Arn: aws.String("arn-2"), Id: aws.String("id-2"), Name: aws.String("serv-2")},
							{Arn: aws.String("arn-3"), Id: aws.String("id-3"), Name: aws.String("serv-3")},
						},
					}, nil
				},
				_ListTagsForResource: func(ctx context.Context, params *servicediscovery.ListTagsForResourceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListTagsForResourceOutput, error) {
					return nil, fmt.Errorf("whatever-error")
				},
			},
			expRes: []*sr.Service{
				{Name: "serv-1", NsName: "ns-1", Metadata: map[string]string{}},
				{Name: "serv-2", NsName: "ns-1", Metadata: map[string]string{}},
				{Name: "serv-3", NsName: "ns-1", Metadata: map[string]string{}},
			},
			expIDs: []*cloudMapIDs{
				{arn: "arn-1", id: "id-1"},
				{arn: "arn-2", id: "id-2"},
				{arn: "arn-3", id: "id-3"},
			},
		},
		{
			cli: &fakeCloudMapClient{
				_ListNamespaces: listNamespaces,
				_ListServices: func(ctx context.Context, params *servicediscovery.ListServicesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListServicesOutput, error) {
					if len(params.Filters) != 1 {
						return nil, fmt.Errorf("wrong number of filters")
					}
					ffilter := params.Filters[0]
					if ffilter.Name != types.ServiceFilterNameNamespaceId || ffilter.Condition != types.FilterConditionEq || ffilter.Values[0] != "ns-id-1" {
						return nil, fmt.Errorf("wrong filters passed")
					}

					return &servicediscovery.ListServicesOutput{
						Services: []types.ServiceSummary{
							{Arn: aws.String("serv-arn-1"), Id: aws.String("serv-id-1"), Name: aws.String("serv-1")},
							{Arn: aws.String("serv-arn-2"), Id: aws.String("serv-id-2"), Name: aws.String("serv-2")},
							{Arn: aws.String("serv-arn-3"), Id: aws.String("serv-id-3"), Name: aws.String("serv-3")},
						},
					}, nil
				},
				_ListTagsForResource: listTagsForResource,
			},
			expRes: []*sr.Service{
				{Name: "serv-1", NsName: "ns-1", Metadata: map[string]string{"key": "value"}},
				{Name: "serv-2", NsName: "ns-1", Metadata: map[string]string{"key": "value"}},
				{Name: "serv-3", NsName: "ns-1", Metadata: map[string]string{"key": "value"}},
			},
			expIDs: []*cloudMapIDs{
				{arn: "serv-arn-1", id: "serv-id-1"},
				{arn: "serv-arn-2", id: "serv-id-2"},
				{arn: "serv-arn-3", id: "serv-id-3"},
			},
		},
		{
			name: aws.String("serv-2"),
			cli: &fakeCloudMapClient{
				_ListNamespaces: listNamespaces,
				_ListServices: func(ctx context.Context, params *servicediscovery.ListServicesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListServicesOutput, error) {
					return &servicediscovery.ListServicesOutput{Services: []types.ServiceSummary{
						{Arn: aws.String("serv-arn-1"), Id: aws.String("serv-id-1"), Name: aws.String("serv-1")},
						{Arn: aws.String("serv-arn-2"), Id: aws.String("serv-id-2"), Name: aws.String("serv-2")},
						{Arn: aws.String("serv-arn-3"), Id: aws.String("serv-id-3"), Name: aws.String("serv-3")},
					}}, nil
				},
				_ListTagsForResource: listTagsForResource,
			},

			expRes: []*sr.Service{
				{Name: "serv-2", NsName: "ns-1", Metadata: map[string]string{"key": "value"}},
			},
			expIDs: []*cloudMapIDs{
				{arn: "serv-arn-2", id: "serv-id-2"},
			},
		},
	}

	a := assert.New(t)
	for i, c := range cases {
		h := &Handler{Client: c.cli, mainCtx: context.Background(), log: ctrl.Log.WithName("test")}

		res, ids, err := h.listOrGetService("ns-1", c.name)

		if !a.Equal(c.expRes, res) || !a.Equal(c.expIDs, ids) || !a.Equal(c.expErr, err) {
			a.FailNow("case failed", "case", i)
		}
	}
}

func TestGetServ(t *testing.T) {
	// prepare the testing environment
	listNamespaces := func(ctx context.Context, params *servicediscovery.ListNamespacesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListNamespacesOutput, error) {
		return &servicediscovery.ListNamespacesOutput{Namespaces: []types.NamespaceSummary{
			{Arn: aws.String("ns-arn-1"), Id: aws.String("ns-id-1"), Name: aws.String("ns-1")},
			{Arn: aws.String("ns-arn-2"), Id: aws.String("ns-id-2"), Name: aws.String("ns-2")},
			{Arn: aws.String("ns-arn-3"), Id: aws.String("ns-id-3"), Name: aws.String("ns-3")},
		}}, nil
	}
	listServices := func(ctx context.Context, params *servicediscovery.ListServicesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListServicesOutput, error) {
		return &servicediscovery.ListServicesOutput{Services: []types.ServiceSummary{
			{Arn: aws.String("serv-arn-1"), Id: aws.String("serv-id-1"), Name: aws.String("serv-1")},
			{Arn: aws.String("serv-arn-2"), Id: aws.String("serv-id-2"), Name: aws.String("serv-2")},
			{Arn: aws.String("serv-arn-3"), Id: aws.String("serv-id-3"), Name: aws.String("serv-3")},
		}}, nil
	}
	listTagsForResource := func(ctx context.Context, params *servicediscovery.ListTagsForResourceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListTagsForResourceOutput, error) {
		switch aws.ToString(params.ResourceARN) {
		case "ns-arn-1", "ns-arn-2", "ns-arn-3", "serv-arn-1", "serv-arn-2", "serv-arn-3":
			return &servicediscovery.ListTagsForResourceOutput{Tags: []types.Tag{{Key: aws.String("key"), Value: aws.String("value")}}}, nil
		default:
			return nil, fmt.Errorf("whatever-error")
		}
	}

	cases := []struct {
		nsName   string
		servName string
		cli      cloudMapClientIface
		expRes   *sr.Service
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
			cli: &fakeCloudMapClient{
				_ListNamespaces:      listNamespaces,
				_ListTagsForResource: listTagsForResource,
			},
			expErr: sr.ErrNotFound,
		},
		{
			nsName:   "ns-1",
			servName: "serv-1",
			cli: &fakeCloudMapClient{
				_ListNamespaces:      listNamespaces,
				_ListTagsForResource: listTagsForResource,
				_ListServices: func(ctx context.Context, params *servicediscovery.ListServicesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListServicesOutput, error) {
					return nil, fmt.Errorf("whatever-error")
				},
			},
			expErr: fmt.Errorf("whatever-error"),
		},
		{
			nsName:   "ns-1",
			servName: "serv-4",
			cli: &fakeCloudMapClient{
				_ListNamespaces:      listNamespaces,
				_ListTagsForResource: listTagsForResource,
				_ListServices:        listServices,
			},
			expErr: sr.ErrNotFound,
		},
		{
			nsName:   "ns-1",
			servName: "serv-1",
			cli: &fakeCloudMapClient{
				_ListNamespaces:      listNamespaces,
				_ListTagsForResource: listTagsForResource,
				_ListServices:        listServices,
			},
			expRes: &sr.Service{Name: "serv-1", NsName: "ns-1", Metadata: map[string]string{"key": "value"}},
		},
	}

	a := assert.New(t)
	for i, c := range cases {
		h := &Handler{Client: c.cli, mainCtx: context.Background(), log: ctrl.Log.WithName("test")}

		res, err := h.GetServ(c.nsName, c.servName)

		if !a.Equal(c.expRes, res) || !a.Equal(c.expErr, err) {
			a.FailNow("case failed", "case", i)
		}
	}
}

func TestListServ(t *testing.T) {
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

	cases := []struct {
		nsName string
		cli    cloudMapClientIface
		expRes []*sr.Service
		expErr error
	}{
		{
			expErr: sr.ErrNsNameNotProvided,
		},
		{
			nsName: "ns-1",
			cli: &fakeCloudMapClient{
				_ListNamespaces: func(ctx context.Context, params *servicediscovery.ListNamespacesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListNamespacesOutput, error) {
					return nil, fmt.Errorf("whatever-error")
				},
			},
			expErr: fmt.Errorf("whatever-error"),
		},
		{
			nsName: "ns-4",
			cli: &fakeCloudMapClient{
				_ListNamespaces:      listNamespaces,
				_ListTagsForResource: listTagsForResource,
			},
			expErr: sr.ErrNotFound,
		},
		{
			nsName: "ns-1",
			cli: &fakeCloudMapClient{
				_ListNamespaces:      listNamespaces,
				_ListTagsForResource: listTagsForResource,
				_ListServices: func(ctx context.Context, params *servicediscovery.ListServicesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListServicesOutput, error) {
					return nil, fmt.Errorf("whatever-error")
				},
			},
			expErr: fmt.Errorf("whatever-error"),
		},
		{
			nsName: "ns-1",
			cli: &fakeCloudMapClient{
				_ListNamespaces:      listNamespaces,
				_ListTagsForResource: listTagsForResource,
				_ListServices:        listServices,
			},
			expRes: []*sr.Service{
				{Name: "serv-1", NsName: "ns-1", Metadata: map[string]string{"key": "value"}},
				{Name: "serv-2", NsName: "ns-1", Metadata: map[string]string{"key": "value"}},
			},
		},
	}

	a := assert.New(t)
	for i, c := range cases {
		h := &Handler{Client: c.cli, mainCtx: context.Background(), log: ctrl.Log.WithName("test")}

		res, err := h.ListServ(c.nsName)

		if !a.Equal(c.expRes, res) || !a.Equal(c.expErr, err) {
			a.FailNow("case failed", "case", i)
		}
	}
}

func TestCreateServ(t *testing.T) {
	// prepare the testing environment
	listNamespaces := func(ctx context.Context, params *servicediscovery.ListNamespacesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListNamespacesOutput, error) {
		return &servicediscovery.ListNamespacesOutput{Namespaces: []types.NamespaceSummary{
			{Arn: aws.String("ns-arn-1"), Id: aws.String("ns-id-1"), Name: aws.String("ns-1")},
			{Arn: aws.String("ns-arn-2"), Id: aws.String("ns-id-2"), Name: aws.String("ns-2")},
			{Arn: aws.String("ns-arn-3"), Id: aws.String("ns-id-3"), Name: aws.String("ns-3")},
		}}, nil
	}
	createServ := func(ctx context.Context, params *servicediscovery.CreateServiceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.CreateServiceOutput, error) {
		switch aws.ToString(params.Name) {
		case "serv-1":
			return nil, &types.NamespaceNotFound{}
		case "serv-2":
			return nil, &types.ServiceAlreadyExists{}
		case "serv-3":
			return nil, fmt.Errorf("whatever-error")
		default:
			return &servicediscovery.CreateServiceOutput{}, nil
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

	a := assert.New(t)
	cases := []struct {
		serv   *sr.Service
		cli    cloudMapClientIface
		expRes *sr.Service
		expErr error
	}{
		{
			expErr: sr.ErrServNotProvided,
		},
		{
			serv:   &sr.Service{},
			expErr: sr.ErrNsNameNotProvided,
		},
		{
			serv:   &sr.Service{NsName: "ns-1"},
			expErr: sr.ErrServNameNotProvided,
		},
		{
			serv: &sr.Service{NsName: "ns-1", Name: "serv-1"},
			cli: &fakeCloudMapClient{_ListNamespaces: func(ctx context.Context, params *servicediscovery.ListNamespacesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListNamespacesOutput, error) {
				return nil, fmt.Errorf("whatever-error")
			}, _CreateService: createServ, _ListTagsForResource: listTagsForResource},
			expErr: fmt.Errorf("whatever-error"),
		},
		{
			serv: &sr.Service{NsName: "ns-1", Name: "serv-1"},
			cli: &fakeCloudMapClient{_ListNamespaces: func(ctx context.Context, params *servicediscovery.ListNamespacesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListNamespacesOutput, error) {
				return &servicediscovery.ListNamespacesOutput{Namespaces: []types.NamespaceSummary{}}, nil
			}, _CreateService: createServ, _ListTagsForResource: listTagsForResource},
			expErr: sr.ErrNotFound,
		},
		{
			serv:   &sr.Service{NsName: "ns-1", Name: "serv-1"},
			cli:    &fakeCloudMapClient{_ListNamespaces: listNamespaces, _CreateService: createServ, _ListTagsForResource: listTagsForResource},
			expErr: sr.ErrNotFound,
		},
		{
			serv:   &sr.Service{NsName: "ns-1", Name: "serv-2"},
			cli:    &fakeCloudMapClient{_ListNamespaces: listNamespaces, _CreateService: createServ, _ListTagsForResource: listTagsForResource},
			expErr: sr.ErrAlreadyExists,
		},
		{
			serv:   &sr.Service{NsName: "ns-1", Name: "serv-3"},
			cli:    &fakeCloudMapClient{_ListNamespaces: listNamespaces, _CreateService: createServ, _ListTagsForResource: listTagsForResource},
			expErr: fmt.Errorf("whatever-error"),
		},
		{
			serv: &sr.Service{NsName: "ns-1", Name: "serv-4", Metadata: map[string]string{"key": "value"}},
			cli: &fakeCloudMapClient{
				_ListNamespaces:      listNamespaces,
				_ListTagsForResource: listTagsForResource,
				_CreateService: func(ctx context.Context, params *servicediscovery.CreateServiceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.CreateServiceOutput, error) {
					// check that we are sending the right values
					if aws.ToString(params.Name) != "serv-4" {
						return nil, fmt.Errorf("wrong service name provided")
					}
					if aws.ToString(params.NamespaceId) != "ns-id-1" {
						return nil, fmt.Errorf("wrong namespace id provided")
					}
					if len(params.Tags) != 1 {
						return nil, fmt.Errorf("wrong number of tags provided")
					}
					if params.Type != types.ServiceTypeOptionHttp {
						return nil, fmt.Errorf("wrong service type provided")
					}
					if aws.ToString(params.Tags[0].Key) != "key" || aws.ToString(params.Tags[0].Value) != "value" {
						return nil, fmt.Errorf("wrong tags provided")
					}

					return createServ(ctx, params, optFns...)
				},
			},
			expRes: &sr.Service{NsName: "ns-1", Name: "serv-4", Metadata: map[string]string{"key": "value"}},
		},
	}

	for i, c := range cases {
		h := &Handler{Client: c.cli, mainCtx: context.Background(), log: ctrl.Log.WithName("test")}

		res, err := h.CreateServ(c.serv)

		if !a.Equal(c.expRes, res) || !a.Equal(c.expErr, err) {
			a.FailNow("case failed", "case", i)
		}
	}
}

func TestUpdateServ(t *testing.T) {
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

	a := assert.New(t)
	cases := []struct {
		serv   *sr.Service
		cli    cloudMapClientIface
		expRes *sr.Service
		expErr error
	}{
		{
			expErr: sr.ErrServNotProvided,
		},
		{
			serv:   &sr.Service{},
			expErr: sr.ErrNsNameNotProvided,
		},
		{
			serv:   &sr.Service{NsName: "ns-1"},
			expErr: sr.ErrServNameNotProvided,
		},
		{
			serv: &sr.Service{NsName: "ns-1", Name: "serv-1"},
			cli: &fakeCloudMapClient{_ListNamespaces: func(ctx context.Context, params *servicediscovery.ListNamespacesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListNamespacesOutput, error) {
				return nil, fmt.Errorf("whatever-error")
			}, _ListTagsForResource: listTagsForResource},
			expErr: fmt.Errorf("whatever-error"),
		},
		{
			serv: &sr.Service{NsName: "ns-1", Name: "serv-1"},
			cli: &fakeCloudMapClient{_ListNamespaces: func(ctx context.Context, params *servicediscovery.ListNamespacesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListNamespacesOutput, error) {
				return &servicediscovery.ListNamespacesOutput{Namespaces: []types.NamespaceSummary{}}, nil
			}, _ListTagsForResource: listTagsForResource},
			expErr: sr.ErrNotFound,
		},
		{
			serv: &sr.Service{NsName: "ns-1", Name: "serv-1"},
			cli: &fakeCloudMapClient{_ListNamespaces: listNamespaces, _ListTagsForResource: listTagsForResource,
				_ListServices: func(ctx context.Context, params *servicediscovery.ListServicesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListServicesOutput, error) {
					return nil, fmt.Errorf("whatever-error")
				}},
			expErr: fmt.Errorf("whatever-error"),
		},
		{
			serv:   &sr.Service{NsName: "ns-1", Name: "serv-4"},
			cli:    &fakeCloudMapClient{_ListNamespaces: listNamespaces, _ListTagsForResource: listTagsForResource, _ListServices: listServices},
			expErr: sr.ErrNotFound,
		},
		{
			serv: &sr.Service{NsName: "ns-1", Name: "serv-1", Metadata: map[string]string{"key-1": "value-1"}},
			cli: &fakeCloudMapClient{
				_ListNamespaces:      listNamespaces,
				_ListTagsForResource: listTagsForResource,
				_ListServices:        listServices,
				_TagResource: func(ctx context.Context, params *servicediscovery.TagResourceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.TagResourceOutput, error) {
					if aws.ToString(params.ResourceARN) != "serv-arn-1" {
						return nil, fmt.Errorf("sent-wrong-arn")
					}
					if len(params.Tags) != 1 {
						return nil, fmt.Errorf("wrong-number-of-tags")
					}
					if aws.ToString(params.Tags[0].Key) != "key-1" && aws.ToString(params.Tags[0].Value) != "value-1" &&
						aws.ToString(params.Tags[1].Key) != "key-2" && aws.ToString(params.Tags[1].Value) != "value-2" {
						return nil, fmt.Errorf("wrong-tags")
					}

					return &servicediscovery.TagResourceOutput{}, nil
				}},
			expRes: &sr.Service{NsName: "ns-1", Name: "serv-1", Metadata: map[string]string{"key-1": "value-1"}},
		},
		{
			serv: &sr.Service{NsName: "ns-1", Name: "serv-1"},
			cli: &fakeCloudMapClient{_ListNamespaces: listNamespaces, _ListTagsForResource: listTagsForResource, _ListServices: listServices,
				_TagResource: func(ctx context.Context, params *servicediscovery.TagResourceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.TagResourceOutput, error) {
					return nil, &types.NamespaceNotFound{}
				},
			},
			expErr: sr.ErrNotFound,
		},
		{
			serv: &sr.Service{NsName: "ns-1", Name: "serv-1"},
			cli: &fakeCloudMapClient{_ListNamespaces: listNamespaces, _ListTagsForResource: listTagsForResource, _ListServices: listServices,
				_TagResource: func(ctx context.Context, params *servicediscovery.TagResourceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.TagResourceOutput, error) {
					return nil, &types.ServiceAlreadyExists{}
				},
			},
			expErr: sr.ErrAlreadyExists,
		},
		{
			serv: &sr.Service{NsName: "ns-1", Name: "serv-1"},
			cli: &fakeCloudMapClient{_ListNamespaces: listNamespaces, _ListTagsForResource: listTagsForResource, _ListServices: listServices,
				_TagResource: func(ctx context.Context, params *servicediscovery.TagResourceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.TagResourceOutput, error) {
					return nil, fmt.Errorf("whatever-error")
				},
			},
			expErr: fmt.Errorf("whatever-error"),
		},
	}

	for i, c := range cases {
		h := &Handler{Client: c.cli, mainCtx: context.Background(), log: ctrl.Log.WithName("test")}

		res, err := h.UpdateServ(c.serv)

		if !a.Equal(c.expRes, res) || !a.Equal(c.expErr, err) {
			a.FailNow("case failed", "case", i)
		}
	}
}

func TestDeleteServ(t *testing.T) {
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

	a := assert.New(t)
	cases := []struct {
		nsName   string
		servName string
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
			cli: &fakeCloudMapClient{
				_ListNamespaces: func(ctx context.Context, params *servicediscovery.ListNamespacesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListNamespacesOutput, error) {
					return nil, fmt.Errorf("whatever-error")
				},
			},
			expErr: fmt.Errorf("whatever-error"),
		},
		{
			nsName:   "ns-1",
			servName: "serv-1",
			cli: &fakeCloudMapClient{
				_ListNamespaces: func(ctx context.Context, params *servicediscovery.ListNamespacesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListNamespacesOutput, error) {
					return &servicediscovery.ListNamespacesOutput{Namespaces: []types.NamespaceSummary{}}, nil
				},
			},
			expErr: sr.ErrNotFound,
		},
		{
			nsName:   "ns-1",
			servName: "serv-1",
			cli: &fakeCloudMapClient{
				_ListNamespaces:      listNamespaces,
				_ListTagsForResource: listTagsForResource,
				_ListServices: func(ctx context.Context, params *servicediscovery.ListServicesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListServicesOutput, error) {
					return nil, fmt.Errorf("whatever-error")
				},
			},
			expErr: fmt.Errorf("whatever-error"),
		},
		{
			nsName:   "ns-1",
			servName: "serv-1",
			cli: &fakeCloudMapClient{
				_ListNamespaces:      listNamespaces,
				_ListTagsForResource: listTagsForResource,
				_ListServices: func(ctx context.Context, params *servicediscovery.ListServicesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListServicesOutput, error) {
					return &servicediscovery.ListServicesOutput{Services: []types.ServiceSummary{}}, nil
				},
			},
			expErr: sr.ErrNotFound,
		},
		{
			nsName:   "ns-1",
			servName: "serv-1",
			cli: &fakeCloudMapClient{
				_ListNamespaces:      listNamespaces,
				_ListTagsForResource: listTagsForResource,
				_ListServices:        listServices,
				_DeleteService: func(ctx context.Context, params *servicediscovery.DeleteServiceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.DeleteServiceOutput, error) {
					return nil, fmt.Errorf("whatever-error")
				},
			},
			expErr: fmt.Errorf("whatever-error"),
		},
		{
			nsName:   "ns-1",
			servName: "serv-1",
			cli: &fakeCloudMapClient{
				_ListNamespaces:      listNamespaces,
				_ListTagsForResource: listTagsForResource,
				_ListServices:        listServices,
				_DeleteService: func(ctx context.Context, params *servicediscovery.DeleteServiceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.DeleteServiceOutput, error) {
					return nil, &types.ServiceNotFound{}
				},
			},
			expErr: sr.ErrNotFound,
		},
		{
			nsName:   "ns-1",
			servName: "serv-1",
			cli: &fakeCloudMapClient{
				_ListNamespaces:      listNamespaces,
				_ListTagsForResource: listTagsForResource,
				_ListServices:        listServices,
				_DeleteService: func(ctx context.Context, params *servicediscovery.DeleteServiceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.DeleteServiceOutput, error) {
					if aws.ToString(params.Id) != "serv-id-1" {
						return nil, fmt.Errorf("wrong service id provided")
					}
					return &servicediscovery.DeleteServiceOutput{}, nil
				},
			},
		},
	}

	for i, c := range cases {
		h := &Handler{Client: c.cli, mainCtx: context.Background(), log: ctrl.Log.WithName("test")}

		if !a.Equal(c.expErr, h.DeleteServ(c.nsName, c.servName)) {
			a.FailNow("case failed", "case", i)
		}
	}
}
