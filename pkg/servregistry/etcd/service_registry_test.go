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
	"testing"

	sr "github.com/CloudNativeSDWAN/cnwan-operator/pkg/servregistry"
	"github.com/stretchr/testify/assert"
	"go.etcd.io/etcd/api/v3/mvccpb"
	"go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
	clientv3 "go.etcd.io/etcd/client/v3"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewServiceRegistryWithEtcd(t *testing.T) {
	a := assert.New(t)

	prefix := "something"
	res := NewServiceRegistryWithEtcd(context.Background(), &clientv3.Client{}, &prefix)
	a.NotNil(res)
}

func TestGetOne(t *testing.T) {
	a := assert.New(t)
	unknErr := fmt.Errorf("unknown")
	e := &EtcdServReg{}
	ns := &sr.Namespace{
		Name: "namespace-name",
		Metadata: map[string]string{
			"env": "beta",
		},
	}
	nsBytes, _ := yaml.Marshal(ns)
	serv := &sr.Service{
		NsName: ns.Name,
		Name:   "service-name",
		Metadata: map[string]string{
			"version": "v0.2.1",
		},
	}
	servBytes, _ := yaml.Marshal(serv)
	endp := &sr.Endpoint{
		NsName:   ns.Name,
		ServName: serv.Name,
		Name:     "endpoint-name",
		Metadata: map[string]string{
			"protocol": "tcp",
		},
	}
	endpBytes, _ := yaml.Marshal(endp)

	cases := []struct {
		get    func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error)
		key    *KeyBuilder
		expObj interface{}
		expErr error
	}{
		{
			key: KeyFromNames(ns.Name),
			get: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
				return nil, rpctypes.ErrGRPCKeyNotFound
			},
			expErr: sr.ErrNotFound,
		},
		{
			key: KeyFromNames(ns.Name),
			get: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
				return nil, fmt.Errorf("any error")
			},
			expErr: unknErr,
		},
		{
			key: KeyFromNames(ns.Name),
			get: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
				return &clientv3.GetResponse{
					Kvs: []*mvccpb.KeyValue{},
				}, nil
			},
			expErr: sr.ErrNotFound,
		},
		{
			key: KeyFromNames(ns.Name),
			get: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
				return &clientv3.GetResponse{
					Kvs: []*mvccpb.KeyValue{
						{
							Value: []byte("invalid"),
						},
					},
				}, nil
			},
			expErr: unknErr,
		},
		{
			key: KeyFromNames(serv.NsName, serv.Name),
			get: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
				return &clientv3.GetResponse{
					Kvs: []*mvccpb.KeyValue{
						{
							Value: []byte("invalid"),
						},
					},
				}, nil
			},
			expErr: unknErr,
		},
		{
			key: KeyFromNames(endp.NsName, endp.ServName, endp.Name),
			get: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
				return &clientv3.GetResponse{
					Kvs: []*mvccpb.KeyValue{
						{
							Value: []byte("invalid"),
						},
					},
				}, nil
			},
			expErr: unknErr,
		},
		{
			key: KeyFromNames(),
			get: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
				return &clientv3.GetResponse{
					Kvs: []*mvccpb.KeyValue{
						{
							Value: []byte("invalid"),
						},
					},
				}, nil
			},
			expErr: ErrUnknownObject,
		},
		{
			key: KeyFromNames(ns.Name),
			get: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
				return &clientv3.GetResponse{
					Kvs: []*mvccpb.KeyValue{
						{
							Value: nsBytes,
						},
					},
				}, nil
			},
			expObj: ns,
		},
		{
			key: KeyFromNames(serv.NsName, serv.Name),
			get: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
				return &clientv3.GetResponse{
					Kvs: []*mvccpb.KeyValue{
						{
							Value: servBytes,
						},
					},
				}, nil
			},
			expObj: serv,
		},
		{
			key: KeyFromNames(endp.NsName, endp.ServName, endp.Name),
			get: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
				return &clientv3.GetResponse{
					Kvs: []*mvccpb.KeyValue{
						{
							Value: endpBytes,
						},
					},
				}, nil
			},
			expObj: endp,
		},
	}

	for i, currCase := range cases {
		f := &fakeKV{
			_get: currCase.get,
		}
		e.kv = f

		var errErr bool
		res, err := e.getOne(context.Background(), currCase.key)

		errRes := a.Equal(currCase.expObj, res)
		if currCase.expErr == unknErr {
			errErr = a.Error(err)
		} else {
			errErr = a.Equal(currCase.expErr, err)
		}

		if !errRes || !errErr {
			a.FailNow(fmt.Sprintf("case %d failed", i))
		}
	}
}

