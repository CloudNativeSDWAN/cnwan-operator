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
	"testing"

	"github.com/stretchr/testify/assert"
	clientv3 "go.etcd.io/etcd/clientv3"
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
