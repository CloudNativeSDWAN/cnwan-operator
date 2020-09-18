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

package servicedirectory

import (
	"path"

	sr "github.com/CloudNativeSDWAN/cnwan-operator/pkg/servregistry"
)

type servDirPath struct {
	project   string
	region    string
	namespace string
	service   string
	endpoint  string
}

func (s *servDir) getResourcePath(res servDirPath) string {
	resource := ""

	proj := res.project
	if len(proj) == 0 {
		proj = s.project
	}
	resource = path.Join("projects", proj)

	loc := res.region
	if len(loc) == 0 {
		loc = s.region
	}
	resource = path.Join(resource, "locations", loc)

	if len(res.namespace) > 0 {
		resource = path.Join(resource, "namespaces", res.namespace)

		if len(res.service) > 0 {
			resource = path.Join(resource, "services", res.service)

			if len(res.endpoint) > 0 {
				resource = path.Join(resource, "endpoints", res.endpoint)
			}
		}
	}

	return resource
}

func (s *servDir) checkNames(nsName, servName, endpName *string) error {
	if nsName != nil && len(*nsName) == 0 {
		return sr.ErrNsNameNotProvided
	}
	if servName != nil && len(*servName) == 0 {
		return sr.ErrServNameNotProvided
	}
	if endpName != nil && len(*endpName) == 0 {
		return sr.ErrEndpNameNotProvided
	}

	return nil
}
