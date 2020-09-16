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

import "errors"

var (
	// ErrServRegNotProvided is returned when the broker has no service
	// registry and, thus, cannot make any changes
	ErrServRegNotProvided error = errors.New("no service registry is provided")
	// ErrNotFound is returned when the resource does not exist
	ErrNotFound error = errors.New("resource not found")
	// ErrAlreadyExists is returned when the resource already exists
	ErrAlreadyExists error = errors.New("resource already exists")
	// ErrServNotProvided is returned when the service is missing, i.e. is nil
	ErrServNotProvided error = errors.New("service is empty")
	// ErrServNoMetadata is returned when the provided service has no metadata
	ErrServNoMetadata error = errors.New("service has no metadata")
	// ErrNsNotProvided is returned when the namespace is missing, i.e. is nil
	ErrNsNotProvided error = errors.New("namespace is empty")
	// ErrServNameNotProvided is returned when the service name is empty
	ErrServNameNotProvided error = errors.New("service name is empty")
	// ErrNsNameNotProvided is returned when the namespace name is empty
	ErrNsNameNotProvided error = errors.New("namespace name is empty")
	// ErrNoEndpoints is returned when a service has no endpoints
	ErrNoEndpoints error = errors.New("service has no endpoints")
	// ErrNsNotEmpty is returned when trying to delete a namespace that is not
	// empty
	ErrNsNotEmpty error = errors.New("namespace is not empty")
	// ErrServNotEmpty is returned when trying to delete a service that is not
	// empty
	ErrServNotEmpty error = errors.New("service is not empty")
	// ErrNsNotOwnedServs is returned when trying to delete a namespace that
	// has services that are not owned by the operator
	ErrNsNotOwnedServs error = errors.New("namespace contains services not owned by the operator")
	// ErrServNotOwnedEndps is returned when trying to delete a service that
	// has endpoints that are not owned by the operator
	ErrServNotOwnedEndps error = errors.New("service contains endpoints not owned by the operator")
	// ErrNsNotOwnedByOp is returned when a namespace is not owned by the
	// cnwan operator and therefore the action cannot be performed
	ErrNsNotOwnedByOp error = errors.New("namespace is not owned by the cnwan operator")
	// ErrServNotOwnedByOp is returned when a service is not owned by the
	// cnwan operator and therefore the action cannot be performed
	ErrServNotOwnedByOp error = errors.New("service is not owned by the cnwan operator")
)
