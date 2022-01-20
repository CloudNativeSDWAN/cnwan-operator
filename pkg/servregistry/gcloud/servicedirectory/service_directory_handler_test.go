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
	"strings"
	"testing"

	sr "github.com/CloudNativeSDWAN/cnwan-operator/pkg/servregistry"
	a "github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestExtractData(t *testing.T) {
	nsName, servName := "ns", "serv"
	s := &Handler{}
	ips := []string{"10.10.10.10", "11.11.11.11"}
	ports := []int32{3333, 4444}
	nsToTest := &corev1.Namespace{
		ObjectMeta: v1.ObjectMeta{
			Name: nsName,
			Annotations: map[string]string{
				"key": "val",
			},
		},
	}
	servToTest := &corev1.Service{
		ObjectMeta: v1.ObjectMeta{
			Name:      servName,
			Namespace: nsName,
			Annotations: map[string]string{
				"key": "val",
			},
		},
		Spec: corev1.ServiceSpec{
			ExternalIPs: []string{ips[0], ips[1]},
			Ports: []corev1.ServicePort{
				{
					Port: ports[0],
					Name: "3333",
				},
				{
					Port: ports[1],
					Name: "4444",
				},
			},
		},
	}

	assert := a.New(t)

	ns, serv, endp, err := s.ExtractData(nsToTest, nil)
	assert.Nil(ns)
	assert.Nil(serv)
	assert.Nil(endp)
	assert.Equal(sr.ErrServNotProvided, err)

	ns, serv, endp, err = s.ExtractData(nil, servToTest)
	assert.Nil(ns)
	assert.Nil(serv)
	assert.Nil(endp)
	assert.Equal(sr.ErrNsNotProvided, err)

	ns, serv, endp, err = s.ExtractData(nsToTest, servToTest)
	assert.NotNil(ns)
	assert.NotNil(serv)
	assert.NotNil(endp)
	assert.NoError(err)
	assert.Equal(&sr.Namespace{
		Name:     nsName,
		Metadata: nsToTest.Annotations,
	}, ns)
	assert.Equal(&sr.Service{
		Name:     servName,
		NsName:   nsName,
		Metadata: servToTest.Annotations,
	}, serv)
	assert.Len(endp, 4)
	for _, e := range endp {
		assert.Contains(ips, e.Address)
		assert.Contains(ports, e.Port)
		assert.Empty(e.Metadata)
		assert.Equal(nsName, e.NsName)
		assert.Equal(servName, e.ServName)

		if !strings.HasPrefix(e.Name, servName+"-") {
			assert.Fail("endpoint name is incorrect. Should start with", servName, "but is", e.Name)
		}

		suffix := e.Name[len(servName)+1:]
		assert.Len(suffix, 10)
	}
}
