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

package etcd

import (
	"context"
	"fmt"
	"testing"

	sr "github.com/CloudNativeSDWAN/cnwan-operator/pkg/servregistry"
	"github.com/stretchr/testify/assert"
	clientv3 "go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/etcdserver/api/v3rpc/rpctypes"
	"go.etcd.io/etcd/mvcc/mvccpb"
	"gopkg.in/yaml.v3"
)

func TestNewServiceRegistryWithEtcd(t *testing.T) {
	a := assert.New(t)

	res, err := NewServiceRegistryWithEtcd(context.Background(), nil, nil)
	a.Nil(res)
	a.Equal(err, ErrNilClient)

	prefix := "something"
	res, err = NewServiceRegistryWithEtcd(context.Background(), &clientv3.Client{}, &prefix)
	a.NotNil(res)
	a.Nil(err)
}

func TestGetOne(t *testing.T) {
	a := assert.New(t)
	unknErr := fmt.Errorf("unknown")
	e := &etcdServReg{}
	ns := &sr.Namespace{
		Name: "namespace-name",
		Metadata: map[string]string{
			"env": "beta",
		},
	}
	nsBytes, _ := yaml.Marshal(ns)
	serv := &sr.Service{
		NsName: ns.Name,
		Name:   "service-name",
		Metadata: map[string]string{
			"version": "v0.2.1",
		},
	}
	servBytes, _ := yaml.Marshal(serv)
	endp := &sr.Endpoint{
		NsName:   ns.Name,
		ServName: serv.Name,
		Name:     "endpoint-name",
		Metadata: map[string]string{
			"protocol": "tcp",
		},
	}
	endpBytes, _ := yaml.Marshal(endp)

	cases := []struct {
		get    func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error)
		key    *KeyBuilder
		expObj interface{}
		expErr error
	}{
		{
			key: KeyFromNames(ns.Name),
			get: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
				return nil, rpctypes.ErrGRPCKeyNotFound
			},
			expErr: sr.ErrNotFound,
		},
		{
			key: KeyFromNames(ns.Name),
			get: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
				return nil, fmt.Errorf("any error")
			},
			expErr: unknErr,
		},
		{
			key: KeyFromNames(ns.Name),
			get: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
				return &clientv3.GetResponse{
					Kvs: []*mvccpb.KeyValue{},
				}, nil
			},
			expErr: sr.ErrNotFound,
		},
		{
			key: KeyFromNames(ns.Name),
			get: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
				return &clientv3.GetResponse{
					Kvs: []*mvccpb.KeyValue{
						{
							Value: []byte("invalid"),
						},
					},
				}, nil
			},
			expErr: unknErr,
		},
		{
			key: KeyFromNames(serv.NsName, serv.Name),
			get: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
				return &clientv3.GetResponse{
					Kvs: []*mvccpb.KeyValue{
						{
							Value: []byte("invalid"),
						},
					},
				}, nil
			},
			expErr: unknErr,
		},
		{
			key: KeyFromNames(endp.NsName, endp.ServName, endp.Name),
			get: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
				return &clientv3.GetResponse{
					Kvs: []*mvccpb.KeyValue{
						{
							Value: []byte("invalid"),
						},
					},
				}, nil
			},
			expErr: unknErr,
		},
		{
			key: KeyFromNames(),
			get: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
				return &clientv3.GetResponse{
					Kvs: []*mvccpb.KeyValue{
						{
							Value: []byte("invalid"),
						},
					},
				}, nil
			},
			expErr: ErrUnknownObject,
		},
		{
			key: KeyFromNames(ns.Name),
			get: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
				return &clientv3.GetResponse{
					Kvs: []*mvccpb.KeyValue{
						{
							Value: nsBytes,
						},
					},
				}, nil
			},
			expObj: ns,
		},
		{
			key: KeyFromNames(serv.NsName, serv.Name),
			get: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
				return &clientv3.GetResponse{
					Kvs: []*mvccpb.KeyValue{
						{
							Value: servBytes,
						},
					},
				}, nil
			},
			expObj: serv,
		},
		{
			key: KeyFromNames(endp.NsName, endp.ServName, endp.Name),
			get: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
				return &clientv3.GetResponse{
					Kvs: []*mvccpb.KeyValue{
						{
							Value: endpBytes,
						},
					},
				}, nil
			},
			expObj: endp,
		},
	}

	for i, currCase := range cases {
		f := &fakeKV{
			_get: currCase.get,
		}
		e.kv = f

		var errErr bool
		res, err := e.getOne(context.Background(), currCase.key)

		errRes := a.Equal(currCase.expObj, res)
		if currCase.expErr == unknErr {
			errErr = a.Error(err)
		} else {
			errErr = a.Equal(currCase.expErr, err)
		}

		if !errRes || !errErr {
			a.FailNow(fmt.Sprintf("case %d failed", i))
		}
	}
}
