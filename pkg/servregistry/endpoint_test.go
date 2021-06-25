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

func TestManageServEndps(t *testing.T) {
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
		regEndps, err := b.ManageServEndps(nsName, servName, nil)
		assert.Empty(regEndps)
		assert.Equal(ErrServRegNotProvided, err)

		// ns name is not specified
		b.Reg = f
		regEndps, err = b.ManageServEndps("", servName, nil)
		assert.Empty(regEndps)
		assert.Equal(ErrNsNameNotProvided, err)

		// serv name is not specified
		regEndps, err = b.ManageServEndps(nsName, "", nil)
		assert.Empty(regEndps)
		assert.Equal(ErrServNameNotProvided, err)

		// error while getting list of endpoints
		f.endpList = map[string]*Endpoint{"list-error": {}}
		regEndps, err = b.ManageServEndps(nsName, servName, []*Endpoint{})
		assert.Empty(regEndps)
		assert.Error(err)

		assert.Empty(f.createdEndp)
		assert.Empty(f.updatedEndp)
		assert.Empty(f.deletedEndp)
	}

	// Test endpoints not owned by the operator are not modified
	testNotOwned := func(tt *testing.T) {
		defer resetFake()
		assert := a.New(tt)

		oneNotOwned := &Endpoint{Name: "one", NsName: nsName, ServName: servName, Metadata: map[string]string{b.opMetaPair.Key: "someone-else", "key": "val"}}
		twoNotOwned := &Endpoint{Name: "two", NsName: nsName, ServName: servName, Metadata: map[string]string{"key": "val"}}
		oneChange := &Endpoint{Name: "one", NsName: nsName, ServName: servName, Metadata: map[string]string{b.opMetaPair.Key: b.opMetaPair.Value, "key": "val"}}
		twoChange := &Endpoint{Name: "two", NsName: nsName, ServName: servName, Metadata: map[string]string{b.opMetaPair.Key: b.opMetaPair.Value, "key": "val"}}
		f.endpList[oneNotOwned.Name] = oneNotOwned
		f.endpList[twoNotOwned.Name] = twoNotOwned

		endpErrs, err := b.ManageServEndps(nsName, servName, []*Endpoint{oneChange, twoChange})
		assert.NoError(err)
		assert.Len(endpErrs, 2)
		assert.Equal(ErrEndpNotOwnedByOp, endpErrs[oneNotOwned.Name])
		assert.Equal(ErrEndpNotOwnedByOp, endpErrs[twoNotOwned.Name])

		assert.Empty(f.createdEndp)
		assert.Empty(f.updatedEndp)
		assert.Empty(f.deletedEndp)
	}

	// Test endpoints owned by the operator are deleted
	testOwnedDelete := func(tt *testing.T) {
		defer resetFake()
		assert := a.New(tt)

		// an error occurs while deleting one of them
		oneOwned := &Endpoint{Name: "one-owned", NsName: nsName, ServName: servName, Metadata: map[string]string{b.opMetaPair.Key: b.opMetaPair.Value, "key": "val"}}
		errored := &Endpoint{Name: "delete-error", NsName: nsName, ServName: servName, Metadata: map[string]string{b.opMetaPair.Key: b.opMetaPair.Value, "key": "val"}}
		f.endpList[oneOwned.Name] = oneOwned
		f.endpList[errored.Name] = errored

		endpErrs, err := b.ManageServEndps(nsName, servName, []*Endpoint{})
		assert.NoError(err)
		assert.Len(endpErrs, 1)
		// assert that the error was indeed thrown by DeleteEndp and not by someone else
		assert.EqualError(f.DeleteEndp(nsName, servName, "delete-error"), endpErrs[errored.Name].Error())

		assert.Len(f.deletedEndp, 1)
		assert.Equal(f.deletedEndp[0], oneOwned.Name)
		assert.Empty(f.createdEndp)
		assert.Empty(f.updatedEndp)
	}

	// Test owned endpoints are updated
	testOwnedUpd := func(tt *testing.T) {
		defer resetFake()
		assert := a.New(tt)

		// an error occurs while updating one of them
		no := &Endpoint{Name: "no", NsName: nsName, ServName: servName, Address: "1.1.1.1", Port: 1010,
			Metadata: map[string]string{b.opMetaPair.Key: b.opMetaPair.Value, "key": "val"},
		}
		metad := &Endpoint{Name: "metad", NsName: nsName, ServName: servName, Address: "10.10.10.10", Port: 8080,
			Metadata: map[string]string{b.opMetaPair.Key: b.opMetaPair.Value, "key": "val"},
		}
		adr := &Endpoint{Name: "adr", NsName: nsName, ServName: servName, Address: "11.11.11.11", Port: 8181,
			Metadata: map[string]string{b.opMetaPair.Key: b.opMetaPair.Value, "key": "val"},
		}
		por := &Endpoint{Name: "por", NsName: nsName, ServName: servName, Address: "12.12.12.12", Port: 8282,
			Metadata: map[string]string{b.opMetaPair.Key: b.opMetaPair.Value, "key": "val"},
		}
		metadChange := &Endpoint{Name: "metad", NsName: nsName, ServName: servName, Address: "10.10.10.10", Port: 8080,
			Metadata: map[string]string{b.opMetaPair.Key: b.opMetaPair.Value, "key": "val-1"},
		}
		adrChange := &Endpoint{Name: "adr", NsName: nsName, ServName: servName, Address: "12.12.12.12", Port: 8181,
			Metadata: map[string]string{b.opMetaPair.Key: b.opMetaPair.Value, "key": "val"},
		}
		porChange := &Endpoint{Name: "por", NsName: nsName, ServName: servName, Address: "12.12.12.12", Port: 2828,
			Metadata: map[string]string{b.opMetaPair.Key: b.opMetaPair.Value, "key": "val"},
		}
		errored := &Endpoint{Name: "update-error", NsName: nsName, ServName: servName, Metadata: map[string]string{b.opMetaPair.Key: b.opMetaPair.Value, "key": "val"}}
		erroredChange := &Endpoint{Name: "update-error", NsName: nsName, ServName: servName, Metadata: map[string]string{b.opMetaPair.Key: b.opMetaPair.Value, "key": "val-1"}}

		f.endpList[no.Name] = no
		f.endpList[metad.Name] = metad
		f.endpList[adr.Name] = adr
		f.endpList[por.Name] = por
		f.endpList[errored.Name] = errored

		endpErrs, err := b.ManageServEndps(nsName, servName, []*Endpoint{metadChange, adrChange, porChange, erroredChange, no})
		assert.NoError(err)
		assert.Len(endpErrs, 1)
		_, expErr := f.UpdateEndp(&Endpoint{Name: "update-error"})
		assert.EqualError(expErr, endpErrs[errored.Name].Error())

		// check that the function was actually called for each change
		assert.Contains(f.updatedEndp, metad.Name)
		assert.Contains(f.updatedEndp, adr.Name)
		assert.Contains(f.updatedEndp, por.Name)
		assert.NotContains(f.updatedEndp, no.Name)

		// assert that nothing was created or deleted
		assert.Empty(f.createdEndp)
		assert.Empty(f.deletedEndp)
	}

	// Test endpoints are created correctly
	testOwnedCreate := func(tt *testing.T) {
		defer resetFake()
		assert := a.New(tt)

		// an error occurs while creating one of them
		createNil := &Endpoint{Name: "createNil", NsName: nsName, ServName: servName, Address: "1.1.1.1", Port: 1010}
		create := &Endpoint{Name: "create", NsName: nsName, ServName: servName, Address: "1.1.1.1", Port: 1010,
			Metadata: map[string]string{"key": "val"},
		}
		createErr := &Endpoint{Name: "create-error", Metadata: map[string]string{b.opMetaPair.Key: b.opMetaPair.Value, "key": "val"}}
		exists := &Endpoint{Name: "exists", NsName: nsName, ServName: servName, Metadata: map[string]string{b.opMetaPair.Key: b.opMetaPair.Value, "key": "val"}}
		del := &Endpoint{Name: "del", NsName: nsName, ServName: servName, Metadata: map[string]string{b.opMetaPair.Key: b.opMetaPair.Value, "key": "val"}}
		f.endpList[exists.Name] = exists
		f.endpList[del.Name] = del

		endpErrs, err := b.ManageServEndps(nsName, servName, []*Endpoint{create, createNil, createErr, exists})
		assert.NoError(err)
		assert.Len(endpErrs, 1)
		_, expErr := f.CreateEndp(&Endpoint{Name: "create-error"})
		assert.EqualError(expErr, endpErrs[createErr.Name].Error())

		// check that the function was actually called
		assert.Len(f.createdEndp, 2)
		assert.Contains(f.createdEndp, create.Name)
		assert.Contains(f.createdEndp, createNil.Name)
		assert.Equal(create.Name, f.endpList[create.Name].Name)
		assert.Equal(create.ServName, f.endpList[create.Name].ServName)
		assert.Equal(map[string]string{b.opMetaPair.Key: b.opMetaPair.Value, "key": "val"}, f.endpList[create.Name].Metadata)
		assert.Equal(createNil.Name, f.endpList[createNil.Name].Name)
		assert.Equal(createNil.ServName, f.endpList[createNil.Name].ServName)
		assert.Equal(map[string]string{b.opMetaPair.Key: b.opMetaPair.Value}, f.endpList[createNil.Name].Metadata)
		assert.Equal(create.NsName, f.endpList[create.Name].NsName)

		// assert that nothing was updated
		assert.Empty(f.updatedEndp)
		// assert that no one apart from del was deleted
		assert.Len(f.deletedEndp, 1)
		assert.Equal(f.deletedEndp[0], del.Name)
	}

	testValidation(t)
	testNotOwned(t)
	testOwnedDelete(t)
	testOwnedUpd(t)
	testOwnedCreate(t)
}
