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

package etcd

import (
	"context"
	"fmt"
	"testing"

	"go.etcd.io/etcd/api/v3/etcdserverpb"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"gopkg.in/yaml.v3"

	sr "github.com/CloudNativeSDWAN/cnwan-operator/pkg/servregistry"
	"github.com/stretchr/testify/assert"
)

func TestGetServ(t *testing.T) {
	a := assert.New(t)
	e := &EtcdServReg{
		mainCtx: context.Background(),
	}
	unknownErr := fmt.Errorf("unknwon")
	serv := &sr.Service{
		NsName:   "namespace-name",
		Name:     "service-name",
		Metadata: map[string]string{"v": "v0.2.1"},
	}
	servBytes, _ := yaml.Marshal(serv)
	cases := []struct {
		get    func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error)
		nsName string
		name   string
		expRes *sr.Service
		expErr error
	}{
		{
			expErr: sr.ErrNsNameNotProvided,
		},
		{
			nsName: serv.NsName,
			expErr: sr.ErrServNameNotProvided,
		},
		{
			nsName: serv.NsName,
			name:   serv.Name,
			get: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
				// errors are already tested in getOne
				return nil, fmt.Errorf("any error")
			},
			expErr: unknownErr,
		},
		{
			nsName: serv.NsName,
			name:   serv.Name,
			get: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
				return &clientv3.GetResponse{
					Kvs: []*mvccpb.KeyValue{
						{
							Value: servBytes,
						},
					},
				}, nil
			},
			expRes: serv,
		},
	}

	for i, currCase := range cases {
		f := &fakeKV{
			_get: currCase.get,
		}
		e.kv = f

		var errErr bool
		res, err := e.GetServ(currCase.nsName, currCase.name)
		errRes := a.Equal(currCase.expRes, res)

		if currCase.expErr == unknownErr {
			errErr = a.Error(err)
		} else {
			errErr = a.Equal(currCase.expErr, err)
		}

		if !errRes || !errErr {
			a.FailNow(fmt.Sprintf("case %d failed", i))
		}
	}
}

