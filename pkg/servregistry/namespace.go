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
	"path"

	"github.com/patrickmn/go-cache"
)

// This file contains functions that perform operations on namespaces,
// such as create/update/delete.
// These functions belong to a ServiceRegistryBroker, defined in
// broker.go

// ManageNs takes data of a namespace and performs the necessary steps
// to reflect that data to the service registry.
//
// For example: create a namespace in service registry or update it
// properly.
func (b *Broker) ManageNs(nsData *Namespace) (regNs *Namespace, err error) {
	// nsData: data of the namespace in Kubernetes (latest state)
	// regNs: data of the namespace currently in the service registry

	if b.Reg == nil {
		return nil, ErrServRegNotProvided
	}

	// -- Validate
	if nsData == nil {
		return nil, ErrNsNotProvided
	}

	if len(nsData.Name) == 0 {
		return nil, ErrNsNameNotProvided
	}
	cacheKey := path.Join("namespaces", nsData.Name)

	// -- Init
	b.lock.Lock()
	defer b.lock.Unlock()
	if nsData.Metadata == nil {
		nsData.Metadata = map[string]string{}
	}
	nsData.Metadata[b.opMetaPair.Key] = b.opMetaPair.Value
	l := b.log.WithName("ManageNs").WithValues("ns-name", nsData.Name)

	// -- Do stuff
	l.V(1).Info("going to load namespace from service registry")
	if val, found := b.cache.Get(cacheKey); found {
		regNs = val.(*Namespace)
		l.Info("retrieved from cache")
	} else {
		regNs, err = b.Reg.GetNs(nsData.Name)
		if err != nil {
			if err != ErrNotFound {
				l.Error(err, "error occurred while getting namespace from service registry")
				return
			}

			// If you're here, it means that the namespace does not exist.
			// Let's create it.
			l.V(1).Info("namespace does not exist in service registry, going to create it")
			regNs, err = b.Reg.CreateNs(nsData)
			if err != nil {
				l.Error(err, "error occurred while creating namespace in service registry")
				return
			}

			l.V(0).Info("namespace created correctly")
			regNs = nsData
		}
		b.cache.Add(cacheKey, regNs, cache.DefaultExpiration)
	}

	if by, exists := regNs.Metadata[b.opMetaPair.Key]; by != b.opMetaPair.Value || !exists {
		// If the namespace is not owned (as in, managed by) us, then it's
		// better not to touch it.
		l.V(0).Info("namespace is not owned by the operator and thus will not be updated")
		return
	}

	if !b.deepEqualMetadata(nsData.Metadata, regNs.Metadata) {
		l.V(1).Info("namespace metadata need to be updated")
		b.cache.Delete(cacheKey)

		regNs, err = b.Reg.UpdateNs(nsData)
		if err != nil {
			l.Error(err, "error while trying to update namespace in service registry")
			return nil, err
		}

		b.cache.Add(cacheKey, regNs, cache.DefaultExpiration)
	}

	return
}

// RemoveNs checks if a namespace can be safely deleted from the
// service registry before actually delete it. The second parameter forces
// the function to delete the namespace even if it is not empty.
// NOTE: setting forceNotEmpty to true will have no effect if the namespace
// contains services not owned by the operator, and therefore the namespace
// will not be deleted.
// NOTE: this function does *not* check if one of the contained services has
// endpoints not owned by the cnwan operator!
//
// For example: it checks if the namespace is actually owned by us.
func (b *Broker) RemoveNs(nsName string, forceNotEmpty bool) (err error) {
	if b.Reg == nil {
		return ErrServRegNotProvided
	}

	// -- Validate
	if len(nsName) == 0 {
		return ErrNsNameNotProvided
	}
	cacheKey := path.Join("namespaces", nsName)

	// -- Init
	b.lock.Lock()
	defer b.lock.Unlock()
	l := b.log.WithName("RemoveNs").WithValues("ns-name", nsName)

	// -- Do stuff
	l.V(1).Info("going to remove namespace from service registry")
	var regNs *Namespace

	if val, found := b.cache.Get(cacheKey); found {
		l.Info("retrieved from cache")
		regNs = val.(*Namespace)
	} else {
		// Load the namespace first
		regNs, err = b.Reg.GetNs(nsName)
		if err != nil {
			if err != ErrNotFound {
				l.Error(err, "error occurred while removing namespace from service registry")
				return
			}

			// If you're here, it means that the namespace does not exist.
			// This doesn't change anything for us.
			l.V(0).Info("namespace does not exist in service registry, going to stop here")
			return nil
		}
		b.cache.Add(cacheKey, regNs, cache.DefaultExpiration)
	}

	// Is it empty?
	var listServ []*Service
	l.V(1).Info("checking if namespace is empty before deleting")
	if val, found := b.cache.Get(path.Join(cacheKey, "services")); found {
		l.Info("retreived services list from cache")
		listServ = val.([]*Service)
		defer b.cache.Delete(path.Join(cacheKey, "services"))
	} else {
		// We're going to try and delete these services, so there is no point
		// in putting them in the cache.
		listServ, err = b.Reg.ListServ(nsName)
		if err != nil {
			return
		}
	}

	if len(listServ) > 0 && !forceNotEmpty {
		l.V(0).Info("namespace is not empty and will not be deleted from service registry")
		return ErrNsNotEmpty
	}

	l.V(0).Info("namespace is not empty: checking if it can be removed")
	servs := []string{}
	hasNotOwned := false
	for _, serv := range listServ {
		if by, exists := serv.Metadata[b.opMetaPair.Key]; by != b.opMetaPair.Value || !exists {
			l.V(0).Info("namespace contains services not owned by the operator")
			hasNotOwned = true
			continue
		}

		servs = append(servs, serv.Name)
	}

	if hasNotOwned {
		// There are some services not owned by the operator, so we must delete
		// services singularly
		l.V(0).Info("namespace contains services not owned by the operator and will not be removed from service registry")
		for _, servName := range servs {
			if delErr := b.Reg.DeleteServ(nsName, servName); delErr != nil {
				l.WithValues("serv-name", servName).Error(delErr, "error while deleting service from service registry")
			}
		}

		return ErrNsNotOwnedServs
	}

	if by, exists := regNs.Metadata[b.opMetaPair.Key]; by != b.opMetaPair.Value || !exists {
		// If the namespace is not owned (as in, managed by) us, then it's
		// better not to touch it.
		l.V(0).Info("WARNING: namespace is not owned by the operator and will not be removed from service registry")
		return ErrNsNotOwnedByOp
	}

	defer b.cache.Delete(cacheKey)
	err = b.Reg.DeleteNs(nsName)
	if err != nil {
		l.Error(err, "error while deleting namespace from service registry")
	} else {
		l.V(0).Info("namespace deleted from service registry successfully")
	}

	return
}
