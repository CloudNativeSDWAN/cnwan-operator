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
	"testing"

	sr "github.com/CloudNativeSDWAN/cnwan-operator/pkg/servregistry"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery/types"
	"github.com/stretchr/testify/assert"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestCreateNs(t *testing.T) {
	cases := []struct {
		h      *Handler
		ns     *sr.Namespace
		expRes *sr.Namespace
		expErr error
	}{
		{
			expErr: sr.ErrNsNotProvided,
		},
		{
			ns:     &sr.Namespace{},
			expErr: sr.ErrNsNameNotProvided,
		},
		{
			ns: &sr.Namespace{Name: "whatever"},
			h: &Handler{
				log:     ctrl.Log.WithName("test"),
				mainCtx: context.Background(),
				Client: &fakeCloudMapClient{
					_CreateHttpNamespace: func(ctx context.Context, params *servicediscovery.CreateHttpNamespaceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.CreateHttpNamespaceOutput, error) {
						return nil, &types.NamespaceAlreadyExists{}
					},
				},
			},
			expErr: sr.ErrAlreadyExists,
		},
		{
			ns: &sr.Namespace{Name: "whatever"},
			h: &Handler{
				log:     ctrl.Log.WithName("test"),
				mainCtx: context.Background(),
				Client: &fakeCloudMapClient{
					_CreateHttpNamespace: func(ctx context.Context, params *servicediscovery.CreateHttpNamespaceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.CreateHttpNamespaceOutput, error) {
						return nil, fmt.Errorf("whatever-error")
					},
				},
			},
			expErr: fmt.Errorf("whatever-error"),
		},
		{
			ns: &sr.Namespace{Name: "whatever", Metadata: map[string]string{"key": "val"}},
			h: &Handler{
				log:     ctrl.Log.WithName("test"),
				mainCtx: context.Background(),
				Client: &fakeCloudMapClient{
					_CreateHttpNamespace: func(ctx context.Context, params *servicediscovery.CreateHttpNamespaceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.CreateHttpNamespaceOutput, error) {
						if aws.ToString(params.Name) != "whatever" {
							return nil, fmt.Errorf("provided namespace name is not correct")
						}
						if len(params.Tags) != 1 {
							return nil, fmt.Errorf("provided tags length is not correct")
						}
						if aws.ToString(params.Tags[0].Key) != "key" || aws.ToString(params.Tags[0].Value) != "val" {
							return nil, fmt.Errorf("provided tags are not correct")
						}

						return &servicediscovery.CreateHttpNamespaceOutput{OperationId: aws.String("abc123")}, nil
					},
					_GetOperation: func(ctx context.Context, params *servicediscovery.GetOperationInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.GetOperationOutput, error) {
						if aws.ToString(params.OperationId) != "abc123" {
							return nil, fmt.Errorf("wrong operation id provided")
						}
						return &servicediscovery.GetOperationOutput{Operation: &types.Operation{Status: types.OperationStatusFail, ErrorMessage: aws.String("whatever-error")}}, fmt.Errorf("whatever-error")
					},
				},
			},
			expErr: fmt.Errorf("whatever-error"),
		},
		{
			ns: &sr.Namespace{Name: "whatever", Metadata: map[string]string{"key": "val"}},
			h: &Handler{
				log:     ctrl.Log.WithName("test"),
				mainCtx: context.Background(),
				Client: &fakeCloudMapClient{
					_CreateHttpNamespace: func(ctx context.Context, params *servicediscovery.CreateHttpNamespaceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.CreateHttpNamespaceOutput, error) {
						return &servicediscovery.CreateHttpNamespaceOutput{OperationId: aws.String("abc123")}, nil
					},
					_GetOperation: func(ctx context.Context, params *servicediscovery.GetOperationInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.GetOperationOutput, error) {
						return &servicediscovery.GetOperationOutput{Operation: &types.Operation{Status: types.OperationStatusSuccess}}, nil
					},
				},
			},
			expRes: &sr.Namespace{Name: "whatever", Metadata: map[string]string{"key": "val"}},
		},
	}

	a := assert.New(t)
	for i, c := range cases {
		res, err := c.h.CreateNs(c.ns)
		_ = res

		if !a.Equal(c.expErr, err) {
			a.FailNow("case failed", "case", i)
		}
	}
}

