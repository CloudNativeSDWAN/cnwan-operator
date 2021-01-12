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

	"go.etcd.io/etcd/mvcc/mvccpb"
	"gopkg.in/yaml.v3"

	sr "github.com/CloudNativeSDWAN/cnwan-operator/pkg/servregistry"
	"github.com/stretchr/testify/assert"
	"go.etcd.io/etcd/clientv3"
)

func TestGetNs(t *testing.T) {
	a := assert.New(t)
	e := &etcdServReg{
		mainCtx: context.Background(),
	}
	unknownErr := fmt.Errorf("unknwon")
	ns := &sr.Namespace{
		Name:     "namespace-name",
		Metadata: map[string]string{"env": "beta"},
	}
	nsBytes, _ := yaml.Marshal(ns)
	cases := []struct {
		get    func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error)
		name   string
		expRes *sr.Namespace
		expErr error
	}{
		{
			name:   "",
			expErr: sr.ErrNsNameNotProvided,
		},
		{
			name: ns.Name,
			get: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
				// errors are already tested in getOne
				return nil, fmt.Errorf("any error")
			},
			expErr: unknownErr,
		},
		{
			name: ns.Name,
			get: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
				return &clientv3.GetResponse{
					Kvs: []*mvccpb.KeyValue{
						{
							Value: nsBytes,
						},
					},
				}, nil
			},
			expRes: ns,
		},
	}

	for i, currCase := range cases {
		f := &fakeKV{
			_get: currCase.get,
		}
		e.kv = f

		var errErr bool
		res, err := e.GetNs(currCase.name)
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

func TestListNs(t *testing.T) {
	a := assert.New(t)
	unknErr := fmt.Errorf("unknown")
	e := &etcdServReg{mainCtx: context.Background()}
	ns := &sr.Namespace{
		Name: "namespace-name",
		Metadata: map[string]string{
			"env": "beta",
		},
	}
	ns2 := &sr.Namespace{
		Name: "namespace-name-2",
		Metadata: map[string]string{
			"env": "prod",
		},
	}
	nsBytes, _ := yaml.Marshal(ns)
	nsBytes2, _ := yaml.Marshal(ns2)
	invalid := []byte(`name: invalid
	name: invalid2`)

	cases := []struct {
		id     string
		get    func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error)
		expRes []*sr.Namespace
		expErr error
	}{
		{
			id: "any-error", // all the specific errors are tested in TestGetList
			get: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
				return nil, fmt.Errorf("any error")
			},
			expErr: unknErr,
		},
		{
			id: "should-marshal-some", // Other test cases are done in TestGetList
			get: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
				return &clientv3.GetResponse{
					Kvs: []*mvccpb.KeyValue{
						{Key: []byte(KeyFromNames(ns.Name).String()), Value: nsBytes},
						{Key: []byte(KeyFromNames(ns2.Name).String()), Value: nsBytes2},
						{Key: []byte(KeyFromNames("invalid").String()), Value: invalid},
					},
				}, nil
			},
			expRes: []*sr.Namespace{ns, ns2},
		},
	}

	for _, currCase := range cases {
		e.kv = &fakeKV{
			_get: currCase.get,
		}

		var errErr bool
		res, err := e.ListNs()

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

func TestCreateNs(t *testing.T) {
	a := assert.New(t)
	e := &etcdServReg{
		mainCtx: context.Background(),
	}
	unknownErr := fmt.Errorf("unknwon")
	ns := &sr.Namespace{
		Name:     "namespace-name",
		Metadata: map[string]string{"env": "beta"},
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
		ns     *sr.Namespace
		commit func() (*clientv3.TxnResponse, error)
		expRes *sr.Namespace
		expErr error
	}{
		{
			ns: ns,
			commit: func() (*clientv3.TxnResponse, error) {
				// All other errors are tested in testPut
				return nil, fmt.Errorf("any error")
			},
			expErr: unknownErr,
		},
		{
			ns: ns,
			commit: func() (*clientv3.TxnResponse, error) {
				return &clientv3.TxnResponse{
					Succeeded: true,
				}, nil
			},
			expRes: ns,
		},
		{
			ns: ns,
			commit: func() (*clientv3.TxnResponse, error) {
				return &clientv3.TxnResponse{
					Succeeded: false,
				}, nil
			},
			expErr: sr.ErrAlreadyExists,
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
		res, err := e.CreateNs(currCase.ns)
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

func TestUpdateNs(t *testing.T) {
	a := assert.New(t)
	e := &etcdServReg{
		mainCtx: context.Background(),
	}
	unknownErr := fmt.Errorf("unknwon")
	ns := &sr.Namespace{
		Name:     "namespace-name",
		Metadata: map[string]string{"env": "beta"},
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
		ns     *sr.Namespace
		commit func() (*clientv3.TxnResponse, error)
		expRes *sr.Namespace
		expErr error
	}{
		{
			ns: ns,
			commit: func() (*clientv3.TxnResponse, error) {
				// All other errors are tested in testPut
				return nil, fmt.Errorf("any error")
			},
			expErr: unknownErr,
		},
		{
			ns: ns,
			commit: func() (*clientv3.TxnResponse, error) {
				return &clientv3.TxnResponse{
					Succeeded: true,
				}, nil
			},
			expRes: ns,
		},
		{
			ns: ns,
			commit: func() (*clientv3.TxnResponse, error) {
				return &clientv3.TxnResponse{
					Succeeded: false,
				}, nil
			},
			expErr: sr.ErrNotFound,
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
		res, err := e.UpdateNs(currCase.ns)
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
