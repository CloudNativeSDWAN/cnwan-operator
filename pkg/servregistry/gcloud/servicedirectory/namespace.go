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
	"strings"

	sr "github.com/CloudNativeSDWAN/cnwan-operator/pkg/servregistry"
	"google.golang.org/api/iterator"
	sdpb "google.golang.org/genproto/googleapis/cloud/servicedirectory/v1beta1"
	"google.golang.org/genproto/protobuf/field_mask"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetNs returns the namespace if exists.
func (s *servDir) GetNs(name string) (*sr.Namespace, error) {
	// -- Init
	if err := s.checkNames(&name, nil, nil); err != nil {
		return nil, err
	}
	l := s.log.WithName("GetNs").WithValues("ns-name", name)
	nsPath := s.getResourcePath(servDirPath{namespace: name})
	ctx, canc := context.WithTimeout(s.context, s.timeout)
	defer canc()

	sdNs, err := s.client.GetNamespace(ctx, &sdpb.GetNamespaceRequest{Name: nsPath})
	if err == nil {
		namespace := &sr.Namespace{
			Name:     name,
			Metadata: sdNs.Labels,
		}
		if namespace.Metadata == nil {
			namespace.Metadata = map[string]string{}
		}

		return namespace, nil
	}

	// What is the error?
	if err == context.DeadlineExceeded {
		l.Error(err, "timeout expired while waiting for service directory to reply", "timeout-seconds", s.timeout.Seconds())
		return nil, sr.ErrTimeOutExpired
	}

	if status.Code(err) == codes.NotFound {
		return nil, sr.ErrNotFound
	}

	// Any other error
	return nil, err
}

// ListNs returns a list of all namespaces.
func (s *servDir) ListNs() ([]*sr.Namespace, error) {
	// -- Init
	l := s.log.WithName("ListNs")
	ctx, canc := context.WithTimeout(s.context, s.timeout)
	defer canc()

	req := &sdpb.ListNamespacesRequest{
		Parent: s.getResourcePath(servDirPath{}),
	}

	nsList := []*sr.Namespace{}
	iter := s.client.ListNamespaces(ctx, req)
	if iter == nil {
		l.V(0).Info("returned list is nil")
		return nsList, nil
	}
	for {
		nextNs, iterErr := iter.Next()
		if iterErr != nil {

			if iterErr == context.DeadlineExceeded {
				l.Error(iterErr, "timeout expired while waiting for service directory to reply", "timeout-seconds", s.timeout.Seconds())
				return nil, sr.ErrTimeOutExpired
			}

			if iterErr != iterator.Done {
				l.Error(iterErr, "error while loading namespaces")
				return nil, iterErr
			}

			break
		}

		splitName := strings.Split(nextNs.Name, "/")
		ns := &sr.Namespace{
			Name:     splitName[len(splitName)-1],
			Metadata: nextNs.Labels,
		}
		if ns.Metadata == nil {
			ns.Metadata = map[string]string{}
		}

		nsList = append(nsList, ns)
	}

	return nsList, nil
}

// CreateNs creates the namespace.
func (s *servDir) CreateNs(ns *sr.Namespace) (*sr.Namespace, error) {
	// -- Init
	if ns == nil {
		return nil, sr.ErrNsNotProvided
	}
	if err := s.checkNames(&ns.Name, nil, nil); err != nil {
		return nil, err
	}
	l := s.log.WithName("CreateNs").WithValues("ns-name", ns.Name, "labels", ns.Metadata)
	ctx, canc := context.WithTimeout(s.context, s.timeout)
	defer canc()

	nsToCreate := &sdpb.Namespace{
		Name:   ns.Name,
		Labels: ns.Metadata,
	}

	req := &sdpb.CreateNamespaceRequest{
		Parent:      s.getResourcePath(servDirPath{}),
		NamespaceId: ns.Name,
		Namespace:   nsToCreate,
	}

	_, err := s.client.CreateNamespace(ctx, req)
	if err == nil {
		// No need to parse the returned resource, because it is the same
		// resource we want to add. So we can just returned the one we
		// want to add.
		return ns, nil
	}

	// What is the error?
	if err == context.DeadlineExceeded {
		l.Error(err, "timeout expired while waiting for service directory to reply", "timeout-seconds", s.timeout.Seconds())
		return nil, sr.ErrTimeOutExpired
	}

	if status.Code(err) == codes.AlreadyExists {
		return nil, sr.ErrAlreadyExists
	}

	// Any other error
	return nil, err
}

// UpdateNs updates the namespace.
func (s *servDir) UpdateNs(ns *sr.Namespace) (*sr.Namespace, error) {
	// -- Init
	if ns == nil {
		return nil, sr.ErrNsNotProvided
	}
	if err := s.checkNames(&ns.Name, nil, nil); err != nil {
		return nil, err
	}
	l := s.log.WithName("UpdateNs").WithValues("ns-name", ns.Name, "labels", ns.Metadata)
	ctx, canc := context.WithTimeout(s.context, s.timeout)
	defer canc()

	nsToUpd := &sdpb.Namespace{
		Name:   s.getResourcePath(servDirPath{namespace: ns.Name}),
		Labels: ns.Metadata,
	}

	req := &sdpb.UpdateNamespaceRequest{
		Namespace: nsToUpd,
		UpdateMask: &field_mask.FieldMask{
			Paths: []string{"metadata"},
		},
	}

	_, err := s.client.UpdateNamespace(ctx, req)
	if err == nil {
		return ns, nil
	}

	// What is the error?
	if err == context.DeadlineExceeded {
		l.Error(err, "timeout expired while waiting for service directory to reply", "timeout-seconds", s.timeout.Seconds())
		return nil, sr.ErrTimeOutExpired
	}

	if status.Code(err) == codes.NotFound {
		return nil, sr.ErrNotFound
	}

	// Any other error
	return nil, err
}

// DeleteNs deletes the namespace.
func (s *servDir) DeleteNs(name string) error {
	// -- Init
	if err := s.checkNames(&name, nil, nil); err != nil {
		return err
	}
	l := s.log.WithName("DeleteNs").WithValues("ns-name", name)
	ctx, canc := context.WithTimeout(s.context, s.timeout)
	defer canc()

	req := &sdpb.DeleteNamespaceRequest{
		Name: s.getResourcePath(servDirPath{namespace: name}),
	}

	err := s.client.DeleteNamespace(ctx, req)
	if err == nil {
		return nil
	}

	// What is the error?
	if err == context.DeadlineExceeded {
		l.Error(err, "timeout expired while waiting for service directory to reply", "timeout-seconds", s.timeout.Seconds())
		return sr.ErrTimeOutExpired
	}

	if status.Code(err) == codes.NotFound {
		return sr.ErrNotFound
	}

	// Any other error
	return err
}
