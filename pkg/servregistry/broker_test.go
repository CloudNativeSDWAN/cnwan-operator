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
	"errors"
	"testing"

	a "github.com/stretchr/testify/assert"
)

type fakeServReg struct {
	nsList   map[string]*Namespace
	servList map[string]*Service
	endpList map[string]*Endpoint

	createdNs []string
	updatedNs []string
	deletedNs []string

	createdServ []string
	updatedServ []string
	deletedServ []string

	createdEndp []string
	updatedEndp []string
	deletedEndp []string
}

func newFakeInt() ServiceRegistry {
	return &fakeServReg{
		nsList:      map[string]*Namespace{},
		servList:    map[string]*Service{},
		endpList:    map[string]*Endpoint{},
		createdNs:   []string{},
		updatedNs:   []string{},
		deletedNs:   []string{},
		createdServ: []string{},
		updatedServ: []string{},
		deletedServ: []string{},
		createdEndp: []string{},
		updatedEndp: []string{},
		deletedEndp: []string{},
	}
}

func newFakeStruct() *fakeServReg {
	return &fakeServReg{
		nsList:      map[string]*Namespace{},
		servList:    map[string]*Service{},
		endpList:    map[string]*Endpoint{},
		createdNs:   []string{},
		updatedNs:   []string{},
		deletedNs:   []string{},
		createdServ: []string{},
		updatedServ: []string{},
		deletedServ: []string{},
		createdEndp: []string{},
		updatedEndp: []string{},
		deletedEndp: []string{},
	}
}

func (f *fakeServReg) GetNs(name string) (*Namespace, error) {
	if name == "get-error" {
		return nil, errors.New("error")
	}

	ns, exists := f.nsList[name]
	if !exists {
		return nil, ErrNotFound
	}

	return ns, nil
}

func (f *fakeServReg) ListNs() ([]*Namespace, error) {
	if _, exists := f.nsList["list-error"]; exists {
		return nil, errors.New("error")
	}

	list := []*Namespace{}
	for _, n := range f.nsList {
		list = append(list, n)
	}

	return list, nil
}

func (f *fakeServReg) CreateNs(ns *Namespace) (*Namespace, error) {
	if ns.Name == "create-error" {
		return nil, errors.New("error")
	}

	_, exists := f.nsList[ns.Name]
	if exists {
		return nil, ErrAlreadyExists
	}

	f.nsList[ns.Name] = ns

	return f.nsList[ns.Name], nil
}

func (f *fakeServReg) UpdateNs(ns *Namespace) (*Namespace, error) {
	if ns.Name == "update-error" {
		return nil, errors.New("error")
	}

	_, exists := f.nsList[ns.Name]
	if !exists {
		return nil, ErrNotFound
	}

	f.nsList[ns.Name] = ns
	f.updatedNs = append(f.updatedEndp, ns.Name)

	return f.nsList[ns.Name], nil
}

func (f *fakeServReg) DeleteNs(nsName string) error {
	if nsName == "delete-error" {
		return errors.New("error")
	}

	_, exists := f.nsList[nsName]
	if !exists {
		return ErrNotFound
	}

	delete(f.nsList, nsName)
	f.deletedNs = append(f.deletedNs, nsName)

	del := []string{}
	for sname, s := range f.servList {
		if s.NsName == nsName {
			del = append(del, sname)
		}
	}

	for _, sname := range del {
		delete(f.servList, sname)
		f.deletedServ = append(f.deletedServ, sname)
	}

	return nil
}

func (f *fakeServReg) GetServ(nsName, servName string) (*Service, error) { return nil, nil }

func (f *fakeServReg) ListServ(nsName string) ([]*Service, error) { return nil, nil }

func (f *fakeServReg) CreateServ(serv *Service) (*Service, error) { return nil, nil }

func (f *fakeServReg) UpdateServ(serv *Service) (*Service, error) { return nil, nil }

func (f *fakeServReg) DeleteServ(nsName, servName string) error { return nil }

func (f *fakeServReg) GetEndp(nsName, servName, endpName string) (*Endpoint, error) { return nil, nil }

func (f *fakeServReg) ListEndp(nsName, servName string) ([]*Endpoint, error) { return nil, nil }

func (f *fakeServReg) CreateEndp(endp *Endpoint) (*Endpoint, error) { return nil, nil }

func (f *fakeServReg) UpdateEndp(endp *Endpoint) (*Endpoint, error) { return nil, nil }

func (f *fakeServReg) DeleteEndp(nsName, servName, endpName string) error { return nil }

func TestNewBroker(t *testing.T) {
	// prepare
	var f *fakeServReg
	assert := a.New(t)
	b, err := NewBroker(nil, "", "")

	assert.Nil(b)
	assert.Equal(ErrServRegNotProvided, err)

	b, err = NewBroker(f, "", "")
	assert.NotNil(b)
	assert.NoError(err)
	assert.Equal(b.opKey, defOpKey)
	assert.Equal(b.opVal, defOpVal)

	b, err = NewBroker(f, "test", "testing")
	assert.Equal(b.opKey, "test")
	assert.Equal(b.opVal, "testing")
}
