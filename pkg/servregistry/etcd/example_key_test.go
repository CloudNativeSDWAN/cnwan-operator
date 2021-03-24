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
	"os"

	sr "github.com/CloudNativeSDWAN/cnwan-operator/pkg/servregistry"
	clientv3 "go.etcd.io/etcd/clientv3"
	namespace "go.etcd.io/etcd/clientv3/namespace"
)

// This example shows how to use the Keybuilder without any names yet.
// It is useful in case you want to build a key based on some conditions.
//
// In this very simple example, an environment variable is set to drive
// the conditions on how the key should be built.
func ExampleKeyBuilder() {
	namespaceName := "ns-name"
	serviceName := "serv-name"
	endpName := "endp-name"
	builder := &KeyBuilder{}

	os.Setenv("GET", "endpoint")
	builder.SetNamespace(namespaceName).SetService(serviceName)

	if os.Getenv("GET") == "endpoint" {
		builder.SetEndpoint(endpName)
	}
	fmt.Println(builder)
	// Output: namespaces/ns-name/services/serv-name/endpoints/endp-name
}

// This example shows how to use this package's KeyBuilder to build keys
// for etcd to use for operations that are not supported by this package,
// i.e. watching.
func ExampleKeyBuilder_withUnsupportedOperations() {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints: []string{
			"localhost:2379",
		},
	})
	if err != nil {
		fmt.Println("cannot establish connection to etcd:", err)
		os.Exit(1)
	}

	watcher := namespace.NewWatcher(cli.Watcher, "my-prefix/")
	builder := &KeyBuilder{}
	nsName := "namespace-name"

	ctx, canc := context.WithCancel(context.TODO())
	defer canc()
	wchan := watcher.Watch(ctx, builder.SetNamespace(nsName).String())
	for {
		w := <-wchan
		if w.Canceled {
			break
		}
	}
}

// This example shows how to build a key starting from a list of names.
func ExampleKeyFromNames() {
	namespaceName := "ns-name"
	serviceName := "serv-name"

	key := KeyFromNames(namespaceName, serviceName)
	fmt.Println(key)
	// Output: namespaces/ns-name/services/serv-name
}

func ExampleKeyFromString_validKey() {
	key := "namespaces/ns-name/services/service-name/endpoints/endpoint-name"

	fmt.Println(KeyFromString(key).IsValid())
	// Output: true
}

func ExampleKeyFromString_validKeyWithPrefix() {
	key := "/my/prefix/is/long/namespaces/ns-name/services/service-name/endpoints/endpoint-name"

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
