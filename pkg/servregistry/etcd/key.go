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

package etcd

import (
	"path"
	"strings"

	sr "github.com/CloudNativeSDWAN/cnwan-operator/pkg/servregistry"
)

// objectPrefix is a string the precedes the actual name of the object
type objectPrefix string

const (
	// namespacePrefix is the string the precedes the name of the namespace
	namespacePrefix objectPrefix = "namespaces"
	// servicePrefix is the string the precedes the name of the service
	servicePrefix objectPrefix = "services"
	// endpointPrefix is the string the precedes the name of the endpoint
	endpointPrefix objectPrefix = "endpoints"
)

// ObjectType identifies the type of the object we are dealing with and will
// build the key according to it, i.e. namespace, service or endpoint.
type ObjectType int

// These constants define the object type that the key builder will deal with.
const (
	// UnknownOrInvalidObject is an object that is neither a namespace, nor a service,
	// nor an endpoint and is thus not related to service registry.
	UnknownOrInvalidObject ObjectType = 0
	// NamespaceObject represents a namespace.
	NamespaceObject ObjectType = iota
	// ServiceObject represents a service.
	ServiceObject ObjectType = iota
	// EndpointObject represents an endpoint.
	EndpointObject ObjectType = iota
)

// KeyBuilder manages and builds an etcd key for service registry.
// It can create the appropriate path key based on the object type it is
// dealing with or make assumptions on what the value is based on
// its key path so that you know how to unmarshal its value.
//
// Be aware that KeyBuilder will NOT include a prefix when it returns the
// key as a string, so you should either include it manually or use the
// namespace package
// (https://pkg.go.dev/go.etcd.io/etcd@v3.3.25+incompatible/clientv3/namespace).
//
// Take a look at the examples to learn more about this.
//
//
// NOTE: as written above, Key only makes **assumptions**: you need to
// check that the unmarshal operation was successful to make sure the object
// is correct. This is performed automatically by the Service Registry
// implementer, but you have to do it on your own in case you use it with
// a crude client.
type KeyBuilder struct {
	nsName   string
	servName string
	endpName string
}

// KeyFromNames starts building a key based on the provided names.
// This method is useful in case you want build a string and already
// know the name and parents' names of the object that will be stored as
// value.
func KeyFromNames(names ...string) *KeyBuilder {
	e := &KeyBuilder{}

	size := len(names)
	if size == 0 || size > 3 {
		return e
	}

	// namespace must always be there anyway
	e.nsName = names[0]

	if size >= 2 {
		e.servName = names[1]

		if size == 3 {
			e.endpName = names[2]
		}
	}

	return e
}

// KeyFromServiceRegistryObject returns a KeyBuilder starting from a service
// registry object defined in
// https://pkg.go.dev/github.com/CloudNativeSDWAN/cnwan-operator/pkg/servregistry:
// for example a namespace, service or endpoint.
//
// In case the a key couldn't be built this method either returns an error
// belonging to package mentioned above or ErrNilObject if the object is nil.
func KeyFromServiceRegistryObject(object interface{}) (*KeyBuilder, error) {
	if object == nil {
		return nil, ErrNilObject
	}

	switch obj := object.(type) {
	case *sr.Namespace:
		if len(obj.Name) == 0 {
			return nil, sr.ErrNsNameNotProvided
		}
		return KeyFromNames(obj.Name), nil

	case *sr.Service:
		if len(obj.NsName) == 0 {
			return nil, sr.ErrNsNameNotProvided
		}
		if len(obj.Name) == 0 {
			return nil, sr.ErrServNameNotProvided
		}
		return KeyFromNames(obj.NsName, obj.Name), nil

	case *sr.Endpoint:
		if len(obj.NsName) == 0 {
			return nil, sr.ErrNsNameNotProvided
		}
		if len(obj.ServName) == 0 {
			return nil, sr.ErrServNameNotProvided
		}
		if len(obj.Name) == 0 {
			return nil, sr.ErrEndpNameNotProvided
		}
		return KeyFromNames(obj.NsName, obj.ServName, obj.Name), nil

	default:
		return nil, ErrUnknownObject
	}
}

