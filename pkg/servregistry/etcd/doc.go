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

// Package etcd connects to an etcd cluster to perform service registry
// operations.
//
// Read this documentation at
// https://pkg.go.dev/github.com/CloudNativeSDWAN/cnwan-operator/pkg/servregistry/etcd
//
// Etcd as a service registry
//
// etcd is a distributed and reliable key-value store, and while it is
// oblivious of the data you store there, it makes sense to use it as a Service
// Registry: for example, coreDNS can use etcd as a backend where to retrieve
// records from before answering to DNS queries.
//
// Each object inserted to etcd will have a key which identifies it in some way
// and a value with all data that are relevant to the specific object.
//
// To learn more about etcd, see https://etcd.io/.
//
// To learn more about the objects mentioned above you can visit CN-WAN's
// servregistry package
// (https://pkg.go.dev/github.com/CloudNativeSDWAN/cnwan-operator/pkg/servregistry)
// for the technical documentation or CN-WAN Operator's official documentation
// (https://github.com/CloudNativeSDWAN/cnwan-operator).
//
// Values
//
// Values are the service registry objects and can be one of the following:
//
// - Namespace,
//
// - Service,
//
// - or Endpoint.
//
// Please visit the links above to learn how those
// objects are implemented in go and their use/meaning, respectively.
//
// Keys
//
// Being a flat key-value store, there is no real concept of hierarchy.
//
// Thus, given that the objects defined above do need such structure, this will be
// emulated with the well-known use of prefixes, which will make the key
// resemble a path, for example:
// 	/prefix-1/object-1-name/prefix-2/object-2-name
//
// These are the keys that are used by this package:
//
// - namespaces will have keys in the format of
// 	namespaces/<name>
//
// for example:
//	namespaces/my-project
//
// - services will have keys in the format of
// 	namespaces/namespace-name/services/service-name
// for example:
//	namespaces/my-project/services/user-profile
//
// - endpoints will have keys in the format of
// 	namespaces/namespace-name/services/service-name/endpoints/endpoint-name
// for example:
//	namespaces/my-project/services/user-profile/endpoints/user-profile-1
//
// Default global prefix
//
// A sort of "global" prefix can be used: something that specifies that all
// keys that begin with this prefix belong to the service registry. This is
// useful in case you are already using etcd for other purposes or plan to do
// so.
//
// Unless an explicit prefix is passed, this package will use the default one:
//	/service-registry/
// For example, a namespace key will be:
// /service-registry/namespaces/prod
//
// Transactions
//
// Insertions, updates and deletions are all performed in transactions.
//
// Usage
//
// Read the single functions documentation and the example to learn how to use
// this package.
package etcd
