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

	"go.etcd.io/etcd/clientv3"
)

// This example shows how to start the etcd service registry
// without a custom global prefix. This means that default one will be used
// (/service-registry/)
func ExampleNewServiceRegistryWithEtcd() {
	clientConfig := clientv3.Config{
		Endpoints: []string{
			"10.11.12.13:2379",
		},
	}

	cli, err := clientv3.New(clientConfig)
	if err != nil {
		fmt.Println("cannot establish connection to etcd:", err)
		os.Exit(1)
	}

	mainCtx, canc := context.WithCancel(context.Background())

	// NewServiceRegistryWithEtcd returns an error only when the client is
	// nil: this is not our case and that's why we do not check the error
	// here.
	servreg := NewServiceRegistryWithEtcd(mainCtx, cli, nil)

	// Do something with the service registry...
	_ = servreg

	// Do other stuf...

	// Cancel the context
	canc()
}

// This example shows how to start the etcd service registry
// with a custom global prefix. As it is shown, you can even use
// multiple slashes.
func ExampleNewServiceRegistryWithEtcd_withPrefix() {
	clientConfig := clientv3.Config{
		Endpoints: []string{
			"10.11.12.13:2379",
		},
	}
	prefix := "/app-1/service-registry/"

	cli, err := clientv3.New(clientConfig)
	if err != nil {
		fmt.Println("cannot establish connection to etcd:", err)
		os.Exit(1)
	}

	mainCtx, canc := context.WithCancel(context.Background())

	// NewServiceRegistryWithEtcd returns an error only when the client is
	// nil: this is not our case and that's why we do not check the error
	// here.
	servreg := NewServiceRegistryWithEtcd(mainCtx, cli, &prefix)

	// Do something with the service registry...
	_ = servreg

	// Do other stuf...

	// Cancel the context
	canc()
}

// This example shows how to start the etcd service registry
// with a no prefix.
// Actually, only "/" will be used in that case.
func ExampleNewServiceRegistryWithEtcd_withEmptyPrefix() {
	clientConfig := clientv3.Config{
		Endpoints: []string{
			"10.11.12.13:2379",
		},
	}
	prefix := ""

	cli, err := clientv3.New(clientConfig)
	if err != nil {
		fmt.Println("cannot establish connection to etcd:", err)
		os.Exit(1)
	}

	mainCtx, canc := context.WithCancel(context.Background())

	// NewServiceRegistryWithEtcd returns an error only when the client is
	// nil: this is not our case and that's why we do not check the error
	// here.
	servreg := NewServiceRegistryWithEtcd(mainCtx, cli, &prefix)

	// Do something with the service registry...
	_ = servreg

	// Do other stuf...

	// Cancel the context
	canc()
}
