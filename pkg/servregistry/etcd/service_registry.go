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

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path"
	"time"

	sr "github.com/CloudNativeSDWAN/cnwan-operator/pkg/servregistry"
	clientv3 "go.etcd.io/etcd/clientv3"
	namespace "go.etcd.io/etcd/clientv3/namespace"
	"go.etcd.io/etcd/etcdserver/api/v3rpc/rpctypes"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
)

const (
	defaultPrefix string = "/service-registry/"
	// timeout used when sending requests
	// TODO: make this configurable or include context explicitly on each
	// method (best way)
	defaultTimeout time.Duration = time.Duration(15) * time.Second
)

// EtcdServReg is a wrap around an etcd client that allows you to perform
// service registry operations on etcd, such as storing, updating, deleting
// or retrieving a namespace, service, or endpoint.
// It is an implementation of ServiceRegistry defined in
// https://github.com/CloudNativeSDWAN/cnwan-operator/pkg/servregistry.
type EtcdServReg struct {
	cli     *clientv3.Client
	kv      clientv3.KV
	prefix  string
	mainCtx context.Context
}

// NewServiceRegistryWithEtcd returns an instance of ServiceRegistry as defined
// by  with
// ETCD as a backend.
// https://pkg.go.dev/github.com/CloudNativeSDWAN/cnwan-operator/pkg/servregistry#ServiceRegistry.
//
// If prefix is not nil, all data will be prefixed with the value you set on
// prefix, for example:
//	/my-prefix/my-data.
// If you don't want any prefix, set the value of prefix to an empty string or
// just "/" and all keys will be prefixed by just "/".
//
// You may even specify a prefix with multiple slashes: for example, if you
// have multiple clusters/environments, a key could be:
// 	"cluster-1/service-registry".
// Be aware that any leading AND trailing slashes will be removed to prevent
// key paths errors, but will be inserted correctly automatically when calling
// any of its methods.
//
// Be careful with this value as it can potentially overwrite existing data.
//
// If context is not nil, it will be used as the main context upon which all
// queries to etcd will be based on.
//
// This method returns an error only if the client provided to it is nil.
func NewServiceRegistryWithEtcd(ctx context.Context, cli *clientv3.Client, prefix *string) *EtcdServReg {
	// Use the default prefix (/service-registry),
	// unless the prefix is not nil, in which case we use that one.
	pref := parsePrefix(prefix)

	return &EtcdServReg{
		cli:     cli,
		kv:      namespace.NewKV(cli.KV, pref),
		prefix:  pref,
		mainCtx: ctx,
	}
}

func (e *EtcdServReg) ExtractData(ns *corev1.Namespace, serv *corev1.Service) (*sr.Namespace, *sr.Service, []*sr.Endpoint, error) {
	// NOTE: on future versions, this function will be removed from service
	// registry and moved to the broker instead: it's not this package's job
	// to convert structs.
	if ns == nil {
		return nil, nil, nil, sr.ErrNsNotProvided
	}
	if serv == nil {
		return nil, nil, nil, sr.ErrServNotProvided
	}

	// Parse the namespace
	namespaceData := &sr.Namespace{
		Name:     ns.Name,
		Metadata: ns.Annotations,
	}
	if namespaceData.Metadata == nil {
		namespaceData.Metadata = map[string]string{}
	}

	// Parse the service
	// NOTE: we put metadata on the service in service directory,
	// not on the endpoints
	serviceData := &sr.Service{
		Name:     serv.Name,
		NsName:   ns.Name,
		Metadata: serv.Annotations,
	}
	if serviceData.Metadata == nil {
		serviceData.Metadata = map[string]string{}
	}

	// Get the endpoints from the service
	// First, build the ips
	ips := []string{}
	// TODO: check if Spec is nil
	// TODO: check if ExternalIPS is not nil
	// ^ this will be performed on the new version of ExternalData
	ips = append(ips, serv.Spec.ExternalIPs...)

	// Get data from load balancers
	for _, ing := range serv.Status.LoadBalancer.Ingress {
		ips = append(ips, ing.IP)
	}

	endpointsData := []*sr.Endpoint{}
	for _, port := range serv.Spec.Ports {
		for _, ip := range ips {

			// Create an hashed name for this
			toBeHashed := fmt.Sprintf("%s-%d", ip, port.Port)
			h := sha256.New()
			h.Write([]byte(toBeHashed))
			hash := hex.EncodeToString(h.Sum(nil))

			// Only take the first 10 characters of the hashed name
			name := fmt.Sprintf("%s-%s", serv.Name, hash[:10])
			endpointsData = append(endpointsData, &sr.Endpoint{
				Name:     name,
				NsName:   namespaceData.Name,
				ServName: serviceData.Name,
				Address:  ip,
				Port:     port.Port,
				Metadata: map[string]string{},
			})
		}
	}

	return namespaceData, serviceData, endpointsData, nil
}