func TestGetList(t *testing.T) {
	a := assert.New(t)
	unknErr := fmt.Errorf("unknown")
	e := &EtcdServReg{mainCtx: context.Background()}
	nsSearchPref := string(namespacePrefix)
	ns := &sr.Namespace{
		Name: "namespace-name",
		Metadata: map[string]string{
			"env": "beta",
		},
	}
	nsBytes, _ := yaml.Marshal(ns)
	servSearchPref := path.Join(string(namespacePrefix), ns.Name, string(servicePrefix))
	serv := &sr.Service{
		NsName: ns.Name,
		Name:   "service-name",
		Metadata: map[string]string{
			"version": "v0.2.1",
		},
	}
	endpSearchPref := path.Join(string(namespacePrefix), ns.Name, string(servicePrefix), serv.Name, string(endpointPrefix))
	servBytes, _ := yaml.Marshal(serv)
	endp := &sr.Endpoint{
		NsName:   ns.Name,
		ServName: serv.Name,
		Name:     "endpoint-name",
		Metadata: map[string]string{
			"protocol": "tcp",
		},
	}
	endpBytes, _ := yaml.Marshal(endp)

	cases := []struct {
		id     string
		get    func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error)
		key    *KeyBuilder
		each   func([]byte)
		expErr error
	}{
		{
			id: "nil-key-check-search-prefix",
			get: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
				if !a.Equal(nsSearchPref, key) {
					a.FailNow("case 0: key failed")
				}
				return nil, fmt.Errorf("just to stop the execution")
			},
			expErr: unknErr,
		},
		{
			id:  "not-nil-key-check-search-prefix",
			key: &KeyBuilder{},
			get: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
				if !a.Equal(nsSearchPref, key) {
					a.FailNow("case 1: key failed")
				}
				return nil, fmt.Errorf("just to stop the execution")
			},
			expErr: unknErr,
		},
		{
			id:  "ns-key-check-search-prefix",
			key: KeyFromNames(ns.Name),
			get: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
				if !a.Equal(servSearchPref, key) {
					a.FailNow("case 2: key failed")
				}
				return nil, fmt.Errorf("just to stop the execution")
			},
			expErr: unknErr,
		},
		{
			id:  "serv-key-check-search-prefix",
			key: KeyFromNames(ns.Name, serv.Name),
			get: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
				if !a.Equal(endpSearchPref, key) {
					a.FailNow("case 3: key failed")
				}
				return nil, fmt.Errorf("just to stop the execution")
			},
			expErr: unknErr,
		},
		{
			id:  "should-return-ErrGRPCKeyNotFound",
			key: KeyFromNames(ns.Name),
			get: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
				return nil, rpctypes.ErrGRPCKeyNotFound
			},
			expErr: sr.ErrNotFound,
		},
		{
			id:  "should-return-empty-kvs",
			key: KeyFromNames(ns.Name),
			get: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
				return &clientv3.GetResponse{
					Kvs: []*mvccpb.KeyValue{},
				}, nil
			},
		},
		{
			id: "should-call-each-func-on-ns",
			get: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
				return &clientv3.GetResponse{
					Kvs: []*mvccpb.KeyValue{
						{Key: []byte(KeyFromNames(ns.Name, serv.Name, endp.Name).String()), Value: endpBytes},
						{Key: []byte(KeyFromNames(ns.Name, serv.Name).String()), Value: servBytes},
						{Key: []byte(KeyFromNames(ns.Name).String()), Value: nsBytes},
					},
				}, nil
			},
			each: func(b []byte) {
				if !a.Equal(nsBytes, b) {
					a.FailNow("case 6: object provided is not correct")
				}
			},
		},
		{
			id:  "should-call-each-func-on-ns",
			key: &KeyBuilder{},
			get: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
				return &clientv3.GetResponse{
					Kvs: []*mvccpb.KeyValue{
						{Key: []byte(KeyFromNames(ns.Name, serv.Name, endp.Name).String()), Value: endpBytes},
						{Key: []byte(KeyFromNames(ns.Name, serv.Name).String()), Value: servBytes},
						{Key: []byte(KeyFromNames(ns.Name).String()), Value: nsBytes},
					},
				}, nil
			},
			each: func(b []byte) {
				if !a.Equal(nsBytes, b) {
					a.FailNow("case 7: object provided is not correct")
				}
			},
		},
		{
			id:  "should-call-each-func-on-serv",
			key: KeyFromNames(ns.Name),
			get: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
				return &clientv3.GetResponse{
					Kvs: []*mvccpb.KeyValue{
						{Key: []byte(KeyFromNames(ns.Name, serv.Name, endp.Name).String()), Value: endpBytes},
						{Key: []byte(KeyFromNames(ns.Name).String()), Value: nsBytes},
						{Key: []byte(KeyFromNames(ns.Name, serv.Name).String()), Value: servBytes},
					},
				}, nil
			},
			each: func(b []byte) {
				if !a.Equal(servBytes, b) {
					a.FailNow("case 8: object provided is not correct")
				}
			},
		},
		{
			id:  "should-call-each-func-on-endp",
			key: KeyFromNames(ns.Name, serv.Name),
			get: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
				return &clientv3.GetResponse{
					Kvs: []*mvccpb.KeyValue{
						{Key: []byte(KeyFromNames(ns.Name, serv.Name).String()), Value: servBytes},
						{Key: []byte(KeyFromNames(ns.Name).String()), Value: nsBytes},
						{Key: []byte(KeyFromNames(ns.Name, serv.Name, endp.Name).String()), Value: endpBytes},
					},
				}, nil
			},
			each: func(b []byte) {
				if !a.Equal(endpBytes, b) {
					a.FailNow("case 9: object provided is not correct")
				}
			},
		},
	}

	for i, currCase := range cases {
		e.kv = &fakeKV{
			_get: currCase.get,
		}

		var errErr bool
		err := e.getList(e.mainCtx, currCase.key, currCase.each)
		if currCase.expErr == unknErr {
			errErr = a.Error(err)
		} else {
			errErr = a.Equal(currCase.expErr, err)
		}

		if !errErr {
			a.FailNow(fmt.Sprintf("case %d failed", i))
		}
	}
}
func TestPut(t *testing.T) {
	a := assert.New(t)
	e := &EtcdServReg{}
	txn := &fakeTXN{}
	txn._if = func(cs ...clientv3.Cmp) clientv3.Txn {
		return txn
	}
	txn._then = func(ops ...clientv3.Op) clientv3.Txn {
		return txn
	}
	txn._else = func(ops ...clientv3.Op) clientv3.Txn {
		return txn
	}
	unknErr := fmt.Errorf("unknown")

	cases := []struct {
		obj    interface{}
		commit func() (*clientv3.TxnResponse, error)
		upd    bool
		expErr error
	}{
		{
			expErr: ErrNilObject,
		},
		{
			obj:    &sr.Namespace{},
			expErr: sr.ErrNsNameNotProvided,
		},
		{
			obj:    &sr.Service{NsName: "namespace-name"},
			expErr: sr.ErrServNameNotProvided,
		},
		{
			obj:    &sr.Endpoint{NsName: "namespace-name", ServName: "service-name"},
			expErr: sr.ErrEndpNameNotProvided,
		},
		{
			obj:    &sr.Endpoint{NsName: "namespace-name", ServName: "service-name"},
			expErr: sr.ErrEndpNameNotProvided,
		},
	}

	for i, currCase := range cases {
		f := &fakeKV{}
		f._txn = func(ctx context.Context) clientv3.Txn {
			return txn
		}
		txn._commit = currCase.commit
		e.kv = f

		var errErr bool
		err := e.put(context.Background(), currCase.obj, currCase.upd)
		if currCase.expErr == unknErr {
			errErr = a.Error(err)
		} else {
			errErr = a.Equal(currCase.expErr, err)
		}

		if !errErr {
			a.FailNow(fmt.Sprintf("case %d failed", i))
		}
	}
}