// KeyFromString returns a KeyBuilder starting from a string, i.e.
// 	/namespaces/namespace-name/services/service-name`
// or
//	/something/something-name/another/another-name
//
// This is very useful in case you want to check if the key is valid for
// the service registry.
//
// Note that this WILL also strip any prefix from the key, so if you really
// need it you should either write it manually or use the
// namespace package
// (https://pkg.go.dev/go.etcd.io/etcd@v3.3.25+incompatible/clientv3/namespace)
// from etcd, which includes/excludes it automatically for each key.
//
// Take a look at the examples to learn more.
func KeyFromString(key string) *KeyBuilder {
	_key := strings.Trim(key, "/")
	names := strings.Split(_key, "/")
	nsIndex := -1

	for i, name := range names {
		if name == string(namespacePrefix) {
			nsIndex = i
			break
		}
	}

	if nsIndex == -1 {
		return KeyFromNames()
	}

	names = names[nsIndex:]
	length := len(names)
	if length%2 != 0 || length > 6 {
		// We cannot have an odd number of names and the number cannot be more
		// than 6.
		// It should be one of the following:
		// * ["namespaces", "name"]
		// * ["namespaces", "name", "services", "name"]
		// * ["namespaces", "name", "services", "name", "endpoints", "name"]
		return KeyFromNames()
	}

	if length == 2 {
		// First example above
		return KeyFromNames(names[1])
	}

	if length == 4 {
		// Second example above
		if names[2] != string(servicePrefix) {
			return KeyFromNames()
		}

		return KeyFromNames(names[1], names[3])
	}

	if length == 6 && names[4] != string(endpointPrefix) {
		return KeyFromNames()
	}

	// Third example above
	return KeyFromNames(names[1], names[3], names[5])
}

// Clone returns another pointer to a KeyBuilder with the same settings
// as the one you're cloning from.
//
// Since golang doesn't have a DeepCopy method, use this in case you want
// to generate other keys leaving this one intact.
func (k *KeyBuilder) Clone() *KeyBuilder {
	return &KeyBuilder{
		nsName:   k.nsName,
		servName: k.servName,
		endpName: k.endpName,
	}
}

// IsValid returns true if the key is a valid key for service registry and is
// the equivalent of doing:
// 	k.ObjectType() != UnknownOrInvalidObject
func (k *KeyBuilder) IsValid() bool {
	return k.ObjectType() != UnknownOrInvalidObject
}

// ObjectType returns the assumed type of the object stored as value.
func (k *KeyBuilder) ObjectType() ObjectType {
	if k.nsName == "" {
		// If no namespace is there, then it is always invalid
		return UnknownOrInvalidObject
	}

	if k.servName == "" && k.endpName == "" {
		return NamespaceObject
	}

	if k.servName != "" {
		if k.endpName != "" {
			return EndpointObject
		}

		return ServiceObject
	}

	return UnknownOrInvalidObject
}

// SetNamespace sets the namespace name.
func (k *KeyBuilder) SetNamespace(name string) *KeyBuilder {
	if len(name) > 0 {
		k.nsName = name
	}

	return k
}

// SetService sets the service name.
func (k *KeyBuilder) SetService(name string) *KeyBuilder {
	if len(name) > 0 {
		k.servName = name
	}

	return k
}

// SetEndpoint sets the endpoint name.
func (k *KeyBuilder) SetEndpoint(name string) *KeyBuilder {
	if len(name) > 0 {
		k.endpName = name
	}

	return k
}

// GetNamespace returns the name of the namespace, if set.
func (k *KeyBuilder) GetNamespace() (name string) {
	if k.IsValid() {
		name = k.nsName
	}

	return name
}

// GetService returns the name of the service, if set.
func (k *KeyBuilder) GetService() (name string) {
	if k.IsValid() {
		name = k.servName
	}

	return name
}

// GetEndpoint returns the name of the endpoint, if set.
func (k *KeyBuilder) GetEndpoint() (name string) {
	if k.IsValid() {
		name = k.endpName
	}

	return name
}

// String "marshals" the key into a string.
//
// This will not print any prefix and the key will never start with a
// `/`.
//
// In case you need that, you will have to put that manually.
//
// Note: this method will return an empty string if the key is not valid,
// i.e. when no namespace is set or the key is not suitable for service
// registry usage.
//
// Make sure to call IsValid() before marshaling to string.
func (k *KeyBuilder) String() string {
	if !k.IsValid() {
		return ""
	}

	key := []string{}
	key = append(key, string(namespacePrefix), k.nsName)

	if k.ObjectType() > NamespaceObject {
		key = append(key, string(servicePrefix), k.servName)

		if k.ObjectType() == EndpointObject {
			key = append(key, string(endpointPrefix), k.endpName)
		}
	}

	return path.Join(key...)
}
