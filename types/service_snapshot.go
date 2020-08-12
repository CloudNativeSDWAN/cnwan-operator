// Copyright © 2020 Cisco
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

package types

// ServiceSnapshot represents the current state of a service
type ServiceSnapshot struct {
	Name      string
	Namespace string
	Metadata  map[string]string
	Endpoints map[string]EndpointSnapshot
}

// EndpointSnapshot represents the current state of an endpoint
type EndpointSnapshot struct {
	Name     string
	Metadata map[string]string
	Address  string
	Port     int32
}
