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

package cloudmap

import (
	"context"
	"errors"
	"time"

	sr "github.com/CloudNativeSDWAN/cnwan-operator/pkg/servregistry"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery/types"
)

func (h *Handler) listOrGetService(nsName string, name *string) ([]*sr.Service, []*cloudMapIDs, error) {
	// NOTE: as for other resources on Cloud Map, we cannot get a resource
	// by its name but *only* with its AWS-generated ID. Thus, here too we
	// need to actually search for the service rather than its name. Same
	// reason why we got the namespace ID above.
	_, ids, err := h.listOrGetNamespace(&nsName)
	if err != nil {
		return nil, nil, err
	}
	if len(ids) == 0 {
		return nil, nil, sr.ErrNotFound
	}
	nsID := ids[0].id

	ctx, canc := context.WithTimeout(h.mainCtx, time.Minute)
	defer canc()

	servs, err := h.Client.ListServices(ctx, &servicediscovery.ListServicesInput{
		Filters: []types.ServiceFilter{
			{
				Name:      types.ServiceFilterNameNamespaceId,
				Values:    []string{nsID},
				Condition: types.FilterConditionEq,
			},
		},
	})
	if err != nil {
		return nil, nil, err
	}

	servList := []*sr.Service{}
	servIDs := []*cloudMapIDs{}
	for _, serv := range servs.Services {
		if name != nil && aws.ToString(serv.Name) != *name {
			// move on, this is not the service you're looking for.
			continue
		}
		arn := aws.ToString(serv.Arn)
		id := aws.ToString(serv.Id)

		servFound := &sr.Service{
			NsName:   nsName,
			Name:     aws.ToString(serv.Name),
			Metadata: map[string]string{},
		}

		servFound.Metadata = func() map[string]string {
			tagCtx, tagCanc := context.WithTimeout(h.mainCtx, 30*time.Second)
			defer tagCanc()

			tags, err := h.Client.ListTagsForResource(tagCtx, &servicediscovery.ListTagsForResourceInput{
				ResourceARN: serv.Arn,
			})
			if err != nil {
				h.log.WithName("ListTagsForResource").Info("error while getting tags", "error", err, "name", servFound.Name)
				return map[string]string{}
			}

			return fromTagsSliceToMap(tags.Tags)
		}()

		if name != nil {
			// if you're here, it means that you are indeed the service we
			// are looking for.
			return []*sr.Service{servFound}, []*cloudMapIDs{{arn, id}}, nil
		}

		servList = append(servList, servFound)
		servIDs = append(servIDs, &cloudMapIDs{arn, id})
	}

	return servList, servIDs, nil
}

// GetServ returns the service if exists.
func (h *Handler) GetServ(nsName, servName string) (*sr.Service, error) {
	if nsName == "" {
		return nil, sr.ErrNsNameNotProvided
	}
	if servName == "" {
		return nil, sr.ErrServNameNotProvided
	}

	// -- then get the service
	servs, _, err := h.listOrGetService(nsName, &servName)
	if err != nil {
		return nil, err
	}
	if len(servs) == 0 {
		return nil, sr.ErrNotFound
	}

	return servs[0], nil
}

// ListServ returns a list of services inside the provided namespace.
func (h *Handler) ListServ(nsName string) (servList []*sr.Service, err error) {
	if nsName == "" {
		return nil, sr.ErrNsNameNotProvided
	}

	servs, _, err := h.listOrGetService(nsName, nil)
	if err != nil {
		return nil, err
	}

	return servs, nil
}

// CreateServ creates the service.
func (h *Handler) CreateServ(serv *sr.Service) (*sr.Service, error) {
	if serv == nil {
		return nil, sr.ErrServNotProvided
	}
	if serv.NsName == "" {
		return nil, sr.ErrNsNameNotProvided
	}
	if serv.Name == "" {
		return nil, sr.ErrServNameNotProvided
	}

	_, ids, err := h.listOrGetNamespace(&serv.NsName)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, sr.ErrNotFound
	}
	nsID := ids[0].id

	ctx, canc := context.WithTimeout(h.mainCtx, time.Minute)
	defer canc()

	_, err = h.Client.CreateService(ctx, &servicediscovery.CreateServiceInput{
		Name:        aws.String(serv.Name),
		NamespaceId: aws.String(nsID),
		Tags:        fromMapToTagsSlice(serv.Metadata),
		Type:        types.ServiceTypeOptionHttp,
	})
	if err == nil {
		return serv, nil
	}

	// -- the probability of this occurring is very rare but we're handling
	// this anyways
	var oe *types.NamespaceNotFound
	if errors.As(err, &oe) {
		return nil, sr.ErrNotFound
	}

	var os *types.ServiceAlreadyExists
	if errors.As(err, &os) {
		return nil, sr.ErrAlreadyExists
	}

	// any other error
	return nil, err
}

// UpdateServ updates the service.
func (h *Handler) UpdateServ(serv *sr.Service) (*sr.Service, error) {
	if serv == nil {
		return nil, sr.ErrServNotProvided
	}
	if serv.NsName == "" {
		return nil, sr.ErrNsNameNotProvided
	}
	if serv.Name == "" {
		return nil, sr.ErrServNameNotProvided
	}

	_, servIDs, err := h.listOrGetService(serv.NsName, &serv.Name)
	if err != nil {
		return nil, err
	}
	if len(servIDs) == 0 {
		return nil, sr.ErrNotFound
	}

	err = h.tagResource(servIDs[0].arn, fromMapToTagsSlice(serv.Metadata))
	if err == nil {
		return serv, nil
	}

	// -- the probability of occurring is very rare but we're
	// handling it anyways
	var oe *types.NamespaceNotFound
	if errors.As(err, &oe) {
		return nil, sr.ErrNotFound
	}

	var os *types.ServiceAlreadyExists
	if errors.As(err, &os) {
		return nil, sr.ErrAlreadyExists
	}

	// any other error
	return nil, err
}

// DeleteServ deletes the service.
func (h *Handler) DeleteServ(nsName, servName string) error {
	if nsName == "" {
		return sr.ErrNsNameNotProvided
	}
	if servName == "" {
		return sr.ErrServNameNotProvided
	}

	_, servIDs, err := h.listOrGetService(nsName, &servName)
	if err != nil {
		return err
	}
	if len(servIDs) == 0 {
		return sr.ErrNotFound
	}
	servID := servIDs[0].id

	ctx, canc := context.WithTimeout(h.mainCtx, time.Minute)
	defer canc()

	_, err = h.Client.DeleteService(ctx, &servicediscovery.DeleteServiceInput{
		Id: aws.String(servID),
	})
	if err == nil {
		return nil
	}

	var oe *types.ServiceNotFound
	if errors.As(err, &oe) {
		return sr.ErrNotFound
	}

	// any other error
	return err
}
