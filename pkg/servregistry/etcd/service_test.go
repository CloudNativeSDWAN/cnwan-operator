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

func TestGetServ(t *testing.T) {
	a := assert.New(t)
	e := &etcdServReg{
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
	e := &etcdServReg{mainCtx: context.Background()}
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
