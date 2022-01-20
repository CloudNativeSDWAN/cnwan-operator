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

import (
	"testing"

	a "github.com/stretchr/testify/assert"
)

func TestManageNs(t *testing.T) {
	// prepare
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

		invalidNsData := &Namespace{}
		regNs := &Namespace{}
		var err error

		// no service registry provided
		b.Reg = nil
		regNs, err = b.ManageNs(nil)
		assert.Nil(regNs)
		assert.Equal(ErrServRegNotProvided, err)

		// namespace is nil
		b.Reg = f
		regNs, err = b.ManageNs(nil)
		assert.Nil(regNs)
		assert.Equal(ErrNsNotProvided, err)

		// namespace has no name
		regNs, err = b.ManageNs(invalidNsData)
		assert.Nil(regNs)
		assert.Equal(ErrNsNameNotProvided, err)

		assert.Empty(f.createdNs)
		assert.Empty(f.updatedNs)
	}

	// Test returns nil when an unknown error is thrown by the service registry
	testUnErr := func(tt *testing.T) {
		defer resetFake()
		assert := a.New(tt)

		regNs, err := b.ManageNs(&Namespace{Name: "get-error"})
		assert.Nil(regNs)
		assert.Error(err)
		assert.NotEqual(ErrNsNameNotProvided, err)
		assert.NotEqual(ErrNsNotProvided, err)
		assert.NotEqual(ErrNsNameNotProvided, err)

		assert.Empty(f.createdNs)
		assert.Empty(f.updatedNs)
	}

	// Test namespaces not owned by the operator are not modified
	testNotOwned := func(tt *testing.T) {
		defer resetFake()
		assert := a.New(tt)

		one := &Namespace{Name: "one", Metadata: map[string]string{b.opMetaPair.Key: "someone-else", "key": "val"}}
		two := &Namespace{Name: "two", Metadata: map[string]string{"key": "val"}}

		oneChange := &Namespace{Name: "one", Metadata: map[string]string{b.opMetaPair.Key: b.opMetaPair.Value, "key": "val-1"}}
		twoChange := &Namespace{Name: "two", Metadata: map[string]string{b.opMetaPair.Key: b.opMetaPair.Value, "key": "val-1"}}
		f.nsList[one.Name] = one
		f.nsList[two.Name] = two

		regNs, err := b.ManageNs(oneChange)
		assert.Equal(one, regNs)
		assert.NoError(err)

		regNs, err = b.ManageNs(twoChange)
		assert.Equal(two, regNs)
		assert.NoError(err)

		assert.Empty(f.createdNs)
		assert.Empty(f.updatedNs)
	}

	// Test namespaces owned by the operator are modified
	testOwned := func(tt *testing.T) {
		defer resetFake()
		assert := a.New(tt)

		// should return nil because an error in updating
		shouldErr := &Namespace{Name: "update-error", Metadata: map[string]string{b.opMetaPair.Key: b.opMetaPair.Value, "key": "val"}}
		changeErr := &Namespace{Name: "update-error", Metadata: map[string]string{b.opMetaPair.Key: b.opMetaPair.Value, "key": "val-1"}}
		f.nsList[shouldErr.Name] = shouldErr

		regNs, err := b.ManageNs(changeErr)
		assert.Nil(regNs)
		assert.Error(err)

		// no error so it should return the new value
		shouldOk := &Namespace{Name: "update", Metadata: map[string]string{b.opMetaPair.Key: b.opMetaPair.Value, "key": "val"}}
		changeOk := &Namespace{Name: "update", Metadata: map[string]string{b.opMetaPair.Key: b.opMetaPair.Value, "key": "val-1"}}
		f.nsList[shouldOk.Name] = shouldOk

		regNs, err = b.ManageNs(changeOk)
		assert.Equal(changeOk, regNs)
		assert.NoError(err)

		assert.Empty(f.createdNs)
	}

	// Test namespaces are created if they do not exist
	testCreate := func(tt *testing.T) {
		defer resetFake()
		assert := a.New(tt)

		// should return nil because an error in creating
		// (this also happens if someone else creates this)
		create := &Namespace{Name: "create-error", Metadata: map[string]string{b.opMetaPair.Key: b.opMetaPair.Value, "key": "val"}}
		regNs, err := b.ManageNs(create)
		assert.Nil(regNs)
		assert.Error(err)

		// no error so it should return the created
		create = &Namespace{Name: "create", Metadata: map[string]string{"key": "val"}}
		regNs, err = b.ManageNs(create)
		assert.Equal(create.Name, regNs.Name)
		assert.Equal(map[string]string{b.opMetaPair.Key: b.opMetaPair.Value, "key": "val"}, regNs.Metadata)
		assert.NoError(err)

		assert.Empty(f.updatedNs)
	}

	testValidation(t)
	testUnErr(t)
	testNotOwned(t)
	testOwned(t)
	testCreate(t)
}

