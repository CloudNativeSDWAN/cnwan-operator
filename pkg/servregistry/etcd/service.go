// Copyright © 2021 Cisco
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
	"context"

	sr "github.com/CloudNativeSDWAN/cnwan-operator/pkg/servregistry"
	"gopkg.in/yaml.v3"
)

// GetServ returns the service if exists.
func (e *etcdServReg) GetServ(nsName, servName string) (*sr.Service, error) {
	key, err := KeyFromServiceRegistryObject(&sr.Service{NsName: nsName, Name: servName})
	if err != nil {
		return nil, err
	}

	ctx, canc := context.WithTimeout(e.mainCtx, defaultTimeout)
	defer canc()

	serv, err := e.getOne(ctx, key)
	if err != nil {
		return nil, err
	}

	return serv.(*sr.Service), nil
}

// ListServ returns a list of services inside the provided namespace.
func (e *etcdServReg) ListServ(nsName string) (servList []*sr.Service, err error) {
	if !KeyFromNames(nsName).IsValid() {
		return nil, sr.ErrNsNameNotProvided
	}

	ctx, canc := context.WithTimeout(e.mainCtx, defaultTimeout)
	defer canc()

	err = e.getList(ctx, KeyFromNames(nsName), func(item []byte) {
		var serv *sr.Service
		if err := yaml.Unmarshal(item, &serv); err == nil {
			servList = append(servList, serv)
			return
		}
	})

	if err != nil {
		return nil, err
	}

	return
}

// CreateServ creates the service.
func (e *etcdServReg) CreateServ(serv *sr.Service) (*sr.Service, error) {
	ctx, canc := context.WithTimeout(e.mainCtx, defaultTimeout)
	defer canc()

	if err := e.put(ctx, serv, false); err != nil {
		return nil, err
	}

	return serv, nil
}

// UpdateServ updates the service.
func (e *etcdServReg) UpdateServ(serv *sr.Service) (*sr.Service, error) {
	ctx, canc := context.WithTimeout(e.mainCtx, defaultTimeout)
	defer canc()

	if err := e.put(ctx, serv, true); err != nil {
		return nil, err
	}

	return serv, nil
}

// DeleteServ deletes the service.
func (e *etcdServReg) DeleteServ(nsName, servName string) error {
	// TODO: implement me
	return nil
}
