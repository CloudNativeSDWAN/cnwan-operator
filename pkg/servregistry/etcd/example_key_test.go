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

	sr "github.com/CloudNativeSDWAN/cnwan-operator/pkg/servregistry"
	clientv3 "go.etcd.io/etcd/clientv3"
	namespace "go.etcd.io/etcd/clientv3/namespace"
)

func ExampleKeyBuilder() {
	namespaceName := "ns-name"
	serviceName := "serv-name"
	endpName := "endp-name"
	builder := &KeyBuilder{}

	key := builder.SetNamespace(namespaceName).SetService(serviceName).SetEndpoint(endpName)
	fmt.Println(key)
	// Output: namespaces/ns-name/services/serv-name/endpoints/endp-name
}

func ExampleKeyBuilder_withEtcdNamespace() {
	cli, err := clientv3.New(clientv3.Config{Endpoints: []string{"localhost:2379"}})
	if err != nil {
		// handle error!
	}

	kv := namespace.NewKV(cli.KV, "my-prefix/")
	builder := &KeyBuilder{}
	nsName := "namespace-name"

	// Please look at the service registry package to learn how to get
	// namespaces, services and endpoints from etcd more easily.
	resp, _ := kv.Get(context.TODO(), builder.SetNamespace(nsName).String())
	for _, kvs := range resp.Kvs {
		// Handle the key values here
		_ = kvs
	}
}

func ExampleKeyFromNames() {
	namespaceName := "ns-name"
	serviceName := "serv-name"

	key := KeyFromNames(namespaceName, serviceName)
	fmt.Println(key)
	// Output: namespaces/ns-name/services/serv-name
}

func ExampleKeyFromNames_withEtcdNamespace() {
	cli, err := clientv3.New(clientv3.Config{Endpoints: []string{"localhost:2379"}})
	if err != nil {
		// handle error!
	}

	kv := namespace.NewKV(cli.KV, "my-prefix/")
	nsName := "namespace-name"
	servName := "serv-name"

	// Please look at the service registry package to learn how to get
	// namespaces, services and endpoints from etcd more easily.
	resp, _ := kv.Get(context.TODO(), KeyFromNames(nsName, servName).String())
	for _, kvs := range resp.Kvs {
		// Handle the key values here
		_ = kvs
	}
}

func ExampleKeyFromString_validKey() {
	key := "namespaces/ns-name/services/service-name/endpoints/endpoint-name"

	fmt.Println(KeyFromString(key).IsValid())
	// Output: true
}

func ExampleKeyFromString_invalidKey() {
	key := "/objects/users/user-name"

	fmt.Println(KeyFromString(key).IsValid())
	// Output: false
}

func ExampleKeyFromServiceRegistryObject_validObject() {
	ns := &sr.Namespace{
		Name: "namespace-name",
		Metadata: map[string]string{
			"env": "beta",
		},
	}

	key, err := KeyFromServiceRegistryObject(ns)
	if err != nil {
		// Handle the error...
	}

	fmt.Println(key.String())
	// Output: namespaces/namespace-name
}

func ExampleKeyFromServiceRegistryObject_invalidObject() {
	ns := &sr.Service{
		// A service must belong to a namespace.
		// We comment this to make it an invalid object.
		// NsName : "namespace-name"
		Name: "service-name",
		Metadata: map[string]string{
			"version":     "v0.2.1",
			"commit-hash": "aqvkepsclg",
		},
	}

	key, err := KeyFromServiceRegistryObject(ns)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println(key.String())
	// Output: error: namespace name is empty
}
