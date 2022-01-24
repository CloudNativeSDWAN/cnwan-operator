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

package servicedirectory

import (
	"testing"

	sr "github.com/CloudNativeSDWAN/cnwan-operator/pkg/servregistry"
	a "github.com/stretchr/testify/assert"
)

func TestGetEndp(t *testing.T) {
	s := getFakeHandler()
	nsName, servName, endpName := "ns", "serv", "endp"

	// Test errors
	testErr := func(tt *testing.T) {
		assert := a.New(tt)

		regEndp, err := s.GetEndp(nsName, servName, "get-error")
		assert.Nil(regEndp)
		assert.Error(err)
		assert.NotEqual(sr.ErrTimeOutExpired, err)
		assert.NotEqual(sr.ErrNotFound, err)

		regEndp, err = s.GetEndp(nsName, servName, "timeout-error")
		assert.Nil(regEndp)
		assert.Equal(sr.ErrTimeOutExpired, err)

		regEndp, err = s.GetEndp(nsName, servName, "get-not-found")
		assert.Nil(regEndp)
		assert.Equal(sr.ErrNotFound, err)
	}

	// Test success on service directory
	testOk := func(tt *testing.T) {
		assert := a.New(tt)

		regEndp, err := s.GetEndp(nsName, servName, endpName)
		assert.NotNil(regEndp)
		assert.NoError(err)
		assert.NotContains(regEndp.Name, "/")
		assert.Equal(regEndp.NsName, nsName)
	}

	testErr(t)
	testOk(t)
}

func TestCreateEndp(t *testing.T) {
	s := getFakeHandler()
	nsName, servName, endpName := "ns", "serv", "endp"
	req := &sr.Endpoint{NsName: nsName, ServName: servName}

	// Test errors on service directory
	testErr := func(tt *testing.T) {
		assert := a.New(tt)

		regEndp, err := s.CreateEndp(nil)
		assert.Nil(regEndp)
		assert.Equal(sr.ErrEndpNotProvided, err)

		req.Name = "create-error"
		regEndp, err = s.CreateEndp(req)
		assert.Nil(regEndp)
		assert.Error(err)
		assert.NotEqual(sr.ErrTimeOutExpired, err)
		assert.NotEqual(sr.ErrNotFound, err)

		req.Name = "timeout-error"
		regEndp, err = s.CreateEndp(req)
		assert.Nil(regEndp)
		assert.Equal(sr.ErrTimeOutExpired, err)

		req.Name = "create-exists"
		regEndp, err = s.CreateEndp(req)
		assert.Nil(regEndp)
		assert.Equal(sr.ErrAlreadyExists, err)
	}

	// Test success on service directory
	testOk := func(tt *testing.T) {
		assert := a.New(tt)

		req.Name = endpName
		regEndp, err := s.CreateEndp(req)
		assert.NotNil(regEndp)
		assert.NoError(err)
		assert.NotContains(regEndp.Name, "/")
		assert.Equal(regEndp.NsName, nsName)
		assert.Equal(regEndp.ServName, servName)
	}

	testErr(t)
	testOk(t)
}

func TestUpdateEndp(t *testing.T) {
	s := getFakeHandler()
	nsName, servName, endpName := "ns", "serv", "endp"
	req := &sr.Endpoint{NsName: nsName, ServName: servName}

	// Test errors on service directory
	testErr := func(tt *testing.T) {
		assert := a.New(tt)

		regEndp, err := s.UpdateEndp(nil)
		assert.Nil(regEndp)
		assert.Equal(sr.ErrEndpNotProvided, err)

		req.Name = "update-error"
		regEndp, err = s.UpdateEndp(req)
		assert.Nil(regEndp)
		assert.Error(err)
		assert.NotEqual(sr.ErrTimeOutExpired, err)
		assert.NotEqual(sr.ErrNotFound, err)

		req.Name = "timeout-error"
		regEndp, err = s.UpdateEndp(req)
		assert.Nil(regEndp)
		assert.Equal(sr.ErrTimeOutExpired, err)

		req.Name = "update-not-found"
		regEndp, err = s.UpdateEndp(req)
		assert.Nil(regEndp)
		assert.Equal(sr.ErrNotFound, err)
	}

	// Test success on service directory
	testOk := func(tt *testing.T) {
		assert := a.New(tt)

		req.Name = endpName
		regEndp, err := s.UpdateEndp(req)
		assert.NotNil(regEndp)
		assert.NoError(err)
		assert.NotContains(regEndp.Name, "/")
		assert.Equal(regEndp.NsName, nsName)
	}

	testErr(t)
	testOk(t)
}

func TestDeleteEndp(t *testing.T) {
	s := getFakeHandler()
	nsName, servName, endpName := "ns", "serv", "endp"

	// Test errors on service directory
	testErr := func(tt *testing.T) {
		assert := a.New(tt)

		err := s.DeleteEndp(nsName, servName, "delete-error")
		assert.Error(err)
		assert.NotEqual(sr.ErrTimeOutExpired, err)
		assert.NotEqual(sr.ErrNotFound, err)

		err = s.DeleteEndp(nsName, servName, "timeout-error")
		assert.Equal(sr.ErrTimeOutExpired, err)

		err = s.DeleteEndp(nsName, servName, "delete-not-found")
		assert.Equal(sr.ErrNotFound, err)
	}

	// Test success on service directory
	testOk := func(tt *testing.T) {
		assert := a.New(tt)

		err := s.DeleteEndp(nsName, servName, endpName)
		assert.NoError(err)
	}

	testErr(t)
	testOk(t)
}
