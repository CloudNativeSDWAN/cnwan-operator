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

// This file contains functions that perform operations on endpoints,
// such as create/update/delete.
// These functions belong to a ServiceRegistryBroker, defined in
// broker.go

// ManageServEndps updates the endpoints of the provided service
// with the provided endpoints. Additionally, it removes endpoints from the
// service registry if they have been removed from the Kubernetes service.
// The first returned value is a key-value map where the key is the name of the
// endpoint and value the error occurred in case the endpoint failed to
// update/delete.
// The second error value is only returned if the function could not perform
// any operation at all, such as when some data is invalid or it could not
// load the endpoints from the service registry.
//
// NOTE: if the array is empty, then *all* the endpoints will be removed from
// the service registry, apart from those not owned by the cnwan operator.
// NOTE: NsName and ServName in endpsData will be ignored by the function,
// as only the first two arguments will be considered.
// This is because endpoints must all belong to the same service.
//
// For example: updates the metadata, address and/or of the endpoints.
func (b *Broker) ManageServEndps(nsName, servName string, endpsData []*Endpoint) (endpErrs map[string]error, err error) {
	// endpsData: data of the endpoints in Kubernetes (latest update)
	// regEndps: data of the endpoints currently in the service registry

	if b.Reg == nil {
		return nil, ErrServRegNotProvided
	}

	// -- Validate
	if len(nsName) == 0 {
		return nil, ErrNsNameNotProvided
	}

	if len(servName) == 0 {
		return nil, ErrServNameNotProvided
	}

	// -- Init
	b.lock.Lock()
	defer b.lock.Unlock()
	l := b.log.WithName("ManageServEndps").WithValues("serv-name", servName, "ns-name", nsName)

	// -- Do stuff
	l.V(1).Info("going to update endpoints in service registry")

	endpsMap := map[string]*Endpoint{}
	for _, endp := range endpsData {
		endpsMap[endp.Name] = endp
		if endpsMap[endp.Name].Metadata == nil {
			endpsMap[endp.Name].Metadata = map[string]string{}
		}
		endpsMap[endp.Name].Metadata[b.opMetaPair.Key] = b.opMetaPair.Value
	}
	endpErrs = map[string]error{}

	// Check what changed
	var listRegEndps []*Endpoint
	listRegEndps, err = b.Reg.ListEndp(nsName, servName)
	if err != nil {
		return
	}

	for _, regEndp := range listRegEndps {
		// endpData: the endpoint as it is in Kubernetes
		// regEndp: the endpoint as it is registered in the service registry

		l := l.WithValues("endp-name", regEndp.Name)

		endpData, exists := endpsMap[regEndp.Name]

		if owner, ownerExists := regEndp.Metadata[b.opMetaPair.Key]; owner != b.opMetaPair.Value || !ownerExists {
			l.V(0).Info("endpoint is not managed by the cnwan operator and is going to be ignored")
			endpErrs[regEndp.Name] = ErrEndpNotOwnedByOp
			delete(endpsMap, regEndp.Name)
			continue
		}

		if !exists {
			l.V(1).Info("going to delete endpoint from service registry")

			// This endpoint is not in the k8s service.
			// We gotta delete this from the service registry.
			delErr := b.Reg.DeleteEndp(nsName, servName, regEndp.Name)

			if delErr != nil {
				l.Error(delErr, "error while deleting endpoint from service registry")
				endpErrs[regEndp.Name] = delErr
			} else {
				l.V(0).Info("endpoint deleted from service registry")
			}

			continue
		}

		// This endpoint exists in the k8s service as well.
		// We gotta check if the k8s one is different.
		if endpData.Address != regEndp.Address || endpData.Port != regEndp.Port ||
			!b.deepEqualMetadata(endpData.Metadata, regEndp.Metadata) {
			_, updErr := b.Reg.UpdateEndp(endpData)

			if updErr != nil {
				l.Error(updErr, "error while updating endpoint in service registry")
				endpErrs[regEndp.Name] = updErr

			} else {
				l.V(0).Info("endpoint updated in service registry")
			}
		}

		// Remove it from the map, so we don't create this later
		delete(endpsMap, regEndp.Name)
	}

	// Create the new ones
	for _, endpData := range endpsMap {
		l := l.WithValues("endp-name", endpData.Name)

		_, createErr := b.Reg.CreateEndp(endpData)
		if createErr != nil {
			l.Error(createErr, "error while creating endpoint in service registry")
			endpErrs[endpData.Name] = createErr
			continue
		}

		l.V(0).Info("endpoint created in service registry")
	}

	return
}
