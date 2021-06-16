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

package servregistry

import (
	"testing"

	a "github.com/stretchr/testify/assert"
)

func TestManageServ(t *testing.T) {
	// prepare
	nsName, servName := "ns", "serv"
	var f *fakeServReg
	b, _ := NewBroker(f, MetadataPair{})

	resetFake := func() {
		f = newFakeStruct()
		b.Reg = f
	}

	resetFake()

	// Test validation
	testValidation := func(tt *testing.T) {
		defer resetFake()
		assert := a.New(tt)

		invalidServData := &Service{}
		regServ := &Service{}
		var err error

		// no service registry provided
		b.Reg = nil
		regServ, err = b.ManageServ(nil)
		assert.Nil(regServ)
		assert.Equal(ErrServRegNotProvided, err)

		// service is nil
		b.Reg = f
		regServ, err = b.ManageServ(nil)
		assert.Nil(regServ)
		assert.Equal(ErrServNotProvided, err)

		// service has no name
		regServ, err = b.ManageServ(invalidServData)
		assert.Nil(regServ)
		assert.Equal(ErrServNameNotProvided, err)

		// service has no namespace name
		invalidServData.Name = servName
		regServ, err = b.ManageServ(invalidServData)
		assert.Nil(regServ)
		assert.Equal(ErrNsNameNotProvided, err)

		assert.Empty(f.createdServ)
		assert.Empty(f.updatedServ)
	}

	// Test returns nil when an unknown error is thrown by the service registry
	testUnErr := func(tt *testing.T) {
		defer resetFake()
		assert := a.New(tt)

		regServ, err := b.ManageServ(&Service{Name: "get-error", NsName: "ns"})
		assert.Nil(regServ)
		assert.Error(err)
		assert.NotEqual(ErrServRegNotProvided, err)
		assert.NotEqual(ErrServNotProvided, err)
		assert.NotEqual(ErrServNameNotProvided, err)
		assert.NotEqual(ErrNsNameNotProvided, err)

		assert.Empty(f.createdServ)
		assert.Empty(f.updatedServ)
	}

	// Test service not owned by the operator are not modified
	testNotOwned := func(tt *testing.T) {
		defer resetFake()
		assert := a.New(tt)

		one := &Service{Name: "one", NsName: nsName, Metadata: map[string]string{b.opMetaPair.Key: "someone-else", "key": "val"}}
		two := &Service{Name: "two", NsName: nsName, Metadata: map[string]string{"key": "val"}}
		f.servList = map[string]*Service{one.Name: one, two.Name: two}

		oneChange := &Service{Name: "one", NsName: nsName, Metadata: map[string]string{b.opMetaPair.Key: b.opMetaPair.Value, "key": "val-1"}}
		twoChange := &Service{Name: "two", NsName: nsName, Metadata: map[string]string{"key": "val-1"}}

		regServ, err := b.ManageServ(oneChange)
		assert.Equal(one, regServ)
		assert.NoError(err)

		regServ, err = b.ManageServ(twoChange)
		assert.Equal(two, regServ)
		assert.NoError(err)

		assert.Empty(f.createdServ)
		assert.Empty(f.updatedServ)
	}

	// Test services owned by the operator are modified
	testOwned := func(tt *testing.T) {
		defer resetFake()
		assert := a.New(tt)

		// should return nil because an error in updating
		shouldErr := &Service{Name: "update-error", NsName: nsName, Metadata: map[string]string{b.opMetaPair.Key: b.opMetaPair.Value, "key": "val"}}
		changeErr := &Service{Name: "update-error", NsName: nsName, Metadata: map[string]string{b.opMetaPair.Key: b.opMetaPair.Value, "key": "val-1"}}
		f.servList[shouldErr.Name] = shouldErr

		regServ, err := b.ManageServ(changeErr)
		assert.Nil(regServ)
		assert.Error(err)
		assert.Empty(f.updatedServ)

		// no error so it should return the new value
		shouldOk := &Service{Name: "update", NsName: nsName, Metadata: map[string]string{b.opMetaPair.Key: b.opMetaPair.Value, "key": "val"}}
		okChange := &Service{Name: "update", NsName: nsName, Metadata: map[string]string{b.opMetaPair.Key: b.opMetaPair.Value, "key": "val-1"}}
		f.servList[shouldOk.Name] = shouldOk

		regServ, err = b.ManageServ(okChange)
		assert.Equal(okChange, regServ)
		assert.NoError(err)

		assert.Empty(f.createdServ)
		assert.Len(f.updatedServ, 1)
	}

	// Test services are created if they do not exist
	testCreate := func(tt *testing.T) {
		defer resetFake()
		assert := a.New(tt)

		// should return nil because an error in creating
		// (this also happens if someone else creates this)
		shouldErr := &Service{Name: "create-error", NsName: nsName, Metadata: map[string]string{b.opMetaPair.Key: b.opMetaPair.Value, "key": "val"}}
		regServ, err := b.ManageServ(shouldErr)
		assert.Nil(regServ)
		assert.Error(err)
		assert.Empty(f.createdServ)

		// no error so it should return the created
		shouldOk := &Service{Name: "create", NsName: nsName, Metadata: map[string]string{"key": "val"}}
		regServ, err = b.ManageServ(shouldOk)
		assert.Equal(shouldOk.Name, regServ.Name)
		assert.Equal(shouldOk.NsName, regServ.NsName)
		assert.Equal(map[string]string{b.opMetaPair.Key: b.opMetaPair.Value, "key": "val"}, regServ.Metadata)
		assert.NoError(err)

		assert.Empty(f.updatedServ)
		assert.Len(f.createdServ, 1)
	}

	testValidation(t)
	testUnErr(t)
	testNotOwned(t)
	testOwned(t)
	testCreate(t)
}

