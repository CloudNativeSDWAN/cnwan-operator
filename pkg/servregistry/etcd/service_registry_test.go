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
	"path"
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

func TestGetList(t *testing.T) {
	a := assert.New(t)
	unknErr := fmt.Errorf("unknown")
	e := &etcdServReg{mainCtx: context.Background()}
	nsSearchPref := string(namespacePrefix)
	ns := &sr.Namespace{
		Name: "namespace-name",
		Metadata: map[string]string{
			"env": "beta",
		},
	}
	nsBytes, _ := yaml.Marshal(ns)
	servSearchPref := path.Join(string(namespacePrefix), ns.Name, string(servicePrefix))
	serv := &sr.Service{
		NsName: ns.Name,
		Name:   "service-name",
		Metadata: map[string]string{
			"version": "v0.2.1",
		},
	}
	endpSearchPref := path.Join(string(namespacePrefix), ns.Name, string(servicePrefix), serv.Name, string(endpointPrefix))
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
		id     string
		get    func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error)
		key    *KeyBuilder
		each   func([]byte)
		expErr error
	}{
		{
			id: "nil-key-check-search-prefix",
			get: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
				if !a.Equal(nsSearchPref, key) {
					a.FailNow("case 0: key failed")
				}
				return nil, fmt.Errorf("just to stop the execution")
			},
			expErr: unknErr,
		},
		{
			id:  "not-nil-key-check-search-prefix",
			key: &KeyBuilder{},
			get: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
				if !a.Equal(nsSearchPref, key) {
					a.FailNow("case 1: key failed")
				}
				return nil, fmt.Errorf("just to stop the execution")
			},
			expErr: unknErr,
		},
		{
			id:  "ns-key-check-search-prefix",
			key: KeyFromNames(ns.Name),
			get: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
				if !a.Equal(servSearchPref, key) {
					a.FailNow("case 2: key failed")
				}
				return nil, fmt.Errorf("just to stop the execution")
			},
			expErr: unknErr,
		},
		{
			id:  "serv-key-check-search-prefix",
			key: KeyFromNames(ns.Name, serv.Name),
			get: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
				if !a.Equal(endpSearchPref, key) {
					a.FailNow("case 3: key failed")
				}
				return nil, fmt.Errorf("just to stop the execution")
			},
			expErr: unknErr,
		},
		{
			id:  "should-return-ErrGRPCKeyNotFound",
			key: KeyFromNames(ns.Name),
			get: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
				return nil, rpctypes.ErrGRPCKeyNotFound
			},
			expErr: sr.ErrNotFound,
		},
		{
			id:  "should-return-empty-kvs",
			key: KeyFromNames(ns.Name),
			get: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
				return &clientv3.GetResponse{
					Kvs: []*mvccpb.KeyValue{},
				}, nil
			},
		},
		{
			id: "should-call-each-func-on-ns",
			get: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
				return &clientv3.GetResponse{
					Kvs: []*mvccpb.KeyValue{
						{Key: []byte(KeyFromNames(ns.Name, serv.Name, endp.Name).String()), Value: endpBytes},
						{Key: []byte(KeyFromNames(ns.Name, serv.Name).String()), Value: servBytes},
						{Key: []byte(KeyFromNames(ns.Name).String()), Value: nsBytes},
					},
				}, nil
			},
			each: func(b []byte) {
				if !a.Equal(nsBytes, b) {
					a.FailNow("case 6: object provided is not correct")
				}
			},
		},
		{
			id:  "should-call-each-func-on-ns",
			key: &KeyBuilder{},
			get: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
				return &clientv3.GetResponse{
					Kvs: []*mvccpb.KeyValue{
						{Key: []byte(KeyFromNames(ns.Name, serv.Name, endp.Name).String()), Value: endpBytes},
						{Key: []byte(KeyFromNames(ns.Name, serv.Name).String()), Value: servBytes},
						{Key: []byte(KeyFromNames(ns.Name).String()), Value: nsBytes},
					},
				}, nil
			},
			each: func(b []byte) {
				if !a.Equal(nsBytes, b) {
					a.FailNow("case 7: object provided is not correct")
				}
			},
		},
		{
			id:  "should-call-each-func-on-serv",
			key: KeyFromNames(ns.Name),
			get: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
				return &clientv3.GetResponse{
					Kvs: []*mvccpb.KeyValue{
						{Key: []byte(KeyFromNames(ns.Name, serv.Name, endp.Name).String()), Value: endpBytes},
						{Key: []byte(KeyFromNames(ns.Name).String()), Value: nsBytes},
						{Key: []byte(KeyFromNames(ns.Name, serv.Name).String()), Value: servBytes},
					},
				}, nil
			},
			each: func(b []byte) {
				if !a.Equal(servBytes, b) {
					a.FailNow("case 8: object provided is not correct")
				}
			},
		},
		{
			id:  "should-call-each-func-on-endp",
			key: KeyFromNames(ns.Name, serv.Name),
			get: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
				return &clientv3.GetResponse{
					Kvs: []*mvccpb.KeyValue{
						{Key: []byte(KeyFromNames(ns.Name, serv.Name).String()), Value: servBytes},
						{Key: []byte(KeyFromNames(ns.Name).String()), Value: nsBytes},
						{Key: []byte(KeyFromNames(ns.Name, serv.Name, endp.Name).String()), Value: endpBytes},
					},
				}, nil
			},
			each: func(b []byte) {
				if !a.Equal(endpBytes, b) {
					a.FailNow("case 9: object provided is not correct")
				}
			},
		},
	}

	for i, currCase := range cases {
		e.kv = &fakeKV{
			_get: currCase.get,
		}

		var errErr bool
		err := e.getList(e.mainCtx, currCase.key, currCase.each)
		if currCase.expErr == unknErr {
			errErr = a.Error(err)
		} else {
			errErr = a.Equal(currCase.expErr, err)
		}

		if !errErr {
			a.FailNow(fmt.Sprintf("case %d failed", i))
		}
	}
}
