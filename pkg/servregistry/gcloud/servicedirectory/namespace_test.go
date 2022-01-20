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

func TestGetNs(t *testing.T) {
	s := getFakeHandler()
	nsName := "ns"

	// Test errors on service directory
	testErr := func(tt *testing.T) {
		assert := a.New(tt)

		regNs, err := s.GetNs("get-error")
		assert.Nil(regNs)
		assert.Error(err)
		assert.NotEqual(sr.ErrTimeOutExpired, err)
		assert.NotEqual(sr.ErrNotFound, err)

		regNs, err = s.GetNs("timeout-error")
		assert.Nil(regNs)
		assert.Equal(sr.ErrTimeOutExpired, err)

		regNs, err = s.GetNs("get-not-found")
		assert.Nil(regNs)
		assert.Equal(sr.ErrNotFound, err)
	}

	// Test success on service directory
	testOk := func(tt *testing.T) {
		assert := a.New(tt)

		regNs, err := s.GetNs(nsName)
		assert.NotNil(regNs)
		assert.NoError(err)
		assert.NotContains(regNs.Name, "/")
	}

	testErr(t)
	testOk(t)
}

func TestCreateNs(t *testing.T) {
	s := getFakeHandler()
	ns := &sr.Namespace{}
	nsName := "ns"

	// Test errors on service directory
	testErr := func(tt *testing.T) {
		assert := a.New(tt)

		ns.Name = "create-error"
		regNs, err := s.CreateNs(ns)
		assert.Nil(regNs)
		assert.Error(err)
		assert.NotEqual(sr.ErrTimeOutExpired, err)
		assert.NotEqual(sr.ErrNotFound, err)

		ns.Name = "timeout-error"
		regNs, err = s.CreateNs(ns)
		assert.Nil(regNs)
		assert.Equal(sr.ErrTimeOutExpired, err)

		ns.Name = "create-exists"
		regNs, err = s.CreateNs(ns)
		assert.Nil(regNs)
		assert.Equal(sr.ErrAlreadyExists, err)
	}

	// Test success on service directory
	testOk := func(tt *testing.T) {
		assert := a.New(tt)

		ns.Name = nsName
		regNs, err := s.CreateNs(ns)
		assert.NotNil(regNs)
		assert.NoError(err)
		assert.NotContains(regNs.Name, "/")
	}

	testErr(t)
	testOk(t)
}

func TestUpdateNs(t *testing.T) {
	s := getFakeHandler()
	ns := &sr.Namespace{}
	nsName := "ns"

	// Test errors on service directory
	testErr := func(tt *testing.T) {
		assert := a.New(tt)

		ns.Name = "update-error"
		regNs, err := s.UpdateNs(ns)
		assert.Nil(regNs)
		assert.Error(err)
		assert.NotEqual(sr.ErrTimeOutExpired, err)
		assert.NotEqual(sr.ErrNotFound, err)

		ns.Name = "timeout-error"
		regNs, err = s.UpdateNs(ns)
		assert.Nil(regNs)
		assert.Equal(sr.ErrTimeOutExpired, err)

		ns.Name = "update-not-found"
		regNs, err = s.UpdateNs(ns)
		assert.Nil(regNs)
		assert.Equal(sr.ErrNotFound, err)
	}

	// Test success on service directory
	testOk := func(tt *testing.T) {
		assert := a.New(tt)

		ns.Name = nsName
		regNs, err := s.UpdateNs(ns)
		assert.NotNil(regNs)
		assert.NoError(err)
		assert.NotContains(regNs.Name, "/")
	}

	testErr(t)
	testOk(t)
}

func TestDeleteNs(t *testing.T) {
	s := getFakeHandler()
	nsName := "ns"

	// Test errors on service directory
	testErr := func(tt *testing.T) {
		assert := a.New(tt)

		err := s.DeleteNs("delete-error")
		assert.Error(err)
		assert.NotEqual(sr.ErrTimeOutExpired, err)
		assert.NotEqual(sr.ErrNotFound, err)

		err = s.DeleteNs("timeout-error")
		assert.Equal(sr.ErrTimeOutExpired, err)

		err = s.DeleteNs("delete-not-found")
		assert.Equal(sr.ErrNotFound, err)
	}

	// Test success on service directory
	testOk := func(tt *testing.T) {
		assert := a.New(tt)

		err := s.DeleteNs(nsName)
		assert.NoError(err)
	}

	testErr(t)
	testOk(t)
}

func TestListNs(t *testing.T) {
	// Not tested as currently not used by the operator
}
