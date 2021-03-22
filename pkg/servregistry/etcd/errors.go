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

import "errors"

// These errors are thrown by the package when an incorrect value is provided
// to some of its functions, or when something unexpected happens.
var (
	// ErrNilClient is returned when the etcd client provided to
	// NewServiceRegistryWithEtcd is nil
	ErrNilClient error = errors.New("no etcd client provided")
	// ErrNilObject is returned when a function is provided with a nil
	// object.
	ErrNilObject error = errors.New("no object provided")
	// ErrUnknownObject is returned when the KeyBuilder is provided with an
	// object that is not a namespace, service or endpoint.
	ErrUnknownObject error = errors.New("object is unknown")
)
