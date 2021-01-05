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
