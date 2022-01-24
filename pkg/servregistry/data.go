// Copyright © 2020 Cisco
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

package servregistry

// Namespace holds data about a namespace
type Namespace struct {
	// Name of the namespace
	Name string `yaml:"name" json:"name"`
	// Metadata is a key-value map with metadata of the namespace
	Metadata map[string]string `yaml:"metadata" json:"metadata"`
}

// Service holds data about a service
type Service struct {
	// Name of the service
	Name string `yaml:"name" json:"name"`
	// NsName is the name of the namespace that contains this service
	NsName string `yaml:"namespaceName" json:"namespaceName"`
	// Metadata is a key-value map with metadata of this service
	Metadata map[string]string `yaml:"metadata" json:"metadata"`
}

// Endpoint holds data about an endpoint
type Endpoint struct {
	// Name of the endpoint
	Name string `yaml:"name" json:"name"`
	// ServName is the name of the service that contains this endpoint
	ServName string `yaml:"serviceName" json:"serviceName"`
	// NsName is the name of the namespace that contains the service this
	// endpoint belongs to
	NsName string `yaml:"namespaceName" json:"namespaceName"`
	// Metadata is a key-value map with metadata of this endpoint
	Metadata map[string]string `yaml:"metadata" json:"metadata"`
	// Address, i.e. IPv4 or IPv6, of this endpoint
	Address string `yaml:"address" json:"address"`
	// Port of this endpoint
	Port int32 `yaml:"port" json:"port"`
}
