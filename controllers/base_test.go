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

package controllers

import (
	"fmt"
	"testing"

	"github.com/CloudNativeSDWAN/cnwan-operator/internal/types"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/discovery/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakecli "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestShouldWatchNs(t *testing.T) {
	a := assert.New(t)

	cases := []struct {
		policy types.ListPolicy
		labels map[string]string
		expRes bool
	}{
		{
			policy: types.AllowList,
		},
		{
			policy: types.AllowList,
			labels: map[string]string{types.BlockedKey: "whatever"},
		},
		{
			policy: types.AllowList,
			labels: map[string]string{types.AllowedKey: "whatever"},
			expRes: true,
		},
		{
			policy: types.BlockList,
			expRes: true,
		},
		{
			policy: types.BlockList,
			labels: map[string]string{types.AllowedKey: "whatever"},
			expRes: true,
		},
		{
			policy: types.BlockList,
			labels: map[string]string{types.BlockedKey: "whatever"},
		},
	}

	failed := func(i int) {
		a.FailNow(fmt.Sprintf("case %d failed", i))
	}
	for i, currCase := range cases {
		b := &BaseReconciler{CurrentNsPolicy: currCase.policy}
		res := b.shouldWatchNs(currCase.labels)

		if !a.Equal(currCase.expRes, res) {
			failed(i)
		}
	}
}

func TestFilterAnnotations(t *testing.T) {
	a := assert.New(t)
	annotations := map[string]string{
		"one.prefix.com/first-name":  "one-first-value",
		"one.prefix.com/second-name": "one-second-value",
		"one-no-prefix-label":        "one-no-prefix-value",
		"two-no-prefix-label":        "two-no-prefix-value",
		"two.prefix.com/first-name":  "two-first-value",
		"two.prefix.com/second-name": "two-second-value",
	}

	cases := []struct {
		allowed     map[string]bool
		annotations map[string]string
		expRes      map[string]string
	}{
		{
			expRes: map[string]string{},
		},
		{
			annotations: map[string]string{"whatever": "whatever"},
			expRes:      map[string]string{},
		},
		{
			annotations: annotations,
			allowed:     map[string]bool{"*/*": true},
			expRes:      annotations,
		},
		{
			annotations: annotations,
			allowed:     map[string]bool{"one-no-prefix-label": true, "two-no-prefix-label": true},
			expRes:      map[string]string{"one-no-prefix-label": "one-no-prefix-value", "two-no-prefix-label": "two-no-prefix-value"},
		},
		{
			annotations: annotations,
			allowed:     map[string]bool{"one.prefix.com/*": true},
			expRes:      map[string]string{"one.prefix.com/first-name": "one-first-value", "one.prefix.com/second-name": "one-second-value"},
		},
		{
			annotations: annotations,
			allowed:     map[string]bool{"*/first-name": true},
			expRes:      map[string]string{"one.prefix.com/first-name": "one-first-value", "two.prefix.com/first-name": "two-first-value"},
		},
		{
			annotations: annotations,
			allowed:     map[string]bool{"*/first-name": true, "one-no-prefix-label": true},
			expRes:      map[string]string{"one.prefix.com/first-name": "one-first-value", "two.prefix.com/first-name": "two-first-value", "one-no-prefix-label": "one-no-prefix-value"},
		},
	}

	failed := func(i int) {
		a.FailNow(fmt.Sprintf("case %d failed", i))
	}
	for i, currCase := range cases {
		b := &BaseReconciler{AllowedAnnotations: currCase.allowed}
		res := b.filterAnnotations(currCase.annotations)

		if !a.Equal(currCase.expRes, res) {
			failed(i)
		}
	}
}

