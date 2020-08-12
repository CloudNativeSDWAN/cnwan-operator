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
	"testing"

	"github.com/CloudNativeSDWAN/cnwan-operator/types"
	"github.com/spf13/viper"
	. "github.com/stretchr/testify/assert"
)

func TestGetResourcePath(t *testing.T) {
	project, location, ns, serv, endp := "my-project", "my-location", "my-ns", "my-serv", "my-endp"
	viper.Set(types.SDProject, project)
	viper.Set(types.SDDefaultRegion, location)
	basePath := path.Join("projects", project, "locations", location)
	s := &sdHandler{}

	// Case 1: empty
	res := s.getResourcePath()
	NotEmpty(t, res)
	Equal(t, basePath, res)

	// Case 2: only namespace
	res = s.getResourcePath(ns)
	expected := path.Join(basePath, "namespaces", ns)
	Equal(t, expected, res)

	// Case 3: service
	res = s.getResourcePath(ns, serv)
	expected = path.Join(expected, "services", serv)
	Equal(t, expected, res)

	// Case 4: endpoint
	res = s.getResourcePath(ns, serv, endp)
	expected = path.Join(expected, "endpoints", endp)
	Equal(t, expected, res)
}