func TestListOrGetNamespace(t *testing.T) {
	cases := []struct {
		cli    cloudMapClientIface
		nsName *string
		expRes []*sr.Namespace
		expIDs []*cloudMapIDs
		expErr error
	}{
		{
			cli: &fakeCloudMapClient{
				_ListNamespaces: func(ctx context.Context, params *servicediscovery.ListNamespacesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListNamespacesOutput, error) {
					return nil, fmt.Errorf("whatever-error")
				},
			},
			expErr: fmt.Errorf("whatever-error"),
		},
		{
			cli: &fakeCloudMapClient{
				_ListNamespaces: func(ctx context.Context, params *servicediscovery.ListNamespacesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListNamespacesOutput, error) {
					if len(params.Filters) != 0 {
						return nil, fmt.Errorf("wrong length of filters provided")
					}

					return &servicediscovery.ListNamespacesOutput{
						Namespaces: []types.NamespaceSummary{
							{Arn: aws.String("arn-1"), Id: aws.String("id-1"), Name: aws.String("ns-1")},
							{Arn: aws.String("arn-2"), Id: aws.String("id-2"), Name: aws.String("ns-2")},
						},
					}, nil
				},
				_ListTagsForResource: func(ctx context.Context, params *servicediscovery.ListTagsForResourceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListTagsForResourceOutput, error) {
					return nil, fmt.Errorf("whatever-error")
				},
			},
			expRes: []*sr.Namespace{
				{
					Name:     "ns-1",
					Metadata: map[string]string{},
				},
				{
					Name:     "ns-2",
					Metadata: map[string]string{},
				},
			},
			expIDs: []*cloudMapIDs{
				{arn: "arn-1", id: "id-1"},
				{arn: "arn-2", id: "id-2"},
			},
		},
		{
			cli: &fakeCloudMapClient{
				_ListNamespaces: func(ctx context.Context, params *servicediscovery.ListNamespacesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListNamespacesOutput, error) {
					return &servicediscovery.ListNamespacesOutput{
						Namespaces: []types.NamespaceSummary{
							{Arn: aws.String("arn-1"), Id: aws.String("id-1"), Name: aws.String("ns-1")},
							{Arn: aws.String("arn-2"), Id: aws.String("id-2"), Name: aws.String("ns-2")},
						},
					}, nil
				},
				_ListTagsForResource: func(ctx context.Context, params *servicediscovery.ListTagsForResourceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListTagsForResourceOutput, error) {
					switch aws.ToString(params.ResourceARN) {
					case "arn-1":
						return &servicediscovery.ListTagsForResourceOutput{Tags: []types.Tag{{Key: aws.String("key-1"), Value: aws.String("val-1")}}}, nil
					case "arn-2":
						return &servicediscovery.ListTagsForResourceOutput{Tags: []types.Tag{{Key: aws.String("key-2"), Value: aws.String("val-2")}}}, nil
					default:
						return nil, fmt.Errorf("whatever-error")
					}
				},
			},
			expRes: []*sr.Namespace{
				{
					Name:     "ns-1",
					Metadata: map[string]string{"key-1": "val-1"},
				},
				{
					Name:     "ns-2",
					Metadata: map[string]string{"key-2": "val-2"},
				},
			},
			expIDs: []*cloudMapIDs{
				{arn: "arn-1", id: "id-1"},
				{arn: "arn-2", id: "id-2"},
			},
		},
		{
			cli: &fakeCloudMapClient{
				_ListNamespaces: func(ctx context.Context, params *servicediscovery.ListNamespacesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListNamespacesOutput, error) {
					return &servicediscovery.ListNamespacesOutput{
						Namespaces: []types.NamespaceSummary{
							{Arn: aws.String("arn-1"), Id: aws.String("id-1"), Name: aws.String("ns-1")},
							{Arn: aws.String("arn-2"), Id: aws.String("id-2"), Name: aws.String("ns-2")},
							{Arn: aws.String("arn-3"), Id: aws.String("id-3"), Name: aws.String("ns-3")},
						},
					}, nil
				},
				_ListTagsForResource: func(ctx context.Context, params *servicediscovery.ListTagsForResourceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListTagsForResourceOutput, error) {
					switch aws.ToString(params.ResourceARN) {
					case "arn-1":
						return &servicediscovery.ListTagsForResourceOutput{Tags: []types.Tag{{Key: aws.String("key-1"), Value: aws.String("val-1")}}}, nil
					case "arn-2":
						return &servicediscovery.ListTagsForResourceOutput{Tags: []types.Tag{{Key: aws.String("key-2"), Value: aws.String("val-2")}}}, nil
					case "arn-3":
						return &servicediscovery.ListTagsForResourceOutput{Tags: []types.Tag{{Key: aws.String("key-3"), Value: aws.String("val-3")}}}, nil
					default:
						return nil, fmt.Errorf("whatever-error")
					}
				},
			},
			expRes: []*sr.Namespace{
				{
					Name:     "ns-2",
					Metadata: map[string]string{"key-2": "val-2"},
				},
			},
			expIDs: []*cloudMapIDs{
				{arn: "arn-2", id: "id-2"},
			},
			nsName: aws.String("ns-2"),
		},
	}

	a := assert.New(t)
	for i, c := range cases {
		h := &Handler{Client: c.cli, mainCtx: context.Background(), log: ctrl.Log.WithName("test")}

		res, ids, err := h.listOrGetNamespace(c.nsName)

		if !a.Equal(c.expRes, res) || !a.Equal(c.expIDs, ids) || !a.Equal(c.expErr, err) {
			a.FailNow("case failed", "case", i)
		}
	}
}