func TestDelete(t *testing.T) {
	a := assert.New(t)
	e := &EtcdServReg{}
	txn := &fakeTXN{}
	txn._if = func(cs ...clientv3.Cmp) clientv3.Txn {
		return txn
	}
	txn._then = func(ops ...clientv3.Op) clientv3.Txn {
		return txn
	}
	unknErr := fmt.Errorf("unknown")

	cases := []struct {
		id     string
		key    *KeyBuilder
		commit func() (*clientv3.TxnResponse, error)
		expErr error
	}{
		{
			key: KeyFromNames("anything"),
			id:  "returns-an-error",
			commit: func() (*clientv3.TxnResponse, error) {
				return nil, fmt.Errorf("any error")
			},
			expErr: unknErr,
		},
		{
			key: KeyFromNames("anything"),
			id:  "is-not-successful",
			commit: func() (*clientv3.TxnResponse, error) {
				return &clientv3.TxnResponse{
					Succeeded: false,
				}, nil
			},
			expErr: sr.ErrNotFound,
		},
		{
			key: KeyFromNames("anything"),
			id:  "is-successful",
			commit: func() (*clientv3.TxnResponse, error) {
				return &clientv3.TxnResponse{
					Succeeded: true,
				}, nil
			},
		},
	}

	for i, currCase := range cases {
		f := &fakeKV{}
		f._txn = func(ctx context.Context) clientv3.Txn {
			return txn
		}
		txn._commit = currCase.commit
		e.kv = f

		var errErr bool
		err := e.delete(context.Background(), currCase.key)
		if currCase.expErr == unknErr {
			errErr = a.Error(err)
		} else {
			errErr = a.Equal(currCase.expErr, err)
		}

		if !errErr {
			a.FailNow(fmt.Sprintf("case %d failed", i))
		}
	}
}

