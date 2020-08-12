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

package servicedirectory

import (
	"context"
	"io/ioutil"
	"strings"
	"sync"

	sd "cloud.google.com/go/servicedirectory/apiv1beta1"
	"github.com/CloudNativeSDWAN/cnwan-operator/types"
	"github.com/go-logr/logr"
	"google.golang.org/api/option"
	sdpb "google.golang.org/genproto/googleapis/cloud/servicedirectory/v1beta1"
)

type sdHandler struct {
	client       *sd.RegistrationClient
	lookupClient *sd.LookupClient
	log          logr.Logger
	lock         sync.Mutex
}

// NewHandler creates a handler for service directory
func NewHandler(credsPath string, logger logr.Logger) (Handler, error) {
	ctx := context.Background()
	jsonBytes, err := ioutil.ReadFile(credsPath)
	if err != nil {
		return nil, err
	}

	c, err := sd.NewRegistrationClient(ctx, option.WithCredentialsJSON(jsonBytes))
	if err != nil {
		return nil, err
	}

	l, err := sd.NewLookupClient(ctx, option.WithCredentialsJSON(jsonBytes))
	if err != nil {
		return nil, err
	}

	return &sdHandler{
		client:       c,
		lookupClient: l,
		log:          logger,
	}, nil
}

// CreateOrUpdateService is used to create or update an existing service.
func (s *sdHandler) CreateOrUpdateService(servSnap types.ServiceSnapshot) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	//------------------------------
	// Init
	//------------------------------

	l := s.log.WithName("CreateOrUpdateService").WithValues("service-name", servSnap.Name, "service-namespace", servSnap.Namespace)

	if len(servSnap.Endpoints) == 0 {
		l.V(0).Info("service has no endpoints and will not be added in service directory")
		return nil
	}

	ctx := context.Background()
	var sdNs *sdpb.Namespace
	var sdServ *sdpb.Service
	nsCreated := false

	//------------------------------
	// Namespace
	//------------------------------

	sdNs, err := s.getNamespace(ctx, servSnap.Namespace)
	if err != nil {
		l.Error(err, "error while getting the namespace from service directory")
		return err
	}

	if sdNs == nil {
		// Namespace does not exist, let's create it

		sdNs, err = s.createNamespace(ctx, servSnap.Namespace)
		if err != nil {
			l.Error(err, "error while creating namespace in service directory")
			return err
		}
		nsCreated = true
	}

	//------------------------------
	// Service
	//------------------------------

	if !nsCreated {
		sdServ, err = s.getService(ctx, servSnap.Namespace, servSnap.Name)
		if err != nil {
			l.Error(err, "error while getting the service from service directory")
			return err
		}
	}

	if sdServ == nil {
		// The service does not exist, we need to create it

		sdServ, err = s.createService(ctx, servSnap)
		if err != nil {
			l.Error(err, "error while creating service in service directory")

			if !nsCreated {
				// The namespace already existed, so no need to delete it.
				// Just return the error.
				return err
			}

			// If the namespace was just created, it must be deleted
			// because it means it is empty now.
			if err := s.client.DeleteNamespace(ctx, &sdpb.DeleteNamespaceRequest{
				Name: s.getResourcePath(servSnap.Namespace),
			}); err != nil {
				l.Error(err, "error while deleting namespace from service directory after failure in creating service")
			}

			// Return the error in creating the service, not the one
			// in deleting the namespace. The service error is more
			// important right now.
			return err
		}
	}

	// Are metadata key/values different?
	if !s.deepEqualMetadata(sdServ.Metadata, servSnap.Metadata) {
		l.V(1).Info("difference in metadata detected: the service will be updated")
		// NOTE: this will ALWAYS be executed in case user does not exclude
		// kubectl.kubernetes.io/last-applied-configuration

		// Metadata are different: update the service
		sdServ.Metadata = servSnap.Metadata
		sdServ, err = s.updateService(ctx, sdServ)
		if err != nil {
			l.Error(err, "error while updating the service in service directory")
			return err
		}
	}

	//------------------------------
	// Endpoint(s)
	//------------------------------

	sdEndpoints := s.client.ListEndpoints(ctx, &sdpb.ListEndpointsRequest{
		Parent: s.getResourcePath(servSnap.Namespace, servSnap.Name),
	})

	for {
		sdEndp, iterErr := sdEndpoints.Next()
		if iterErr != nil {
			break
		}

		le := l.WithName("Endpoint-check").WithValues("endpoint-name", sdEndp.Name)

		// What do we have to do with this?
		byOp, action := s.getEndpointAction(sdEndp, servSnap.Endpoints)
		if !byOp {
			le.V(0).Info("endpoint is not owned by the operator and will be skipped")
			continue
		}

		if action == endpointDelete {
			if err := s.client.DeleteEndpoint(ctx, &sdpb.DeleteEndpointRequest{
				Name: sdEndp.Name,
			}); err != nil {
				le.Error(err, "error while deleting the endpoint from service directory and must be deleted manually")
			}
		}

		if action == endpointUpdate {
			splitName := strings.Split(sdEndp.Name, "/")
			sdName := splitName[len(splitName)-1]
			endpSnap := servSnap.Endpoints[sdName]
			sdEndp.Metadata = endpSnap.Metadata
			if _, err := s.updateEndpoint(ctx, sdEndp); err != nil {
				le.Error(err, "error while updating endpoint in service directory and must be updated manually")
			}
			delete(servSnap.Endpoints, sdName)
		}
	}

	// Create the missing endpoints
	for _, endpSnap := range servSnap.Endpoints {
		if _, err := s.createEndpoint(ctx, servSnap.Namespace, servSnap.Name, endpSnap); err != nil {
			l.WithName("Endpoint-create").WithValues("endpoint-name", endpSnap.Name).
				Error(err, "error while creating the endpoint in service directory and must be created manually")
		}
	}

	l.V(0).Info("service processed successfully")
	return nil
}

