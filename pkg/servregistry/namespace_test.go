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

func TestManageNs(t *testing.T) {
	// prepare
	var f *fakeServReg
	b, _ := NewBroker(f, "", "")

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

		one := &Namespace{Name: "one", Metadata: map[string]string{b.opKey: "someone-else", "key": "val"}}
		two := &Namespace{Name: "two", Metadata: map[string]string{"key": "val"}}

		oneChange := &Namespace{Name: "one", Metadata: map[string]string{b.opKey: b.opVal, "key": "val-1"}}
		twoChange := &Namespace{Name: "two", Metadata: map[string]string{b.opKey: b.opVal, "key": "val-1"}}
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
		shouldErr := &Namespace{Name: "update-error", Metadata: map[string]string{b.opKey: b.opVal, "key": "val"}}
		changeErr := &Namespace{Name: "update-error", Metadata: map[string]string{b.opKey: b.opVal, "key": "val-1"}}
		f.nsList[shouldErr.Name] = shouldErr

		regNs, err := b.ManageNs(changeErr)
		assert.Nil(regNs)
		assert.Error(err)

		// no error so it should return the new value
		shouldOk := &Namespace{Name: "update", Metadata: map[string]string{b.opKey: b.opVal, "key": "val"}}
		changeOk := &Namespace{Name: "update", Metadata: map[string]string{b.opKey: b.opVal, "key": "val-1"}}
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
		create := &Namespace{Name: "create-error", Metadata: map[string]string{b.opKey: b.opVal, "key": "val"}}
		regNs, err := b.ManageNs(create)
		assert.Nil(regNs)
		assert.Error(err)

		// no error so it should return the created
		create = &Namespace{Name: "create", Metadata: map[string]string{"key": "val"}}
		regNs, err = b.ManageNs(create)
		assert.Equal(create.Name, regNs.Name)
		assert.Equal(map[string]string{b.opKey: b.opVal, "key": "val"}, regNs.Metadata)
		assert.NoError(err)

		assert.Empty(f.updatedNs)
	}

	testValidation(t)
	testUnErr(t)
	testNotOwned(t)
	testOwned(t)
	testCreate(t)
}