func TestListServ(t *testing.T) {
	a := assert.New(t)
	unknErr := fmt.Errorf("unknown")
	e := &EtcdServReg{mainCtx: context.Background()}
	serv := &sr.Service{
		NsName: "namespace-name",
		Name:   "service-name",
		Metadata: map[string]string{
			"env": "beta",
		},
	}
	serv2 := &sr.Service{
		NsName: serv.Name,
		Name:   "namespace-name-2",
		Metadata: map[string]string{
			"env": "prod",
		},
	}
	servBytes, _ := yaml.Marshal(serv)
	servBytes2, _ := yaml.Marshal(serv2)
	invalid := []byte(`name: invalid
	name: invalid2`)

	cases := []struct {
		id     string
		nsName string
		get    func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error)
		expRes []*sr.Service
		expErr error
	}{
		{
			id:     "empty-ns-name",
			expErr: sr.ErrNsNameNotProvided,
		},
		{
			id:     "any-error", // all the specific errors are tested in TestGetList
			nsName: serv.NsName,
			get: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
				return nil, fmt.Errorf("any error")
			},
			expErr: unknErr,
		},
		{
			id:     "should-marshal-some",
			nsName: serv.NsName,
			get: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
				return &clientv3.GetResponse{
					Kvs: []*mvccpb.KeyValue{
						{Key: []byte(KeyFromNames(serv.NsName, serv.Name).String()), Value: servBytes},
						{Key: []byte(KeyFromNames(serv2.NsName, serv2.Name).String()), Value: servBytes2},
						{Key: []byte(KeyFromNames("whatever", "invalid").String()), Value: invalid},
					},
				}, nil
			},
			expRes: []*sr.Service{serv, serv2},
		},
	}

	for _, currCase := range cases {
		e.kv = &fakeKV{
			_get: currCase.get,
		}

		var errErr bool
		res, err := e.ListServ(currCase.nsName)

		errRes := a.Equal(currCase.expRes, res)
		if currCase.expErr == unknErr {
			errErr = a.Error(err)
		} else {
			errErr = a.Equal(currCase.expErr, err)
		}

		if !errRes || !errErr {
			a.FailNow(fmt.Sprintf("case %s failed", currCase.id))
		}
	}
}
func TestCreateServ(t *testing.T) {
	a := assert.New(t)
	e := &EtcdServReg{
		mainCtx: context.Background(),
	}
	unknownErr := fmt.Errorf("unknwon")
	serv := &sr.Service{
		NsName:   "namespace-name",
		Name:     "service-name",
		Metadata: map[string]string{"v": "v0.2.0"},
	}
	txn := &fakeTXN{}
	txn._if = func(cs ...clientv3.Cmp) clientv3.Txn {
		return txn
	}
	txn._then = func(ops ...clientv3.Op) clientv3.Txn {
		return txn
	}
	txn._else = func(ops ...clientv3.Op) clientv3.Txn {
		return txn
	}

	cases := []struct {
		serv   *sr.Service
		commit func() (*clientv3.TxnResponse, error)
		expRes *sr.Service
		expErr error
	}{
		{
			serv: serv,
			commit: func() (*clientv3.TxnResponse, error) {
				// All other errors are tested in testPut
				return nil, fmt.Errorf("any error")
			},
			expErr: unknownErr,
		},
		{
			serv: serv,
			commit: func() (*clientv3.TxnResponse, error) {
				return &clientv3.TxnResponse{
					Succeeded: true,
				}, nil
			},
			expRes: serv,
		},
		{
			serv: serv,
			commit: func() (*clientv3.TxnResponse, error) {
				return &clientv3.TxnResponse{
					Succeeded: false,
				}, nil
			},
			expErr: sr.ErrAlreadyExists,
		},
		{
			serv: serv,
			commit: func() (*clientv3.TxnResponse, error) {
				return &clientv3.TxnResponse{
					Succeeded: false,
					Responses: []*etcdserverpb.ResponseOp{
						{
							Response: &etcdserverpb.ResponseOp_ResponseRange{
								ResponseRange: &etcdserverpb.RangeResponse{
									Count: 0,
								},
							},
						},
					},
				}, nil
			},
			expErr: fmt.Errorf("namespace with name %s does not exist", serv.NsName),
		},
		{
			serv: serv,
			commit: func() (*clientv3.TxnResponse, error) {
				return &clientv3.TxnResponse{
					Succeeded: false,
					Responses: []*etcdserverpb.ResponseOp{
						{
							Response: &etcdserverpb.ResponseOp_ResponseRange{
								ResponseRange: &etcdserverpb.RangeResponse{
									Count: 1,
								},
							},
						},
					},
				}, nil
			},
			expErr: unknownErr,
		},
	}

	for i, currCase := range cases {
		f := &fakeKV{}
		f._txn = func(ctx context.Context) clientv3.Txn {
			return txn
		}
		txn._commit = currCase.commit
		e.kv = f

		var errErr bool
		res, err := e.CreateServ(currCase.serv)
		errRes := a.Equal(currCase.expRes, res)

		if currCase.expErr == unknownErr {
			errErr = a.Error(err)
		} else {
			errErr = a.Equal(currCase.expErr, err)
		}

		if !errRes || !errErr {
			a.FailNow(fmt.Sprintf("case %d failed", i))
		}
	}
}

