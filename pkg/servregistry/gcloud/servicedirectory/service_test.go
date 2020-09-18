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
	"testing"

	sr "github.com/CloudNativeSDWAN/cnwan-operator/pkg/servregistry"
	a "github.com/stretchr/testify/assert"
)

func TestGetServ(t *testing.T) {
	s := getFakeHandler()
	nsName, servName := "ns", "serv"

	// Test errors
	testErr := func(tt *testing.T) {
		assert := a.New(tt)

		regServ, err := s.GetServ(nsName, "get-error")
		assert.Nil(regServ)
		assert.Error(err)
		assert.NotEqual(sr.ErrTimeOutExpired, err)
		assert.NotEqual(sr.ErrNotFound, err)

		regServ, err = s.GetServ(nsName, "timeout-error")
		assert.Nil(regServ)
		assert.Equal(sr.ErrTimeOutExpired, err)

		regServ, err = s.GetServ(nsName, "get-not-found")
		assert.Nil(regServ)
		assert.Equal(sr.ErrNotFound, err)
	}

	// Test success on service directory
	testOk := func(tt *testing.T) {
		assert := a.New(tt)

		regServ, err := s.GetServ(nsName, servName)
		assert.NotNil(regServ)
		assert.NoError(err)
		assert.NotContains(regServ.Name, "/")
		assert.Equal(regServ.NsName, nsName)
	}

	testErr(t)
	testOk(t)
}

func TestCreateServ(t *testing.T) {
	s := getFakeHandler()
	nsName, servName := "ns", "serv"
	req := &sr.Service{NsName: nsName}

	// Test errors on service directory
	testErr := func(tt *testing.T) {
		assert := a.New(tt)

		regServ, err := s.CreateServ(nil)
		assert.Nil(regServ)
		assert.Equal(sr.ErrServNotProvided, err)

		req.Name = "create-error"
		regServ, err = s.CreateServ(req)
		assert.Nil(regServ)
		assert.Error(err)
		assert.NotEqual(sr.ErrTimeOutExpired, err)
		assert.NotEqual(sr.ErrNotFound, err)

		req.Name = "timeout-error"
		regServ, err = s.CreateServ(req)
		assert.Nil(regServ)
		assert.Equal(sr.ErrTimeOutExpired, err)

		req.Name = "create-exists"
		regServ, err = s.CreateServ(req)
		assert.Nil(regServ)
		assert.Equal(sr.ErrAlreadyExists, err)
	}

	// Test success on service directory
	testOk := func(tt *testing.T) {
		assert := a.New(tt)

		req.Name = servName
		regServ, err := s.CreateServ(req)
		assert.NotNil(regServ)
		assert.NoError(err)
		assert.NotContains(regServ.Name, "/")
		assert.Equal(regServ.NsName, nsName)
	}

	testErr(t)
	testOk(t)
}

func TestUpdateServ(t *testing.T) {
	s := getFakeHandler()
	nsName, servName := "ns", "serv"
	req := &sr.Service{NsName: nsName}

	// Test errors on service directory
	testErr := func(tt *testing.T) {
		assert := a.New(tt)

		regServ, err := s.UpdateServ(nil)
		assert.Nil(regServ)
		assert.Equal(sr.ErrServNotProvided, err)

		req.Name = "update-error"
		regServ, err = s.UpdateServ(req)
		assert.Nil(regServ)
		assert.Error(err)
		assert.NotEqual(sr.ErrTimeOutExpired, err)
		assert.NotEqual(sr.ErrNotFound, err)

		req.Name = "timeout-error"
		regServ, err = s.UpdateServ(req)
		assert.Nil(regServ)
		assert.Equal(sr.ErrTimeOutExpired, err)

		req.Name = "update-not-found"
		regServ, err = s.UpdateServ(req)
		assert.Nil(regServ)
		assert.Equal(sr.ErrNotFound, err)
	}

	// Test success on service directory
	testOk := func(tt *testing.T) {
		assert := a.New(tt)

		req.Name = servName
		regServ, err := s.UpdateServ(req)
		assert.NotNil(regServ)
		assert.NoError(err)
		assert.NotContains(regServ.Name, "/")
		assert.Equal(regServ.NsName, nsName)
	}

	testErr(t)
	testOk(t)
}

func TestDeleteServ(t *testing.T) {
	s := getFakeHandler()
	nsName, servName := "ns", "serv"

	// Test errors on service directory
	testErr := func(tt *testing.T) {
		assert := a.New(tt)

		err := s.DeleteServ(nsName, "delete-error")
		assert.Error(err)
		assert.NotEqual(sr.ErrTimeOutExpired, err)
		assert.NotEqual(sr.ErrNotFound, err)

		err = s.DeleteServ(nsName, "timeout-error")
		assert.Equal(sr.ErrTimeOutExpired, err)

		err = s.DeleteServ(nsName, "delete-not-found")
		assert.Equal(sr.ErrNotFound, err)
	}

	// Test success on service directory
	testOk := func(tt *testing.T) {
		assert := a.New(tt)

		err := s.DeleteServ(nsName, servName)
		assert.NoError(err)
	}

	testErr(t)
	testOk(t)
}
