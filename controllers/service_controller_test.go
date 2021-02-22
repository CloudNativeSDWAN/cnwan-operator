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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ktypes "k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakecli "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func TestSrvUpdatePredicate(t *testing.T) {
	a := assert.New(t)
	cli := func() client.Client {
		scheme := runtime.NewScheme()
		corev1.AddToScheme(scheme)
		return fakecli.NewFakeClientWithScheme(scheme, &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "ns-name",
				Labels: map[string]string{string(types.AllowedKey): "whatever"},
			},
		})
	}()
	u := &Utils{
		AllowedAnnotations: []string{"yes"},
		CurrentNsPolicy:    types.AllowList,
	}
	cases := []struct {
		old      *corev1.Service
		curr     *corev1.Service
		expRes   bool
		expCache map[string]bool
	}{
		{
			old: &corev1.Service{
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
			curr: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "srv-name",
					Namespace:   "ns-name",
					Annotations: map[string]string{"no": "no", "other": "no"},
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
			old: &corev1.Service{
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
			curr: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "srv-name",
					Namespace:   "ns-name",
					Annotations: map[string]string{"yes": "yes"},
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
			expRes:   true,
			expCache: map[string]bool{ktypes.NamespacedName{Namespace: "ns-name", Name: "srv-name"}.String(): true},
		},
		{
			curr: &corev1.Service{
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
			old: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "srv-name",
					Namespace:   "ns-name",
					Annotations: map[string]string{"yes": "yes"},
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
			expRes:   true,
			expCache: map[string]bool{ktypes.NamespacedName{Namespace: "ns-name", Name: "srv-name"}.String(): true},
		},
		{
			old: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "srv-name",
					Namespace:   "ns-name",
					Annotations: map[string]string{"yes": "yes"},
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
			curr: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "srv-name",
					Namespace:   "ns-name",
					Annotations: map[string]string{"yes": "changed"},
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
			expRes:   true,
			expCache: map[string]bool{ktypes.NamespacedName{Namespace: "ns-name", Name: "srv-name"}.String(): true},
		},
		{
			old: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "srv-name",
					Namespace:   "ns-name",
					Annotations: map[string]string{"yes": "yes"},
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
			curr: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "srv-name",
					Namespace:   "ns-name",
					Annotations: map[string]string{"yes": "yes"},
				},
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeLoadBalancer,
				},
				Status: corev1.ServiceStatus{
					LoadBalancer: corev1.LoadBalancerStatus{
						Ingress: []corev1.LoadBalancerIngress{
							{IP: "10.10.10.10"},
							{IP: "10.10.10.11"},
						},
					},
				},
			},
			expRes:   true,
			expCache: map[string]bool{ktypes.NamespacedName{Namespace: "ns-name", Name: "srv-name"}.String(): true},
		},
		{
			old: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "srv-name",
					Namespace:   "ns-name",
					Annotations: map[string]string{"yes": "yes"},
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
			curr: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "srv-name",
					Namespace:   "ns-name",
					Annotations: map[string]string{"yes": "yes"},
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
	}
	failed := func(i int) {
		a.FailNow(fmt.Sprintf("case %d failed", i))
	}
	for i, currCase := range cases {
		s := &ServiceReconciler{
			Client:        cli,
			Log:           ctrl.Log.WithName("test"),
			Utils:         u,
			cacheSrvWatch: map[string]bool{},
		}

		res := s.updatePredicate(event.UpdateEvent{ObjectOld: currCase.old, ObjectNew: currCase.curr})
		if !a.Equal(currCase.expRes, res) {
			failed(i)
		}
	}
}
