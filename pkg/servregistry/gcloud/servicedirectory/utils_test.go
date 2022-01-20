// Copyright © 2020, 2021 Cisco
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

package servicedirectory

import (
	"testing"

	sr "github.com/CloudNativeSDWAN/cnwan-operator/pkg/servregistry"
	a "github.com/stretchr/testify/assert"
)

func TestGetResourcePath(t *testing.T) {
	s := &Handler{ProjectID: "project", DefaultRegion: "us"}
	assert := a.New(t)

	// Test no project/region provided
	res := s.getResourcePath(servDirPath{})
	assert.Equal("projects/project/locations/us", res)

	// Test project provided
	arg := servDirPath{
		project: "other-project",
	}
	res = s.getResourcePath(arg)
	assert.Equal("projects/other-project/locations/us", res)

	// Test loaction provided
	arg.region = "asia"
	res = s.getResourcePath(arg)
	assert.Equal("projects/other-project/locations/asia", res)

	// Test service cannot exist with namespace provided
	arg.service = "serv"
	res = s.getResourcePath(arg)
	assert.Equal("projects/other-project/locations/asia", res)
	arg.service = ""

	// Test ns provided
	arg.namespace = "ns"
	res = s.getResourcePath(arg)
	assert.Equal("projects/other-project/locations/asia/namespaces/ns", res)

	// Test endpoint cannot live without service
	arg.endpoint = "endp"
	res = s.getResourcePath(arg)
	assert.Equal("projects/other-project/locations/asia/namespaces/ns", res)
	arg.endpoint = ""

	// Test service provided
	arg.service = "serv"
	res = s.getResourcePath(arg)
	assert.Equal("projects/other-project/locations/asia/namespaces/ns/services/serv", res)

	// Test endpoint provided
	arg.endpoint = "endp"
	res = s.getResourcePath(arg)
	assert.Equal("projects/other-project/locations/asia/namespaces/ns/services/serv/endpoints/endp", res)
}

func TestCheckNames(t *testing.T) {
	s := getFakeHandler()
	assert := a.New(t)

	err := s.checkNames(nil, nil, nil)
	assert.NoError(err)

	nsName := ""
	err = s.checkNames(&nsName, nil, nil)
	assert.Equal(sr.ErrNsNameNotProvided, err)

	nsName = "ns"
	servName := ""
	err = s.checkNames(&nsName, &servName, nil)
	assert.Equal(sr.ErrServNameNotProvided, err)

	servName = "serv"
	endpName := ""
	err = s.checkNames(&nsName, &servName, &endpName)
	assert.Equal(sr.ErrEndpNameNotProvided, err)

	endpName = "endp"
	err = s.checkNames(&nsName, &servName, &endpName)
	assert.NoError(err)
}
