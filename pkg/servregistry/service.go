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

// This file contains functions that perform operations on services,
// such as create/update/delete.
// These functions belong to a ServiceRegistryBroker, defined in
// broker.go

// ManageServ takes data from a service and peforms all necessary
// operations to reflect that data to the service registry
//
// For example: create a service in service registry or update it
// properly.
func (b *Broker) ManageServ(servData *Service) (regServ *Service, err error) {
	// servData: data of the service in Kubernetes (latest update)
	// regServ: data of the service currently in the service registry

	// As of now, ManageServ and ProcessNsChange only differ in the type
	// they work with (namespaces vs services), everything else is basically
	// duplicate code. Let's adopt a pragmatic approach: we leave it like this
	// since it is easier to understand and make it better later.

	if b.Reg == nil {
		return nil, ErrServRegNotProvided
	}

	// -- Validate
	if servData == nil {
		return nil, ErrServNotProvided
	}

	if len(servData.Name) == 0 {
		return nil, ErrServNameNotProvided
	}

	if len(servData.NsName) == 0 {
		return nil, ErrNsNameNotProvided
	}

	// -- Init
	b.lock.Lock()
	defer b.lock.Unlock()
	if servData.Metadata == nil {
		servData.Metadata = map[string]string{}
	}
	servData.Metadata[b.opKey] = b.opVal
	l := b.log.WithName("ManageServ").WithValues("serv-name", servData.Name)

	// -- Do stuff
	l.V(1).Info("going to load service from service registry")

	regServ, err = b.Reg.GetServ(servData.NsName, servData.Name)
	if err != nil {
		if err != ErrNotFound {
			l.Error(err, "error occurred while getting service from service registry")
			return
		}

		// If you're here, it means that the service does not exist.
		// Let's create it.
		l.V(1).Info("service does not exist in service registry, going to create it")
		regServ, err = b.Reg.CreateServ(servData)
		if err != nil {
			l.Error(err, "error occurred while creating service in service registry")
			return
		}

		l.V(0).Info("service created correctly")
		regServ = servData
	}

	if by, exists := regServ.Metadata[b.opKey]; by != b.opVal || !exists {
		// If the service is not owned (as in, managed by) us, then it's
		// better not to touch it.
		l.V(0).Info("service is not owned by the operator and thus will not be updated")
		return
	}

	if !b.deepEqualMetadata(servData.Metadata, regServ.Metadata) {
		l.V(1).Info("service metadata need to be updated")
		regServ, err = b.Reg.UpdateServ(servData)
		if err != nil {
			l.Error(err, "error while trying to update service in service registry")
			return nil, err
		}
	}

	return
}