// DeleteService is used to remove a service.
func (s *sdHandler) DeleteService(nsName, servName string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	//------------------------------
	// Init
	//------------------------------

	l := s.log.WithName("DeleteService").WithValues("service-name", servName, "service-namespace", nsName)
	ctx := context.Background()

	//------------------------------
	// The service
	//------------------------------

	sdServ, err := s.getService(ctx, nsName, servName)
	if err != nil {
		l.Error(err, "error while getting the service from service directory")
		return err
	}

	if sdServ == nil {
		l.V(0).Info("service does not exist in service directory")
		return nil
	}

	if createdBy := sdServ.Metadata["owner"]; createdBy != "cnwan-operator" {
		l.V(1).Info("service is not owned by the operator and won't be deleted from service directory")
		return nil
	}

	// Does it have any endpoints that are not managed by us?
	sdEndpoints := s.client.ListEndpoints(ctx, &sdpb.ListEndpointsRequest{
		Parent: s.getResourcePath(nsName, servName),
	})

	for {
		sdEndp, iterErr := sdEndpoints.Next()
		if iterErr != nil {
			break
		}

		if createdBy := sdEndp.Metadata["owner"]; createdBy != "cnwan-operator" {
			l.V(0).Info("service contains endpoints not owned by the operator and will not be deleted from service directory")
			return nil
		}
	}

	// Delete the service
	if err := s.client.DeleteService(ctx, &sdpb.DeleteServiceRequest{
		Name: s.getResourcePath(nsName, servName),
	}); err != nil {
		l.Error(err, "error while deleting the service from service directory")
		return err
	}

	l.V(0).Info("service deleted successfully from service directory")

	//------------------------------
	// The namespace
	//------------------------------

	sdNs, err := s.getNamespace(ctx, nsName)
	if err != nil {
		l.Error(err, "error while getting the namespace from service directory")
		return err
	}

	if nsCreatedBy := sdNs.Labels["owner"]; nsCreatedBy != "cnwan-operator" {
		l.V(0).Info("namespace is not owned by the operator and will not be deleted from service directory")
		return nil
	}

	// Is this empty?
	servIter := s.client.ListServices(ctx, &sdpb.ListServicesRequest{
		Parent: s.getResourcePath(nsName),
	})

	if servIter.PageInfo().Remaining() > 0 {
		// Namespace is not empty, stop here.
		return nil
	}

	for {
		currServ, iterErr := servIter.Next()
		if iterErr != nil {
			break
		}

		if createdBy := currServ.Metadata["owner"]; createdBy != "cnwan-operator" {
			l.V(0).Info("namespace contains services not owned by the operator and will not be deleted from service directory")
			return nil
		}

		// If you're here, it means that there is at least one service managed
		// by the operator. Therefore the namespace must not be deleted.
		return nil
	}

	if err := s.client.DeleteNamespace(ctx, &sdpb.DeleteNamespaceRequest{
		Name: s.getResourcePath(nsName),
	}); err != nil {
		l.Error(err, "error while deleting namespace from service directory")
		return err
	}

	l.V(0).Info("namespace deleted successfully from namespace")

	return nil
}