func TestExtractData(t *testing.T) {
	a := assert.New(t)
	nsName, servName := "ns", "serv"
	e := &EtcdServReg{}
	nsToTest := &corev1.Namespace{
		ObjectMeta: v1.ObjectMeta{
			Name: nsName,
		},
	}
	nsAnnotations := map[string]string{
		"key": "val",
	}
	ips := []string{"10.10.10.10", "11.11.11.11"}
	ports := []int32{3333, 4444}
	servToTest := &corev1.Service{
		ObjectMeta: v1.ObjectMeta{
			Name:      servName,
			Namespace: nsName,
		},
		Spec: corev1.ServiceSpec{
			ExternalIPs: ips,
			Ports: []corev1.ServicePort{
				{
					Port: ports[0],
					Name: "3333",
				},
				{
					Port: ports[1],
					Name: "4444",
				},
			},
		},
	}
	servAnnotations := map[string]string{
		"key": "val",
	}
	statusIPS := []string{"20.20.20.20", "21.21.21.21"}
	servStatus := corev1.ServiceStatus{
		LoadBalancer: corev1.LoadBalancerStatus{
			Ingress: []corev1.LoadBalancerIngress{
				{IP: statusIPS[0]},
				{IP: statusIPS[1]},
			},
		},
	}

	cases := []struct {
		id      string
		ns      *corev1.Namespace
		serv    *corev1.Service
		expNs   *sr.Namespace
		expServ *sr.Service
		expEndp []*sr.Endpoint
		expErr  error
	}{
		{
			id:     "empty-ns",
			expErr: sr.ErrNsNotProvided,
		},
		{
			id:     "empty-serv",
			ns:     &corev1.Namespace{},
			expErr: sr.ErrServNotProvided,
		},
		{
			id:      "empty-metadatas-external-ips",
			ns:      nsToTest,
			serv:    servToTest,
			expNs:   &sr.Namespace{Name: nsToTest.Name, Metadata: map[string]string{}},
			expServ: &sr.Service{NsName: servToTest.Namespace, Name: servToTest.Name, Metadata: map[string]string{}},
			expEndp: []*sr.Endpoint{
				{
					NsName:   servToTest.Namespace,
					ServName: servToTest.Name,
					Name: func() string {
						toBeHashed := fmt.Sprintf("%s-%d", ips[0], servToTest.Spec.Ports[0].Port)
						h := sha256.New()
						h.Write([]byte(toBeHashed))
						return fmt.Sprintf("%s-%s", servToTest.Name, hex.EncodeToString(h.Sum(nil))[:10])
					}(),
					Address:  ips[0],
					Port:     servToTest.Spec.Ports[0].Port,
					Metadata: map[string]string{},
				},
				{
					NsName:   servToTest.Namespace,
					ServName: servToTest.Name,
					Name: func() string {
						toBeHashed := fmt.Sprintf("%s-%d", ips[0], servToTest.Spec.Ports[1].Port)
						h := sha256.New()
						h.Write([]byte(toBeHashed))
						return fmt.Sprintf("%s-%s", servToTest.Name, hex.EncodeToString(h.Sum(nil))[:10])
					}(),
					Address:  ips[0],
					Port:     servToTest.Spec.Ports[1].Port,
					Metadata: map[string]string{},
				},
				{
					NsName:   servToTest.Namespace,
					ServName: servToTest.Name,
					Name: func() string {
						toBeHashed := fmt.Sprintf("%s-%d", ips[1], servToTest.Spec.Ports[0].Port)
						h := sha256.New()
						h.Write([]byte(toBeHashed))
						return fmt.Sprintf("%s-%s", servToTest.Name, hex.EncodeToString(h.Sum(nil))[:10])
					}(),
					Address:  ips[1],
					Port:     servToTest.Spec.Ports[0].Port,
					Metadata: map[string]string{},
				},
				{
					NsName:   servToTest.Namespace,
					ServName: servToTest.Name,
					Name: func() string {
						toBeHashed := fmt.Sprintf("%s-%d", ips[1], servToTest.Spec.Ports[1].Port)
						h := sha256.New()
						h.Write([]byte(toBeHashed))
						return fmt.Sprintf("%s-%s", servToTest.Name, hex.EncodeToString(h.Sum(nil))[:10])
					}(),
					Address:  ips[1],
					Port:     servToTest.Spec.Ports[1].Port,
					Metadata: map[string]string{},
				},
			},
		},
		{
			id: "not-empty-metadatas-ingress-ips",
			ns: func() *corev1.Namespace {
				n := nsToTest.DeepCopy()
				n.Annotations = nsAnnotations
				return n
			}(),
			serv: func() *corev1.Service {
				s := servToTest.DeepCopy()
				s.Spec.ExternalIPs = []string{}
				s.Status = servStatus
				s.Annotations = servAnnotations
				return s
			}(),
			expNs:   &sr.Namespace{Name: nsToTest.Name, Metadata: nsAnnotations},
			expServ: &sr.Service{NsName: servToTest.Namespace, Name: servToTest.Name, Metadata: servAnnotations},
			expEndp: []*sr.Endpoint{
				{
					NsName:   servToTest.Namespace,
					ServName: servToTest.Name,
					Name: func() string {
						toBeHashed := fmt.Sprintf("%s-%d", statusIPS[0], servToTest.Spec.Ports[0].Port)
						h := sha256.New()
						h.Write([]byte(toBeHashed))
						return fmt.Sprintf("%s-%s", servToTest.Name, hex.EncodeToString(h.Sum(nil))[:10])
					}(),
					Address:  statusIPS[0],
					Port:     servToTest.Spec.Ports[0].Port,
					Metadata: map[string]string{},
				},
				{
					NsName:   servToTest.Namespace,
					ServName: servToTest.Name,
					Name: func() string {
						toBeHashed := fmt.Sprintf("%s-%d", statusIPS[0], servToTest.Spec.Ports[1].Port)
						h := sha256.New()
						h.Write([]byte(toBeHashed))
						return fmt.Sprintf("%s-%s", servToTest.Name, hex.EncodeToString(h.Sum(nil))[:10])
					}(),
					Address:  statusIPS[0],
					Port:     servToTest.Spec.Ports[1].Port,
					Metadata: map[string]string{},
				},
				{
					NsName:   servToTest.Namespace,
					ServName: servToTest.Name,
					Name: func() string {
						toBeHashed := fmt.Sprintf("%s-%d", statusIPS[1], servToTest.Spec.Ports[0].Port)
						h := sha256.New()
						h.Write([]byte(toBeHashed))
						return fmt.Sprintf("%s-%s", servToTest.Name, hex.EncodeToString(h.Sum(nil))[:10])
					}(),
					Address:  statusIPS[1],
					Port:     servToTest.Spec.Ports[0].Port,
					Metadata: map[string]string{},
				},
				{
					NsName:   servToTest.Namespace,
					ServName: servToTest.Name,
					Name: func() string {
						toBeHashed := fmt.Sprintf("%s-%d", statusIPS[1], servToTest.Spec.Ports[1].Port)
						h := sha256.New()
						h.Write([]byte(toBeHashed))
						return fmt.Sprintf("%s-%s", servToTest.Name, hex.EncodeToString(h.Sum(nil))[:10])
					}(),
					Address:  statusIPS[1],
					Port:     servToTest.Spec.Ports[1].Port,
					Metadata: map[string]string{},
				},
			},
		},
	}

	for _, currCase := range cases {
		n, s, e, err := e.ExtractData(currCase.ns, currCase.serv)
		errN := a.Equal(currCase.expNs, n)
		errS := a.Equal(currCase.expServ, s)
		errErr := a.Equal(currCase.expErr, err)
		errLen := a.Len(e, len(currCase.expEndp))
		// errE := a.Equal(currCase.expEndp, e)

		for _, eEndp := range currCase.expEndp {
			found := false

			for _, endp := range e {
				if endp.Name == eEndp.Name {
					found = true
					if !a.Equal(eEndp, endp) {
						a.FailNow("case %s failed", currCase.id)
					}
					break
				}
			}

			if !found {
				a.FailNow("case %s failed: endpoint %s not found", currCase.id, eEndp.Name)
			}
		}

		if !errN || !errS || /*!errE ||*/ !errErr || !errLen {
			a.FailNow(fmt.Sprintf("case %s failed", currCase.id))
		}
	}

	// statusIPS := []string{"20.20.20.20", "21.21.21.21", "22.22.22.22"}
	// servStatus := corev1.ServiceStatus{
	// 	LoadBalancer: corev1.LoadBalancerStatus{
	// 		Ingress: []corev1.LoadBalancerIngress{
	// 			{IP: "20.20.20.20"},
	// 			{IP: "21.21.21.21"},
	// 			{IP: "22.22.22.22"},
	// 		},
	// 	},
	// }

	// a := assert.New(t)

	// ns, serv, endp, err := e.ExtractData(nsToTest, nil)
	// a.Nil(ns)
	// a.Nil(serv)
	// a.Nil(endp)
	// a.Equal(sr.ErrServNotProvided, err)

	// ns, serv, endp, err = e.ExtractData(nil, servToTest)
	// a.Nil(ns)
	// a.Nil(serv)
	// a.Nil(endp)
	// a.Equal(sr.ErrNsNotProvided, err)

	// ns, serv, endp, err = e.ExtractData(nsToTest, servToTest)
	// a.NotNil(ns)
	// a.NotNil(serv)
	// a.NotNil(endp)
	// a.NoError(err)
	// a.Equal(&sr.Namespace{
	// 	Name:     nsName,
	// 	Metadata: nsToTest.Annotations,
	// }, ns)
	// a.Equal(&sr.Service{
	// 	Name:     servName,
	// 	NsName:   nsName,
	// 	Metadata: servToTest.Annotations,
	// }, serv)
	// a.Len(endp, 4)
	// for _, e := range endp {
	// 	a.Contains(ips, e.Address)
	// 	a.Contains(ports, e.Port)
	// 	a.Empty(e.Metadata)
	// 	a.Equal(nsName, e.NsName)
	// 	a.Equal(servName, e.ServName)

	// 	if !strings.HasPrefix(e.Name, servName+"-") {
	// 		a.Fail("endpoint name is incorrect. Should start with", servName, "but is", e.Name)
	// 	}

	// 	suffix := e.Name[len(servName)+1:]
	// 	a.Len(suffix, 10)
	// }

	// servToTest.Status = servStatus
	// ns, serv, endp, err = e.ExtractData(nsToTest, servToTest)
	// a.NotNil(ns)
	// a.NotNil(serv)
	// a.NotNil(endp)
	// a.NoError(err)
	// a.Equal(&sr.Namespace{
	// 	Name:     nsName,
	// 	Metadata: nsToTest.Annotations,
	// }, ns)
	// a.Equal(&sr.Service{
	// 	Name:     servName,
	// 	NsName:   nsName,
	// 	Metadata: servToTest.Annotations,
	// }, serv)
	// expIPS := []string{}
	// expIPS = append(expIPS, ips...)
	// expIPS = append(expIPS, statusIPS...)
	// a.Len(endp, len(expIPS)*len(servToTest.Spec.Ports))
	// for _, e := range endp {
	// 	a.Contains(expIPS, e.Address)
	// 	a.Contains(ports, e.Port)
	// 	a.Empty(e.Metadata)
	// 	a.Equal(nsName, e.NsName)
	// 	a.Equal(servName, e.ServName)

	// 	if !strings.HasPrefix(e.Name, servName+"-") {
	// 		a.Fail("endpoint name is incorrect. Should start with", servName, "but is", e.Name)
	// 	}

	// 	suffix := e.Name[len(servName)+1:]
	// 	a.Len(suffix, 10)
	// }
}