func TestUpdateServ(t *testing.T) {
	a := assert.New(t)
	e := &EtcdServReg{
		mainCtx: context.Background(),
	}
	unknownErr := fmt.Errorf("unknwon")
	serv := &sr.Service{
		NsName:   "namespace-name",
		Name:     "service-name",
		Metadata: map[string]string{"v": "v0.2.0"},
	}
	txn := &fakeTXN{}
	txn._if = func(cs ...clientv3.Cmp) clientv3.Txn {
		return txn
	}
	txn._then = func(ops ...clientv3.Op) clientv3.Txn {
		return txn
	}
	txn._else = func(ops ...clientv3.Op) clientv3.Txn {
		return txn
	}

	cases := []struct {
		serv   *sr.Service
		commit func() (*clientv3.TxnResponse, error)
		expRes *sr.Service
		expErr error
	}{
		{
			serv: serv,
			commit: func() (*clientv3.TxnResponse, error) {
				// All other errors are tested in testPut
				return nil, fmt.Errorf("any error")
			},
			expErr: unknownErr,
		},
		{
			serv: serv,
			commit: func() (*clientv3.TxnResponse, error) {
				return &clientv3.TxnResponse{
					Succeeded: true,
				}, nil
			},
			expRes: serv,
		},
		{
			serv: serv,
			commit: func() (*clientv3.TxnResponse, error) {
				return &clientv3.TxnResponse{
					Succeeded: false,
				}, nil
			},
			expErr: sr.ErrNotFound,
		},
		{
			serv: serv,
			commit: func() (*clientv3.TxnResponse, error) {
				return &clientv3.TxnResponse{
					Succeeded: false,
					Responses: []*etcdserverpb.ResponseOp{
						{
							Response: &etcdserverpb.ResponseOp_ResponseRange{
								ResponseRange: &etcdserverpb.RangeResponse{
									Count: 0,
								},
							},
						},
					},
				}, nil
			},
			expErr: fmt.Errorf("namespace with name %s does not exist", serv.NsName),
		},
		{
			serv: serv,
			commit: func() (*clientv3.TxnResponse, error) {
				return &clientv3.TxnResponse{
					Succeeded: false,
					Responses: []*etcdserverpb.ResponseOp{
						{
							Response: &etcdserverpb.ResponseOp_ResponseRange{
								ResponseRange: &etcdserverpb.RangeResponse{
									Count: 1,
								},
							},
						},
					},
				}, nil
			},
			expErr: unknownErr,
		},
	}

	for i, currCase := range cases {
		f := &fakeKV{}
		f._txn = func(ctx context.Context) clientv3.Txn {
			return txn
		}
		txn._commit = currCase.commit
		e.kv = f

		var errErr bool
		res, err := e.UpdateServ(currCase.serv)
		errRes := a.Equal(currCase.expRes, res)

		if currCase.expErr == unknownErr {
			errErr = a.Error(err)
		} else {
			errErr = a.Equal(currCase.expErr, err)
		}

		if !errRes || !errErr {
			a.FailNow(fmt.Sprintf("case %d failed", i))
		}
	}
}

func TestDeleteServ(t *testing.T) {
	a := assert.New(t)
	e := &EtcdServReg{
		mainCtx: context.Background(),
	}
	txn := &fakeTXN{}
	txn._if = func(cs ...clientv3.Cmp) clientv3.Txn {
		return txn
	}
	txn._then = func(ops ...clientv3.Op) clientv3.Txn {
		return txn
	}
	unknErr := fmt.Errorf("unknown")

	cases := []struct {
		id       string
		nsName   string
		servName string
		commit   func() (*clientv3.TxnResponse, error)
		expErr   error
	}{
		{
			id:     "empty-ns-name",
			expErr: sr.ErrNsNameNotProvided,
		},
		{
			id:     "empty-serv-name",
			nsName: "ns-name",
			expErr: sr.ErrServNameNotProvided,
		},
		{
			id:       "returns-any-error", // specific errors are tested in TestDelete
			nsName:   "ns-name",
			servName: "serv-name",
			commit: func() (*clientv3.TxnResponse, error) {
				return nil, fmt.Errorf("any error")
			},
			expErr: unknErr,
		},
		{
			id:       "is-successful",
			nsName:   "ns-name",
			servName: "serv-name",
			commit: func() (*clientv3.TxnResponse, error) {
				return &clientv3.TxnResponse{
					Succeeded: true,
				}, nil
			},
		},
	}

	for _, currCase := range cases {
		f := &fakeKV{}
		f._txn = func(ctx context.Context) clientv3.Txn {
			return txn
		}
		txn._commit = currCase.commit
		e.kv = f

		var errErr bool
		err := e.DeleteServ(currCase.nsName, currCase.servName)
		if currCase.expErr == unknErr {
			errErr = a.Error(err)
		} else {
			errErr = a.Equal(currCase.expErr, err)
		}

		if !errErr {
			a.FailNow(fmt.Sprintf("case %s failed", currCase.id))
		}
	}
}