func TestRemoveNs(t *testing.T) {
	// prepare
	nsName := "ns"
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
		err := b.RemoveNs(nsName, false)
		assert.Equal(ErrServRegNotProvided, err)

		// namespace name not provided
		b.Reg = f
		err = b.RemoveNs("", false)
		assert.Equal(ErrNsNameNotProvided, err)
		assert.Empty(f.deletedNs)
	}

	// Test returns nil when an unknown error is thrown by the service registry
	testUnErr := func(tt *testing.T) {
		defer resetFake()
		assert := a.New(tt)

		err := b.RemoveNs("get-error", false)
		assert.Error(err)
		assert.NotEqual(ErrNsNameNotProvided, err)
		assert.NotEqual(ErrNsNotProvided, err)
		assert.NotEqual(ErrNsNameNotProvided, err)

		assert.Empty(f.deletedNs)
	}

	// Test namespaces not owned by the operator are not deleted
	testNotOwned := func(tt *testing.T) {
		defer resetFake()
		assert := a.New(tt)

		one := &Namespace{Name: "not-owned", Metadata: map[string]string{b.opMetaPair.Key: "someone-else", "key": "val"}}
		two := &Namespace{Name: "not-owned", Metadata: map[string]string{"key": "val"}}
		f.nsList[one.Name] = one
		f.nsList[two.Name] = two

		err := b.RemoveNs(one.Name, false)
		assert.Equal(ErrNsNotOwnedByOp, err)

		err = b.RemoveNs(two.Name, false)
		assert.Equal(ErrNsNotOwnedByOp, err)
		assert.Empty(f.deletedNs)
	}

	// Test empty owned namespaces are deleted
	testEmptyOwned := func(tt *testing.T) {
		defer resetFake()
		assert := a.New(tt)

		// when it doesn't exist, we just log but return no error
		// because it doesn't change anything for us
		err := b.RemoveNs("doesnt-exist", false)
		assert.NoError(err)

		// unknown error
		toDel := &Namespace{Name: "delete-error", Metadata: map[string]string{b.opMetaPair.Key: b.opMetaPair.Value, "key": "val"}}
		f.nsList[toDel.Name] = toDel
		err = b.RemoveNs("delete-error", false)
		assert.NotEqual(ErrServRegNotProvided, err)
		assert.NotEqual(ErrNsNameNotProvided, err)

		// successful
		present := &Namespace{Name: "owned", Metadata: map[string]string{b.opMetaPair.Key: b.opMetaPair.Value, "key": "val"}}
		f.nsList[present.Name] = present
		err = b.RemoveNs("owned", false)
		assert.NoError(err)
		assert.Len(f.deletedNs, 1)
	}

	// Test not empty owned namespaces are deleted/not deleted
	testNotEmptyOwned := func(tt *testing.T) {
		defer resetFake()
		assert := a.New(tt)

		oneOwned := &Service{
			Name:     "one",
			Metadata: map[string]string{b.opMetaPair.Key: b.opMetaPair.Value, "key": "val"},
			NsName:   nsName,
		}
		twoOwned := &Service{
			Name:     "two",
			Metadata: map[string]string{b.opMetaPair.Key: b.opMetaPair.Value, "key": "val"},
			NsName:   nsName,
		}
		threeNotOwned := &Service{
			Name:     "three",
			Metadata: map[string]string{b.opMetaPair.Key: "someone-else", "key": "val"},
			NsName:   nsName,
		}
		nsDel := &Namespace{Name: nsName, Metadata: map[string]string{b.opMetaPair.Key: b.opMetaPair.Value, "key": "val"}}
		f.nsList[nsDel.Name] = nsDel

		// error in listing
		f.servList["list-error"] = &Service{}
		err := b.RemoveNs(nsDel.Name, false)
		assert.Error(err)
		delete(f.servList, "list-error")

		// there is a service not owned by us
		f.servList["one"] = oneOwned
		f.servList["two"] = twoOwned
		f.servList["three"] = threeNotOwned
		err = b.RemoveNs(nsDel.Name, false)
		assert.Empty(f.deletedServ)
		assert.Empty(f.deletedNs)
		assert.Equal(ErrNsNotEmpty, err)

		err = b.RemoveNs(nsDel.Name, true)
		assert.Len(f.deletedServ, 2)
		assert.Empty(f.deletedNs)
		assert.Equal(ErrNsNotOwnedServs, err)
		assert.Contains(f.deletedServ, oneOwned.Name)
		assert.Contains(f.deletedServ, twoOwned.Name)

		// error occurs in deleting namespace
		shouldErr := &Namespace{Name: "delete-error", Metadata: map[string]string{b.opMetaPair.Key: b.opMetaPair.Value, "key": "val"}}
		f.nsList[shouldErr.Name] = shouldErr
		f.deletedServ = []string{}
		f.deletedNs = []string{}
		err = b.RemoveNs(shouldErr.Name, true)
		assert.Error(err)
	}

	testValidation(t)
	testUnErr(t)
	testNotOwned(t)
	testEmptyOwned(t)
	testNotEmptyOwned(t)
}
