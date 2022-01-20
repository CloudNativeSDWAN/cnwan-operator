// Copyright © 2021 Cisco
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

package etcd

import (
	"fmt"
	"testing"

	sr "github.com/CloudNativeSDWAN/cnwan-operator/pkg/servregistry"
	"github.com/stretchr/testify/assert"
)

func TestKeyFromNames(t *testing.T) {
	a := assert.New(t)
	nsName := "ns-test"
	servName := "serv-test"
	endpName := "endp-test"

	cases := []struct {
		names  []string
		expRes *KeyBuilder
	}{
		{
			names:  []string{},
			expRes: &KeyBuilder{},
		},
		{
			names:  []string{nsName},
			expRes: &KeyBuilder{nsName: nsName},
		},
		{
			names:  []string{nsName, servName},
			expRes: &KeyBuilder{nsName: nsName, servName: servName},
		},
		{
			names:  []string{nsName, servName, endpName},
			expRes: &KeyBuilder{nsName: nsName, servName: servName, endpName: endpName},
		},
		{
			names:  []string{nsName, servName, endpName, "another"},
			expRes: &KeyBuilder{},
		},
	}

	for i, currCase := range cases {
		res := KeyFromNames(currCase.names...)
		if !a.Equal(currCase.expRes, res) {
			a.FailNow(fmt.Sprintf("case %d failed", i))
		}
	}
}

func TestKeyFromString(t *testing.T) {
	a := assert.New(t)

	cases := []struct {
		key    string
		expRes *KeyBuilder
	}{
		{
			key:    "/namespaces/test",
			expRes: &KeyBuilder{nsName: "test"},
		},
		{
			key:    "////namespaces/test///",
			expRes: &KeyBuilder{nsName: "test"},
		},
		{
			key:    "namespaces/test",
			expRes: &KeyBuilder{nsName: "test"},
		},
		{
			key:    defaultPrefix + "/namespaces/test",
			expRes: &KeyBuilder{nsName: "test"},
		},
		{
			key:    "something/other/entirely",
			expRes: &KeyBuilder{},
		},
		{
			key:    "namespaces/name/services",
			expRes: &KeyBuilder{},
		},
		{
			key:    "namespaces/name/services/name/endpoints/name/seventh",
			expRes: &KeyBuilder{},
		},
		{
			key:    "namespaces/name",
			expRes: &KeyBuilder{nsName: "name"},
		},
		{
			key:    "namespaces/name/servicesss/name",
			expRes: &KeyBuilder{},
		},
		{
			key:    "namespaces/name/services/name",
			expRes: &KeyBuilder{nsName: "name", servName: "name"},
		},
		{
			key:    "namespaces/name/services/name/eeendpoints/name",
			expRes: &KeyBuilder{},
		},
		{
			key:    "namespaces/name/services/name/endpoints/name",
			expRes: &KeyBuilder{nsName: "name", servName: "name", endpName: "name"},
		},
	}

	for i, currCase := range cases {
		res := KeyFromString(currCase.key)
		if !a.Equal(currCase.expRes, res) {
			a.FailNow(fmt.Sprintf("case %d failed", i))
		}
	}
}

func TestKeyFromObject(t *testing.T) {
	a := assert.New(t)
	unk := struct {
		Name     string
		Metadata map[string]string
	}{
		Name: "test",
		Metadata: map[string]string{
			"type": "unknown",
		},
	}
	ns := &sr.Namespace{
		Name:     "ns",
		Metadata: map[string]string{"type": "ns"},
	}
	serv := &sr.Service{
		NsName:   "ns",
		Name:     "serv",
		Metadata: map[string]string{"type": "serv"},
	}
	endp := &sr.Endpoint{
		NsName:   "ns",
		ServName: "serv",
		Name:     "endp",
		Metadata: map[string]string{"type": "endp"},
	}

	cases := []struct {
		arg    interface{}
		expRes *KeyBuilder
		expErr error
	}{
		{
			arg:    nil,
			expErr: ErrNilObject,
		},
		{
			arg:    &unk,
			expErr: ErrUnknownObject,
		},
		{
			arg:    &sr.Namespace{},
			expErr: sr.ErrNsNameNotProvided,
		},
		{
			arg:    ns,
			expRes: &KeyBuilder{nsName: ns.Name},
		},
		{
			arg:    &sr.Service{},
			expErr: sr.ErrNsNameNotProvided,
		},
		{
			arg:    &sr.Service{NsName: ns.Name},
			expErr: sr.ErrServNameNotProvided,
		},
		{
			arg:    serv,
			expRes: &KeyBuilder{nsName: serv.NsName, servName: serv.Name},
		},
		{
			arg:    &sr.Endpoint{},
			expErr: sr.ErrNsNameNotProvided,
		},
		{
			arg:    &sr.Endpoint{NsName: endp.NsName},
			expErr: sr.ErrServNameNotProvided,
		},
		{
			arg:    &sr.Endpoint{NsName: endp.NsName, ServName: endp.ServName},
			expErr: sr.ErrEndpNameNotProvided,
		},
		{
			arg:    endp,
			expRes: &KeyBuilder{nsName: endp.NsName, servName: endp.ServName, endpName: endp.Name},
		},
	}

	for i, currCase := range cases {
		res, err := KeyFromServiceRegistryObject(currCase.arg)
		errRes := a.Equal(currCase.expRes, res)
		errErr := a.Equal(currCase.expErr, err)
		if !errRes || !errErr {
			a.FailNow(fmt.Sprintf("case %d failed", i))
		}
	}
}