func TestGetNs(t *testing.T) {
	cases := []struct {
		cli    cloudMapClientIface
		name   string
		expRes *sr.Namespace
		expErr error
	}{
		{
			expErr: sr.ErrNsNameNotProvided,
		},
		{
			name: "ns-1",
			cli: &fakeCloudMapClient{
				_ListNamespaces: func(ctx context.Context, params *servicediscovery.ListNamespacesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListNamespacesOutput, error) {
					return nil, fmt.Errorf("whatever-error")
				},
			},
			expErr: fmt.Errorf("whatever-error"),
		},
		{
			name: "ns-1",
			cli: &fakeCloudMapClient{
				_ListNamespaces: func(ctx context.Context, params *servicediscovery.ListNamespacesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListNamespacesOutput, error) {
					return &servicediscovery.ListNamespacesOutput{Namespaces: []types.NamespaceSummary{}}, nil
				},
			},
			expErr: sr.ErrNotFound,
		},
		{
			name: "ns-1",
			cli: &fakeCloudMapClient{
				_ListNamespaces: func(ctx context.Context, params *servicediscovery.ListNamespacesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListNamespacesOutput, error) {
					return &servicediscovery.ListNamespacesOutput{Namespaces: []types.NamespaceSummary{
						{Arn: aws.String("arn-1"), Id: aws.String("id-1"), Name: aws.String("ns-1")},
						{Arn: aws.String("arn-1"), Id: aws.String("id-1"), Name: aws.String("ns-1")},
					}}, nil
				},
				_ListTagsForResource: func(ctx context.Context, params *servicediscovery.ListTagsForResourceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListTagsForResourceOutput, error) {
					switch aws.ToString(params.ResourceARN) {
					case "arn-1":
						return &servicediscovery.ListTagsForResourceOutput{Tags: []types.Tag{{Key: aws.String("key-1"), Value: aws.String("val-1")}}}, nil
					case "arn-2":
						return &servicediscovery.ListTagsForResourceOutput{Tags: []types.Tag{{Key: aws.String("key-2"), Value: aws.String("val-2")}}}, nil
					default:
						return nil, fmt.Errorf("whatever-error")
					}
				},
			},
			expRes: &sr.Namespace{Name: "ns-1", Metadata: map[string]string{"key-1": "val-1"}},
		},
	}

	a := assert.New(t)
	for i, c := range cases {
		h := &Handler{Client: c.cli, mainCtx: context.Background(), log: ctrl.Log.WithName("test")}

		res, err := h.GetNs(c.name)

		if !a.Equal(c.expRes, res) || !a.Equal(c.expErr, err) {
			a.FailNow("case failed", "case", i)
		}
	}
}