func (e *EtcdServReg) getOne(ctx context.Context, key *KeyBuilder) (interface{}, error) {
	// This function is not exported and thus is only for internal purpose
	// only: any checks and validations are performed by the caller
	// and not here.
	resp, err := e.kv.Get(ctx, key.String(), clientv3.WithLimit(1))
	if err != nil {
		if err == rpctypes.ErrGRPCKeyNotFound {
			return nil, sr.ErrNotFound
		}
		return nil, err
	}

	if len(resp.Kvs) == 0 {
		return nil, sr.ErrNotFound
	}

	switch key.ObjectType() {
	case NamespaceObject:
		var ns sr.Namespace
		if err := yaml.Unmarshal(resp.Kvs[0].Value, &ns); err != nil {
			return nil, err
		}
		return &ns, err
	case ServiceObject:
		var serv sr.Service
		if err := yaml.Unmarshal(resp.Kvs[0].Value, &serv); err != nil {
			return nil, err
		}
		return &serv, err
	case EndpointObject:
		var endp sr.Endpoint
		if err := yaml.Unmarshal(resp.Kvs[0].Value, &endp); err != nil {
			return nil, err
		}
		return &endp, err
	default:
		return nil, ErrUnknownObject
	}
}

func (e *EtcdServReg) getList(ctx context.Context, key *KeyBuilder, each func([]byte)) error {
	var objectsToFind ObjectType
	var suffix string
	if key == nil {
		key = &KeyBuilder{}
	}

	switch key.ObjectType() {
	case NamespaceObject:
		objectsToFind = ServiceObject
		suffix = string(servicePrefix)
	case ServiceObject:
		objectsToFind = EndpointObject
		suffix = string(endpointPrefix)
	default:
		objectsToFind = NamespaceObject
		suffix = string(namespacePrefix)
	}

	actualKey := path.Join(key.String(), suffix)
	resp, err := e.kv.Get(ctx, actualKey, clientv3.WithPrefix())
	if err != nil {
		if err == rpctypes.ErrGRPCKeyNotFound {
			return sr.ErrNotFound
		}
		return err
	}

	for _, currentKV := range resp.Kvs {
		currentKey := string(currentKV.Key)
		if KeyFromString(currentKey).ObjectType() != objectsToFind {
			continue
		}

		if each != nil {
			each(currentKV.Value)
		}
	}

	return nil
}

func (e *EtcdServReg) put(ctx context.Context, object interface{}, update bool) error {
	if object == nil {
		return ErrNilObject
	}

	key, err := KeyFromServiceRegistryObject(object)
	if err != nil {
		return err
	}

	// As per documentation, "Conflicting names result in a runtime error."
	// We handle service registry objects, which do not suffer from this.
	// Therefore, there is no need to check the error here.
	bytes, _ := yaml.Marshal(object)

	// revision == 0 means does not exist
	cmp := "="
	if update {
		// revision > 0 means that it does exist
		cmp = ">"
	}

	conditions := []clientv3.Cmp{}
	elses := []clientv3.Op{}

	if key.ObjectType() >= ServiceObject {
		nsKey := KeyFromNames(key.GetNamespace())
		conditions = append(conditions, clientv3.Compare(clientv3.CreateRevision(nsKey.String()), ">", 0))
		elses = append(elses, clientv3.OpGet(nsKey.String(), clientv3.WithCountOnly()))

		if key.ObjectType() == EndpointObject {
			servKey := KeyFromNames(key.GetNamespace(), key.GetService())
			conditions = append(conditions, clientv3.Compare(clientv3.CreateRevision(servKey.String()), ">", 0))
			elses = append(elses, clientv3.OpGet(servKey.String(), clientv3.WithCountOnly()))
		}
	}

	conditions = append(conditions, clientv3.Compare(clientv3.CreateRevision(key.String()), cmp, 0))
	createIt := clientv3.OpPut(key.String(), string(bytes))

	resp, err := e.kv.Txn(ctx).If(conditions...).Then(createIt).Else(elses...).Commit()
	if err != nil {
		return err
	}

	if resp.Succeeded {
		// All ok
		return nil
	}

	if len(resp.Responses) > 0 {
		if resp.Responses[0].GetResponseRange().Count == 0 {
			return fmt.Errorf("namespace with name %s does not exist", key.GetNamespace())
		}

		if len(resp.Responses) == 2 && resp.Responses[1].GetResponseRange().Count == 0 {
			return fmt.Errorf("service with name %s does not exist", key.GetService())
		}
	}

	if !update {
		return sr.ErrAlreadyExists
	}

	return sr.ErrNotFound
}

func (e *EtcdServReg) delete(ctx context.Context, key *KeyBuilder) error {
	condition := clientv3.Compare(clientv3.CreateRevision(key.String()), ">", 0)

	// We need to remove all children elements.
	// The user must check if they care about them manually. (i.e. the broker
	// will have to check this).
	deleteIt := clientv3.OpDelete(key.String(), clientv3.WithPrefix())

	resp, err := e.kv.Txn(ctx).If(condition).Then(deleteIt).Commit()
	if err != nil {
		return err
	}

	if resp.Succeeded {
		// All ok
		return nil
	}

	return sr.ErrNotFound
}
