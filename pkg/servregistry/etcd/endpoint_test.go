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

func TestGetEndp(t *testing.T) {
	a := assert.New(t)
	e := &etcdServReg{
		mainCtx: context.Background(),
	}
	unknownErr := fmt.Errorf("unknwon")
	endp := &sr.Endpoint{
		NsName:   "namespace-name",
		ServName: "service-name",
		Name:     "endpoint-name",
		Metadata: map[string]string{"protocol": "tcp"},
	}
	endpBytes, _ := yaml.Marshal(endp)
	cases := []struct {
		get      func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error)
		nsName   string
		servName string
		name     string
		expRes   *sr.Endpoint
		expErr   error
	}{
		{
			expErr: sr.ErrNsNameNotProvided,
		},
		{
			nsName: endp.NsName,
			expErr: sr.ErrServNameNotProvided,
		},
		{
			nsName:   endp.NsName,
			servName: endp.ServName,
			expErr:   sr.ErrEndpNameNotProvided,
		},
		{
			nsName:   endp.NsName,
			servName: endp.ServName,
			name:     endp.Name,
			get: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
				// errors are already tested in getOne
				return nil, fmt.Errorf("any error")
			},
			expErr: unknownErr,
		},
		{
			nsName:   endp.NsName,
			servName: endp.ServName,
			name:     endp.Name,
			get: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
				return &clientv3.GetResponse{
					Kvs: []*mvccpb.KeyValue{
						{
							Value: endpBytes,
						},
					},
				}, nil
			},
			expRes: endp,
		},
	}

	for i, currCase := range cases {
		f := &fakeKV{
			_get: currCase.get,
		}
		e.kv = f

		var errErr bool
		res, err := e.GetEndp(currCase.nsName, currCase.servName, currCase.name)
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

func TestListEndp(t *testing.T) {
	a := assert.New(t)
	unknErr := fmt.Errorf("unknown")
	e := &etcdServReg{mainCtx: context.Background()}
	endp := &sr.Endpoint{
		NsName:   "namespace-name",
		ServName: "service-name",
		Name:     "endpoint-name",
		Metadata: map[string]string{
			"protocol": "tcp",
		},
	}
	endp2 := &sr.Endpoint{
		NsName:   "namespace-name",
		ServName: "service-name",
		Name:     "endpoint-name-2",
		Metadata: map[string]string{
			"protocol": "udp",
		},
	}
	endpBytes, _ := yaml.Marshal(endp)
	endpBytes2, _ := yaml.Marshal(endp2)
	invalid := []byte(`name: invalid
	name: invalid2`)

	cases := []struct {
		id       string
		nsName   string
		servName string
		get      func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error)
		expRes   []*sr.Endpoint
		expErr   error
	}{
		{
			id:     "empty-ns-name",
			expErr: sr.ErrNsNameNotProvided,
		},
		{
			id:     "empty-serv-name",
			nsName: endp.NsName,
			expErr: sr.ErrServNameNotProvided,
		},
		{
			id:       "any-error", // all the specific errors are tested in TestGetList
			nsName:   endp.NsName,
			servName: endp.ServName,
			get: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
				return nil, fmt.Errorf("any error")
			},
			expErr: unknErr,
		},
		{
			id:       "should-marshal-some", // Other test cases are done in TestGetList
			nsName:   endp.NsName,
			servName: endp.ServName,
			get: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
				return &clientv3.GetResponse{
					Kvs: []*mvccpb.KeyValue{
						{Key: []byte(KeyFromNames(endp.NsName, endp.ServName, endp.Name).String()), Value: endpBytes},
						{Key: []byte(KeyFromNames(endp2.NsName, endp2.ServName, endp2.Name).String()), Value: endpBytes2},
						{Key: []byte(KeyFromNames("whatever", "whatever", "invalid").String()), Value: invalid},
					},
				}, nil
			},
			expRes: []*sr.Endpoint{endp, endp2},
		},
	}

	for _, currCase := range cases {
		e.kv = &fakeKV{
			_get: currCase.get,
		}

		var errErr bool
		res, err := e.ListEndp(currCase.nsName, currCase.servName)

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