func TestUpdateNs(t *testing.T) {
	cases := []struct {
		cli    cloudMapClientIface
		ns     *sr.Namespace
		expRes *sr.Namespace
		expErr error
	}{
		{
			expErr: sr.ErrNsNotProvided,
		},
		{
			ns:     &sr.Namespace{},
			expErr: sr.ErrNsNameNotProvided,
		},
		{
			ns: &sr.Namespace{Name: "ns-1"},
			cli: &fakeCloudMapClient{
				_ListNamespaces: func(ctx context.Context, params *servicediscovery.ListNamespacesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListNamespacesOutput, error) {
					return nil, fmt.Errorf("whatever-error")
				},
			},
			expErr: fmt.Errorf("whatever-error"),
		},
		{
			ns: &sr.Namespace{Name: "ns-1"},
			cli: &fakeCloudMapClient{
				_ListNamespaces: func(ctx context.Context, params *servicediscovery.ListNamespacesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListNamespacesOutput, error) {
					return &servicediscovery.ListNamespacesOutput{Namespaces: []types.NamespaceSummary{}}, nil
				},
			},
			expErr: sr.ErrNotFound,
		},
		{
			ns: &sr.Namespace{Name: "ns-1"},
			cli: &fakeCloudMapClient{
				_ListNamespaces: func(ctx context.Context, params *servicediscovery.ListNamespacesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListNamespacesOutput, error) {
					return &servicediscovery.ListNamespacesOutput{Namespaces: []types.NamespaceSummary{
						{Arn: aws.String("arn-1"), Id: aws.String("id-1"), Name: aws.String("ns-1")},
					}}, nil
				},
				_ListTagsForResource: func(ctx context.Context, params *servicediscovery.ListTagsForResourceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListTagsForResourceOutput, error) {
					return &servicediscovery.ListTagsForResourceOutput{Tags: []types.Tag{{Key: aws.String("key-1"), Value: aws.String("value-1")}}}, nil
				},
				_TagResource: func(ctx context.Context, params *servicediscovery.TagResourceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.TagResourceOutput, error) {
					return nil, &types.ResourceNotFoundException{}
				},
			},
			expErr: sr.ErrNotFound,
		},
		{
			ns: &sr.Namespace{Name: "ns-1", Metadata: map[string]string{"new-key": "new-val"}},
			cli: &fakeCloudMapClient{
				_ListNamespaces: func(ctx context.Context, params *servicediscovery.ListNamespacesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListNamespacesOutput, error) {
					return &servicediscovery.ListNamespacesOutput{Namespaces: []types.NamespaceSummary{
						{Arn: aws.String("arn-1"), Id: aws.String("id-1"), Name: aws.String("ns-1")},
					}}, nil
				},
				_ListTagsForResource: func(ctx context.Context, params *servicediscovery.ListTagsForResourceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListTagsForResourceOutput, error) {
					return &servicediscovery.ListTagsForResourceOutput{Tags: []types.Tag{{Key: aws.String("key-1"), Value: aws.String("value-1")}}}, nil
				},
				_TagResource: func(ctx context.Context, params *servicediscovery.TagResourceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.TagResourceOutput, error) {
					return nil, fmt.Errorf("whatever-error")
				},
			},
			expErr: fmt.Errorf("whatever-error"),
		},
		{
			ns: &sr.Namespace{Name: "ns-1", Metadata: map[string]string{"new-key": "new-val"}},
			cli: &fakeCloudMapClient{
				_ListNamespaces: func(ctx context.Context, params *servicediscovery.ListNamespacesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListNamespacesOutput, error) {
					return &servicediscovery.ListNamespacesOutput{Namespaces: []types.NamespaceSummary{
						{Arn: aws.String("arn-1"), Id: aws.String("id-1"), Name: aws.String("ns-1")},
					}}, nil
				},
				_ListTagsForResource: func(ctx context.Context, params *servicediscovery.ListTagsForResourceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListTagsForResourceOutput, error) {
					return &servicediscovery.ListTagsForResourceOutput{Tags: []types.Tag{{Key: aws.String("key-1"), Value: aws.String("value-1")}}}, nil
				},
				_TagResource: func(ctx context.Context, params *servicediscovery.TagResourceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.TagResourceOutput, error) {
					if aws.ToString(params.ResourceARN) != "arn-1" {
						return nil, fmt.Errorf("wrong arn provided")
					}
					if len(params.Tags) != 1 {
						return nil, fmt.Errorf("wrong length of tags provided")
					}
					if aws.ToString(params.Tags[0].Key) != "new-key" || aws.ToString(params.Tags[0].Value) != "new-val" {
						return nil, fmt.Errorf("wrong tags provided")
					}
					return &servicediscovery.TagResourceOutput{}, nil
				},
			},
			expRes: &sr.Namespace{Name: "ns-1", Metadata: map[string]string{"new-key": "new-val"}},
		},
	}

	a := assert.New(t)
	for i, c := range cases {
		h := &Handler{Client: c.cli, mainCtx: context.Background(), log: ctrl.Log.WithName("test")}

		res, err := h.UpdateNs(c.ns)

		if !a.Equal(c.expRes, res) || !a.Equal(c.expErr, err) {
			a.FailNow("case failed", "case", i)
		}
	}
}