func TestShouldWatchSrv(t *testing.T) {
	a := assert.New(t)
	cases := []struct {
		srv    *corev1.Service
		cli    client.Client
		expRes bool
	}{
		{
			srv: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{Name: "srv-name", Namespace: "ns-name"},
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeClusterIP,
				},
			},
		},
		{
			srv: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{Name: "srv-name", Namespace: "ns-name"},
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeLoadBalancer,
				},
				Status: corev1.ServiceStatus{
					LoadBalancer: corev1.LoadBalancerStatus{
						Ingress: []corev1.LoadBalancerIngress{},
					},
				},
			},
		},
		{
			srv: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "srv-name",
					Namespace:   "ns-name",
					Annotations: map[string]string{"no": "no"},
				},
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeLoadBalancer,
				},
				Status: corev1.ServiceStatus{
					LoadBalancer: corev1.LoadBalancerStatus{
						Ingress: []corev1.LoadBalancerIngress{
							{IP: "10.10.10.10"},
						},
					},
				},
			},
		},
		{
			cli: fakecli.NewFakeClientWithScheme(&runtime.Scheme{}),
			srv: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "srv-name",
					Namespace:   "ns-name",
					Annotations: map[string]string{"yes": "no"},
				},
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeLoadBalancer,
				},
				Status: corev1.ServiceStatus{
					LoadBalancer: corev1.LoadBalancerStatus{
						Ingress: []corev1.LoadBalancerIngress{
							{IP: "10.10.10.10"},
						},
					},
				},
			},
		},
		{
			cli: func() client.Client {
				scheme := runtime.NewScheme()
				corev1.AddToScheme(scheme)
				return fakecli.NewFakeClientWithScheme(scheme, &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "ns-name",
					},
				})
			}(),
			srv: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "srv-name",
					Namespace:   "ns-name",
					Annotations: map[string]string{"yes": "no"},
				},
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeLoadBalancer,
				},
				Status: corev1.ServiceStatus{
					LoadBalancer: corev1.LoadBalancerStatus{
						Ingress: []corev1.LoadBalancerIngress{
							{IP: "10.10.10.10"},
						},
					},
				},
			},
		},
		{
			cli: func() client.Client {
				scheme := runtime.NewScheme()
				corev1.AddToScheme(scheme)
				return fakecli.NewFakeClientWithScheme(scheme, &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "ns-name",
						Labels: map[string]string{string(types.AllowedKey): "whatever"},
					},
				})
			}(),
			srv: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "srv-name",
					Namespace:   "ns-name",
					Annotations: map[string]string{"yes": "no"},
				},
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeLoadBalancer,
				},
				Status: corev1.ServiceStatus{
					LoadBalancer: corev1.LoadBalancerStatus{
						Ingress: []corev1.LoadBalancerIngress{
							{IP: "10.10.10.10"},
						},
					},
				},
			},
			expRes: true,
		},
	}
	failed := func(i int) {
		a.FailNow(fmt.Sprintf("case %d failed", i))
	}
	for i, currCase := range cases {
		b := &BaseReconciler{
			Client:             currCase.cli,
			Log:                ctrl.Log.WithName("test"),
			AllowedAnnotations: map[string]bool{"yes": true},
			CurrentNsPolicy:    types.AllowList,
		}

		res := b.shouldWatchSrv(currCase.srv)
		if !a.Equal(currCase.expRes, res) {
			failed(i)
		}
	}
}