func TestRemoveServ(t *testing.T) {
	// prepare
	nsName, servName := "ns", "serv"
	var f *fakeServReg
	b, _ := NewBroker(f, MetadataPair{})

	resetFake := func() {
		f = newFakeStruct()
		b.Reg = f
	}

	resetFake()

	// Test validation
	testValidation := func(tt *testing.T) {
		defer resetFake()
		assert := a.New(tt)

		// no service registry provided
		b.Reg = nil
		err := b.RemoveServ(nsName, servName, false)
		assert.Equal(ErrServRegNotProvided, err)

		// service name not provided
		b.Reg = f
		err = b.RemoveServ(nsName, "", false)
		assert.Equal(ErrServNameNotProvided, err)

		// namespace name not provided
		err = b.RemoveServ("", servName, false)
		assert.Equal(ErrNsNameNotProvided, err)
		assert.Empty(f.deletedServ)
	}

	// Test returns nil when an unknown error is thrown by the service registry
	testUnErr := func(tt *testing.T) {
		defer resetFake()
		assert := a.New(tt)

		err := b.RemoveServ(nsName, "get-error", false)
		assert.Error(err)
		assert.NotEqual(ErrServRegNotProvided, err)
		assert.NotEqual(ErrServNameNotProvided, err)
		assert.NotEqual(ErrNsNameNotProvided, err)

		assert.Empty(f.deletedServ)
	}

	// Test services not owned by the operator are not deleted
	testNotOwned := func(tt *testing.T) {
		defer resetFake()
		assert := a.New(tt)

		one := &Service{Name: "one", NsName: nsName, Metadata: map[string]string{b.opMetaPair.Key: "someone-else", "key": "val"}}
		two := &Service{Name: "two", NsName: nsName, Metadata: map[string]string{"key": "val"}}
		f.servList[one.Name] = one
		f.servList[two.Name] = two

		err := b.RemoveServ(one.NsName, one.Name, false)
		assert.Equal(ErrServNotOwnedByOp, err)
		err = b.RemoveServ(two.NsName, two.Name, false)
		assert.Equal(ErrServNotOwnedByOp, err)
		assert.Empty(f.deletedServ)
	}

	// Test empty owned services are deleted
	testEmptyOwned := func(tt *testing.T) {
		defer resetFake()
		assert := a.New(tt)

		// unknown error
		err := b.RemoveServ(nsName, "delete-error", false)
		assert.NotEqual(ErrServRegNotProvided, err)
		assert.NotEqual(ErrNsNameNotProvided, err)

		// when it doesn't exist, we just log but return no error
		// because it doesn't change anything for us
		err = b.RemoveServ(nsName, "doesnt-exist", false)
		assert.NoError(err)

		// successful
		toDel := &Service{Name: "to-del", NsName: nsName, Metadata: map[string]string{b.opMetaPair.Key: b.opMetaPair.Value, "key": "val"}}
		f.servList[toDel.Name] = toDel
		err = b.RemoveServ(nsName, "to-del", false)
		assert.NoError(err)
		assert.Len(f.deletedServ, 1)
	}

	// Test not empty owned namespaces are deleted/not deleted
	testNotEmptyOwned := func(tt *testing.T) {
		defer resetFake()
		assert := a.New(tt)

		oneOwned := &Endpoint{
			Name:     "one",
			ServName: servName,
			NsName:   nsName,
			Metadata: map[string]string{b.opMetaPair.Key: b.opMetaPair.Value, "key": "val"},
		}
		twoOwned := &Endpoint{
			Name:     "two",
			ServName: servName,
			NsName:   nsName,
			Metadata: map[string]string{b.opMetaPair.Key: b.opMetaPair.Value, "key": "val"},
		}
		threeNotOwned := &Endpoint{
			Name:     "three",
			ServName: servName,
			NsName:   nsName,
			Metadata: map[string]string{b.opMetaPair.Key: "someone-else", "key": "val"},
		}
		servDel := &Service{Name: servName, NsName: nsName, Metadata: map[string]string{b.opMetaPair.Key: b.opMetaPair.Value, "key": "val"}}
		f.servList[servDel.Name] = servDel

		// error in listing
		f.endpList["list-error"] = &Endpoint{}
		err := b.RemoveServ(servDel.NsName, servDel.Name, false)
		assert.Error(err)
		assert.Empty(f.deletedServ)
		delete(f.endpList, "list-error")

		// there is an endpoint not owned by us
		f.endpList["one"] = oneOwned
		f.endpList["two"] = twoOwned
		f.endpList["three"] = threeNotOwned

		err = b.RemoveServ(servDel.NsName, servDel.Name, false)
		assert.Empty(f.deletedEndp)
		assert.Empty(f.deletedServ)
		assert.Equal(ErrServNotEmpty, err)

		err = b.RemoveServ(servDel.NsName, servDel.Name, true)
		assert.Len(f.deletedEndp, 2)
		assert.Empty(f.deletedServ)
		assert.Equal(ErrServNotOwnedEndps, err)
		assert.Contains(f.deletedEndp, oneOwned.Name)
		assert.Contains(f.deletedEndp, twoOwned.Name)

		// all endpoints owned by the operator, so should be successful
		delete(f.endpList, threeNotOwned.Name)
		f.endpList["one"] = oneOwned
		f.endpList["two"] = twoOwned
		f.deletedEndp = []string{}
		f.deletedServ = []string{}
		err = b.RemoveServ(servDel.NsName, servDel.Name, true)
		assert.NoError(err)

		// error occurs in deleting service
		shouldErr := &Service{Name: "delete-error", NsName: nsName, Metadata: map[string]string{b.opMetaPair.Key: b.opMetaPair.Value, "key": "val"}}
		f.servList[shouldErr.Name] = shouldErr
		f.deletedEndp = []string{}
		f.deletedServ = []string{}
		err = b.RemoveServ(shouldErr.NsName, shouldErr.Name, true)
		assert.Error(err)
	}

	testValidation(t)
	testUnErr(t)
	testNotOwned(t)
	testEmptyOwned(t)
	testNotEmptyOwned(t)
}