func TestDeleteNs(t *testing.T) {
	cases := []struct {
		cli    cloudMapClientIface
		name   string
		expErr error
	}{
		{
			expErr: sr.ErrNsNameNotProvided,
		},
		{
			name: "ns-1",
			cli: &fakeCloudMapClient{
				_ListNamespaces: func(ctx context.Context, params *servicediscovery.ListNamespacesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListNamespacesOutput, error) {
					return nil, fmt.Errorf("whatever-error")
				},
			},
			expErr: fmt.Errorf("whatever-error"),
		},
		{
			name: "ns-1",
			cli: &fakeCloudMapClient{
				_ListNamespaces: func(ctx context.Context, params *servicediscovery.ListNamespacesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListNamespacesOutput, error) {
					return &servicediscovery.ListNamespacesOutput{Namespaces: []types.NamespaceSummary{}}, nil
				},
			},
			expErr: sr.ErrNotFound,
		},
		{
			name: "ns-1",
			cli: &fakeCloudMapClient{
				_ListNamespaces: func(ctx context.Context, params *servicediscovery.ListNamespacesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListNamespacesOutput, error) {
					return &servicediscovery.ListNamespacesOutput{Namespaces: []types.NamespaceSummary{
						{Arn: aws.String("arn-1"), Id: aws.String("id-1"), Name: aws.String("ns-1")},
					}}, nil
				},
				_ListTagsForResource: func(ctx context.Context, params *servicediscovery.ListTagsForResourceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListTagsForResourceOutput, error) {
					return &servicediscovery.ListTagsForResourceOutput{Tags: []types.Tag{{Key: aws.String("key-1"), Value: aws.String("value-1")}}}, nil
				},
				_DeleteNamespace: func(ctx context.Context, params *servicediscovery.DeleteNamespaceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.DeleteNamespaceOutput, error) {
					return nil, &types.NamespaceNotFound{}
				},
			},
			expErr: sr.ErrNotFound,
		},
		{
			name: "ns-1",
			cli: &fakeCloudMapClient{
				_ListNamespaces: func(ctx context.Context, params *servicediscovery.ListNamespacesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListNamespacesOutput, error) {
					return &servicediscovery.ListNamespacesOutput{Namespaces: []types.NamespaceSummary{
						{Arn: aws.String("arn-1"), Id: aws.String("id-1"), Name: aws.String("ns-1")},
					}}, nil
				},
				_ListTagsForResource: func(ctx context.Context, params *servicediscovery.ListTagsForResourceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListTagsForResourceOutput, error) {
					return &servicediscovery.ListTagsForResourceOutput{Tags: []types.Tag{{Key: aws.String("key-1"), Value: aws.String("value-1")}}}, nil
				},
				_DeleteNamespace: func(ctx context.Context, params *servicediscovery.DeleteNamespaceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.DeleteNamespaceOutput, error) {
					return nil, fmt.Errorf("whatever-error")
				},
			},
			expErr: fmt.Errorf("whatever-error"),
		},
		{
			name: "ns-1",
			cli: &fakeCloudMapClient{
				_ListNamespaces: func(ctx context.Context, params *servicediscovery.ListNamespacesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListNamespacesOutput, error) {
					return &servicediscovery.ListNamespacesOutput{Namespaces: []types.NamespaceSummary{
						{Arn: aws.String("arn-1"), Id: aws.String("id-1"), Name: aws.String("ns-1")},
					}}, nil
				},
				_ListTagsForResource: func(ctx context.Context, params *servicediscovery.ListTagsForResourceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListTagsForResourceOutput, error) {
					return &servicediscovery.ListTagsForResourceOutput{Tags: []types.Tag{{Key: aws.String("key-1"), Value: aws.String("value-1")}}}, nil
				},
				_DeleteNamespace: func(ctx context.Context, params *servicediscovery.DeleteNamespaceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.DeleteNamespaceOutput, error) {
					if aws.ToString(params.Id) != "id-1" {
						return nil, fmt.Errorf("wrong id provided")
					}
					return &servicediscovery.DeleteNamespaceOutput{OperationId: aws.String("op-1")}, nil
				},
				_GetOperation: func(ctx context.Context, params *servicediscovery.GetOperationInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.GetOperationOutput, error) {
					if aws.ToString(params.OperationId) != "op-1" {
						return nil, fmt.Errorf("wrong operation id provided")
					}
					return &servicediscovery.GetOperationOutput{Operation: &types.Operation{Status: types.OperationStatusFail, ErrorMessage: aws.String("whaterver-error")}}, fmt.Errorf("whatever-error")
				},
			},
			expErr: fmt.Errorf("whatever-error"),
		},
		{
			name: "ns-1",
			cli: &fakeCloudMapClient{
				_ListNamespaces: func(ctx context.Context, params *servicediscovery.ListNamespacesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListNamespacesOutput, error) {
					return &servicediscovery.ListNamespacesOutput{Namespaces: []types.NamespaceSummary{
						{Arn: aws.String("arn-1"), Id: aws.String("id-1"), Name: aws.String("ns-1")},
					}}, nil
				},
				_ListTagsForResource: func(ctx context.Context, params *servicediscovery.ListTagsForResourceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListTagsForResourceOutput, error) {
					return &servicediscovery.ListTagsForResourceOutput{Tags: []types.Tag{{Key: aws.String("key-1"), Value: aws.String("value-1")}}}, nil
				},
				_DeleteNamespace: func(ctx context.Context, params *servicediscovery.DeleteNamespaceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.DeleteNamespaceOutput, error) {
					return &servicediscovery.DeleteNamespaceOutput{OperationId: aws.String("op-1")}, nil
				},
				_GetOperation: func(ctx context.Context, params *servicediscovery.GetOperationInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.GetOperationOutput, error) {
					return &servicediscovery.GetOperationOutput{Operation: &types.Operation{Status: types.OperationStatusSuccess}}, nil
				},
			},
		},
	}

	a := assert.New(t)
	for i, c := range cases {
		h := &Handler{Client: c.cli, mainCtx: context.Background(), log: ctrl.Log.WithName("test")}

		if !a.Equal(c.expErr, h.DeleteNs(c.name)) {
			a.FailNow("case failed", "case", i)
		}
	}
}