func TestShouldWatchEpSlice(t *testing.T) {
	a := assert.New(t)
	cases := []struct {
		eps    *v1beta1.EndpointSlice
		cli    client.Client
		expRes bool
	}{
		{
			eps: &v1beta1.EndpointSlice{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "ns-name",
					Labels:    map[string]string{},
				},
			},
		},
		{
			eps: &v1beta1.EndpointSlice{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "ns-name",
					Labels: map[string]string{
						v1beta1.LabelServiceName: "srv-name",
					},
				},
			},
			cli: func() client.Client {
				scheme := runtime.NewScheme()
				corev1.AddToScheme(scheme)
				return fakecli.NewFakeClientWithScheme(scheme)
			}(),
		},
		{
			eps: &v1beta1.EndpointSlice{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "ns-name",
					Labels: map[string]string{
						v1beta1.LabelServiceName: "srv-name",
					},
				},
			},
			cli: func() client.Client {
				scheme := runtime.NewScheme()
				corev1.AddToScheme(scheme)
				return fakecli.NewFakeClientWithScheme(scheme, &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "ns-name",
						Labels: map[string]string{},
					},
				}, &corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "srv-name",
						Namespace: "ns-name",
						Labels:    map[string]string{},
					},
				})
			}(),
		},
		{
			eps: &v1beta1.EndpointSlice{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "ns-name",
					Labels: map[string]string{
						v1beta1.LabelServiceName: "srv-name",
					},
				},
			},
			cli: func() client.Client {
				scheme := runtime.NewScheme()
				corev1.AddToScheme(scheme)
				return fakecli.NewFakeClientWithScheme(scheme, &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "ns-name",
						Labels: map[string]string{string(types.AllowedKey): "whatever"},
					},
				}, &corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "srv-name",
						Namespace:   "ns-name",
						Annotations: map[string]string{"yes": "no"},
						Labels:      map[string]string{countPodsLabelKey: disableVal},
					},
					Spec: corev1.ServiceSpec{
						Type: corev1.ServiceTypeLoadBalancer,
					},
					Status: corev1.ServiceStatus{
						LoadBalancer: corev1.LoadBalancerStatus{
							Ingress: []corev1.LoadBalancerIngress{
								{IP: "10.10.10.10"},
							},
						},
					},
				})
			}(),
		},
		{
			eps: &v1beta1.EndpointSlice{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "ns-name",
					Labels: map[string]string{
						v1beta1.LabelServiceName: "srv-name",
					},
				},
			},
			cli: func() client.Client {
				scheme := runtime.NewScheme()
				corev1.AddToScheme(scheme)
				return fakecli.NewFakeClientWithScheme(scheme, &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "ns-name",
						Labels: map[string]string{string(types.AllowedKey): "whatever"},
					},
				}, &corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "srv-name",
						Namespace:   "ns-name",
						Annotations: map[string]string{"yes": "no"},
						Labels:      map[string]string{countPodsLabelKey: enableVal},
					},
					Spec: corev1.ServiceSpec{
						Type: corev1.ServiceTypeLoadBalancer,
					},
					Status: corev1.ServiceStatus{
						LoadBalancer: corev1.LoadBalancerStatus{
							Ingress: []corev1.LoadBalancerIngress{
								{IP: "10.10.10.10"},
							},
						},
					},
				})
			}(),
		},
		{
			eps: &v1beta1.EndpointSlice{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "ns-name",
					Labels: map[string]string{
						v1beta1.LabelServiceName: "srv-name",
					},
				},
				AddressType: v1beta1.AddressTypeIPv4,
			},
			cli: func() client.Client {
				scheme := runtime.NewScheme()
				corev1.AddToScheme(scheme)
				return fakecli.NewFakeClientWithScheme(scheme, &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "ns-name",
						Labels: map[string]string{string(types.AllowedKey): "whatever"},
					},
				}, &corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "srv-name",
						Namespace:   "ns-name",
						Annotations: map[string]string{"yes": "no"},
						Labels:      map[string]string{countPodsLabelKey: enableVal},
					},
					Spec: corev1.ServiceSpec{
						Type: corev1.ServiceTypeLoadBalancer,
					},
					Status: corev1.ServiceStatus{
						LoadBalancer: corev1.LoadBalancerStatus{
							Ingress: []corev1.LoadBalancerIngress{
								{IP: "10.10.10.10"},
							},
						},
					},
				})
			}(),
			expRes: true,
		},
	}
	failed := func(i int) {
		a.FailNow(fmt.Sprintf("case %d failed", i))
	}
	for i, currCase := range cases {
		b := &BaseReconciler{
			Client:             currCase.cli,
			Log:                ctrl.Log.WithName("test"),
			AllowedAnnotations: map[string]bool{"yes": true},
			CurrentNsPolicy:    types.AllowList,
		}

		res := b.shouldWatchEpSlice(currCase.eps)
		if !a.Equal(currCase.expRes, res) {
			failed(i)
		}
	}
}
