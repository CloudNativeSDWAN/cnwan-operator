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

// CreateNs creates the namespace.
func (h *Handler) CreateNs(ns *sr.Namespace) (*sr.Namespace, error) {
	if ns == nil {
		return nil, sr.ErrNsNotProvided
	}
	if ns.Name == "" {
		return nil, sr.ErrNsNameNotProvided
	}

	opID, err := func() (string, error) {
		ctx, canc := context.WithTimeout(h.mainCtx, defaultTimeout)
		defer canc()

		out, err := h.Client.CreateHttpNamespace(ctx, &servicediscovery.CreateHttpNamespaceInput{
			Name: aws.String(ns.Name),
			Tags: fromMapToTagsSlice(ns.Metadata),
		})

		if err == nil {
			return aws.ToString(out.OperationId), nil
		}

		var oe *types.NamespaceAlreadyExists
		if errors.As(err, &oe) {
			return "", sr.ErrAlreadyExists
		}

		return "", err
	}()
	if err != nil {
		return nil, err
	}

	l := h.log.WithName("CreateNamespace")

	l.Info("waiting for operation to complete...")
	if err := h.pollOperationStatus(opID); err != nil {
		l.Info("operation completed with error")
		return nil, err
	}
	l.Info("operation completed successfully")

	return ns, nil
}

func (h *Handler) listOrGetNamespace(name *string) ([]*sr.Namespace, []*cloudMapIDs, error) {
	ctx, canc := context.WithTimeout(h.mainCtx, defaultTimeout)
	out, err := h.Client.ListNamespaces(ctx, &servicediscovery.ListNamespacesInput{})
	if err != nil {
		canc()
		return nil, nil, err
	}
	canc()

	list := []*sr.Namespace{}
	ids := []*cloudMapIDs{}
	for _, ns := range out.Namespaces {
		if name != nil && aws.ToString(ns.Name) != *name {
			// move on, this is not the namespace you're looking for.
			continue
		}
		arn := aws.ToString(ns.Arn)
		id := aws.ToString(ns.Id)

		nsItem := &sr.Namespace{
			Name:     aws.ToString(ns.Name),
			Metadata: map[string]string{},
		}

		nsItem.Metadata = func() map[string]string {
			tagCtx, tagCanc := context.WithTimeout(h.mainCtx, 30*time.Second)
			defer tagCanc()

			tags, err := h.Client.ListTagsForResource(tagCtx, &servicediscovery.ListTagsForResourceInput{
				ResourceARN: ns.Arn,
			})
			if err != nil {
				h.log.WithName("ListTagsForResource").Info("error while getting tags", "error", err, "name", nsItem.Name)
				return map[string]string{}
			}

			return fromTagsSliceToMap(tags.Tags)
		}()

		if name != nil {
			// if you're here, it means that you are indeed the namespace we
			// are looking for.
			return []*sr.Namespace{nsItem}, []*cloudMapIDs{{arn, id}}, nil
		}

		list = append(list, nsItem)
		ids = append(ids, &cloudMapIDs{arn, id})
	}

	return list, ids, nil
}

// ListNs returns a list of all namespaces.
func (h *Handler) ListNs() (nsList []*sr.Namespace, err error) {
	nsList, _, err = h.listOrGetNamespace(nil)
	return
}

// GetNs returns the namespace if exists.
func (h *Handler) GetNs(name string) (*sr.Namespace, error) {
	if name == "" {
		return nil, sr.ErrNsNameNotProvided
	}

	list, _, err := h.listOrGetNamespace(&name)
	if err != nil {
		return nil, err
	}

	if len(list) == 0 {
		return nil, sr.ErrNotFound
	}

	return list[0], nil
}

// GetNs returns the namespace if exists.
func (h *Handler) UpdateNs(ns *sr.Namespace) (*sr.Namespace, error) {
	if ns == nil {
		return nil, sr.ErrNsNotProvided
	}
	if ns.Name == "" {
		return nil, sr.ErrNsNameNotProvided
	}

	_, ids, err := h.listOrGetNamespace(&ns.Name)
	if err != nil {
		return nil, err
	}

	if len(ids) == 0 {
		return nil, sr.ErrNotFound
	}

	err = h.tagResource(ids[0].arn, fromMapToTagsSlice(ns.Metadata))
	if err == nil {
		return ns, nil
	}

	var oe *types.ResourceNotFoundException
	if errors.As(err, &oe) {
		return nil, sr.ErrNotFound
	}

	return nil, err
}

// GetNs returns the namespace if exists.
func (h *Handler) DeleteNs(name string) error {
	if name == "" {
		return sr.ErrNsNameNotProvided
	}

	_, ids, err := h.listOrGetNamespace(&name)
	if err != nil {
		return err
	}

	if len(ids) == 0 {
		return sr.ErrNotFound
	}

	ctx, canc := context.WithTimeout(h.mainCtx, time.Minute)
	defer canc()

	out, err := h.Client.DeleteNamespace(ctx, &servicediscovery.DeleteNamespaceInput{
		Id: aws.String(ids[0].id),
	})
	if err != nil {
		// This can only happen in the very corner case where someone is
		// extremely fast in deleting this after we got its id above.
		var oe *types.NamespaceNotFound
		if errors.As(err, &oe) {
			return sr.ErrNotFound
		}

		return err
	}

	l := h.log.WithName("DeleteNamespace")

	l.Info("waiting for operation to complete...")
	if err := h.pollOperationStatus(aws.ToString(out.OperationId)); err != nil {
		l.Info("operation completed with error")
		return err
	}
	l.Info("operation completed successfully")

	return nil
}