func TestClone(t *testing.T) {
	a := assert.New(t)
	nsName := "ns-test"
	servName := "serv-test"
	endpName := "endp-test"
	k := KeyFromNames(nsName, servName, endpName)
	c := k.Clone()

	a.Equal(k, c)
}

func TestObjectType(t *testing.T) {
	a := assert.New(t)
	nsName := "ns-test"
	servName := "serv-test"
	endpName := "endp-test"

	cases := []struct {
		arg    *KeyBuilder
		expRes ObjectType
	}{
		{
			arg:    &KeyBuilder{},
			expRes: UnknownOrInvalidObject,
		},
		{
			arg:    (&KeyBuilder{}).SetEndpoint(endpName),
			expRes: UnknownOrInvalidObject,
		},
		{
			arg:    (&KeyBuilder{}).SetService(servName),
			expRes: UnknownOrInvalidObject,
		},
		{
			arg:    (&KeyBuilder{}).SetNamespace(nsName).SetEndpoint(endpName),
			expRes: UnknownOrInvalidObject,
		},
		{
			arg:    (&KeyBuilder{}).SetNamespace(nsName),
			expRes: NamespaceObject,
		},
		{
			arg:    (&KeyBuilder{}).SetNamespace(nsName).SetService(servName),
			expRes: ServiceObject,
		},
		{
			arg:    (&KeyBuilder{}).SetNamespace(nsName).SetService(servName).SetEndpoint(endpName),
			expRes: EndpointObject,
		},
	}

	for i, currCase := range cases {
		if !a.Equal(currCase.expRes, currCase.arg.ObjectType()) {
			a.FailNow(fmt.Sprintf("case %d failed", i))
		}
	}
}

func TestString(t *testing.T) {
	a := assert.New(t)
	nsName := "ns-test"
	servName := "serv-test"
	endpName := "endp-test"

	cases := []struct {
		arg    *KeyBuilder
		expRes string
	}{
		{
			arg:    &KeyBuilder{},
			expRes: "",
		},
		{
			arg:    (&KeyBuilder{}).SetEndpoint(endpName),
			expRes: "",
		},
		{
			arg:    (&KeyBuilder{}).SetService(servName),
			expRes: "",
		},
		{
			arg:    (&KeyBuilder{}).SetNamespace(nsName).SetEndpoint(endpName),
			expRes: "",
		},
		{
			arg:    (&KeyBuilder{}).SetNamespace(nsName),
			expRes: fmt.Sprintf("%s/%s", namespacePrefix, nsName),
		},
		{
			arg:    (&KeyBuilder{}).SetNamespace(nsName).SetService(servName),
			expRes: fmt.Sprintf("%s/%s/%s/%s", namespacePrefix, nsName, servicePrefix, servName),
		},
		{
			arg:    (&KeyBuilder{}).SetNamespace(nsName).SetService(servName).SetEndpoint(endpName),
			expRes: fmt.Sprintf("%s/%s/%s/%s/%s/%s", namespacePrefix, nsName, servicePrefix, servName, endpointPrefix, endpName),
		},
	}

	for i, currCase := range cases {
		if !a.Equal(currCase.expRes, currCase.arg.String()) {
			a.FailNow(fmt.Sprintf("case %d failed", i))
		}
	}
}

func TestGetNamespace(t *testing.T) {
	a := assert.New(t)
	k := &KeyBuilder{}
	a.Empty(k.GetNamespace())

	nsName := "ns-test"
	k.nsName = nsName
	a.Equal(nsName, k.GetNamespace())

	k.nsName = nsName
	k.endpName = "test"
	a.Empty(k.GetNamespace())
}

func TestGetService(t *testing.T) {
	a := assert.New(t)
	k := &KeyBuilder{}
	a.Empty(k.GetService())

	servName := "serv-test"
	k.servName = servName
	a.Empty(k.GetService())

	k.nsName = "ns-test"
	a.Equal(servName, k.GetService())
}

func TestGetEndpoint(t *testing.T) {
	a := assert.New(t)
	k := &KeyBuilder{}
	a.Empty(k.GetEndpoint())

	endpName := "endp-test"
	k.endpName = endpName
	a.Empty(k.GetEndpoint())

	k.nsName = "ns-test"
	k.servName = "serv-test"
	a.Equal(endpName, k.GetEndpoint())
}
