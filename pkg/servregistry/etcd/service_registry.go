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
	"time"

	sr "github.com/CloudNativeSDWAN/cnwan-operator/pkg/servregistry"
	"github.com/go-logr/logr"
	clientv3 "go.etcd.io/etcd/clientv3"
	namespace "go.etcd.io/etcd/clientv3/namespace"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	log logr.Logger
)

const (
	defaultPrefix string = "/service-registry/"
	// timeout used when sending requests
	// TODO: make this configurable or include context explicitly on each
	// method (best way)
	defaultTimeout time.Duration = time.Duration(15) * time.Second
)

func init() {
	log = zap.New(zap.UseDevMode(true)).WithName("etcd")
}

// etcdServReg is a wrap around an etcd client that allows you to perform
// service registry operations on etcd, such as storing, updating, deleting
// or retrieving a namespace, service, or endpoint.
// It is an implementation of `ServiceRegistry` defined in
// github.com/CloudNativeSDWAN/cnwan-operator/pkg/servregistry/.
type etcdServReg struct {
	cli     *clientv3.Client
	kv      clientv3.KV
	prefix  string
	mainCtx context.Context
}

// NewServiceRegistryWithEtcd returns an instance of `ServiceRegistry` with
// ETCD as a backend.
//
// If prefix is not nil, all data will be prefixed with the value you set on
// `prefix`, i.e. `/my-prefix/my-data`. If you don't want any prefix, set the
// value of `prefix` to an empty string or just `/` and all keys will be
// prefixed by just `/`, i.e. `/my-key/`.
// Be careful with this value as it can potentially overwrite existing data.
//
// If context is not nil, it will be used as the main context upon which all
// queries to etcd will be based on.
func NewServiceRegistryWithEtcd(ctx context.Context, cli *clientv3.Client, prefix *string) (sr.ServiceRegistry, error) {
	if cli == nil {
		return nil, ErrNilClient
	}

	c := context.Background()
	if ctx != nil {
		c = ctx
	}

	// Use the default prefix (/service-registry),
	// unless the prefix is not nil, in which case we use that one.
	pref := parsePrefix(prefix)
	kv := namespace.NewKV(cli.KV, pref)

	// TODO: etcdServReg does not implement Service Registry fully yet, it will
	// on future commits
	return &etcdServReg{
		cli:     cli,
		kv:      kv,
		prefix:  pref,
		mainCtx: c,
	}, nil
}
