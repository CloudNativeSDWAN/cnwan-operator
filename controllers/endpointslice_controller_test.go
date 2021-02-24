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
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func TestEpSliceCreatePredicate(t *testing.T) {
	a := assert.New(t)
	cli := func() client.Client {
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
	}()
	cases := []struct {
		ev      event.CreateEvent
		expRes  bool
		expData map[string]*epsData
	}{
		{
			ev: event.CreateEvent{
				Object: &v1beta1.EndpointSlice{},
			},
			expData: map[string]*epsData{},
		},
		{
			ev: event.CreateEvent{
				Object: &v1beta1.EndpointSlice{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "epslice-1",
						Namespace: "ns-name",
						Labels:    map[string]string{v1beta1.LabelServiceName: "srv-name"},
					},
					AddressType: v1beta1.AddressTypeIPv4,
					Endpoints: []v1beta1.Endpoint{
						{
							Addresses: []string{"10.10.10.9"}, Conditions: v1beta1.EndpointConditions{
								Ready: func() *bool {
									f := false
									return &f
								}(),
							},
						},
						{Addresses: []string{"10.10.10.10", "10.10.10.11"}}},
				},
			},
			expData: map[string]*epsData{
				"ns-name/epslice-1": {
					count: 2,
					srv:   "srv-name",
				},
			},
			expRes: true,
		},
	}
	failed := func(i int) {
		a.FailNow(fmt.Sprintf("case %d failed", i))
	}
	for i, currCase := range cases {
		eps := &EndpointSliceReconciler{
			BaseReconciler: &BaseReconciler{
				Client:             cli,
				Log:                ctrl.Log.WithName("test"),
				AllowedAnnotations: map[string]bool{"yes": true},
				CurrentNsPolicy:    types.AllowList,
			},
			epsDataActions: map[string]*epsData{},
		}

		res := eps.createPredicate(currCase.ev)
		if !a.Equal(currCase.expRes, res) || !a.Equal(currCase.expData, eps.epsDataActions) {
			failed(i)
		}
	}
}

func TestEpSliceDeletePredicate(t *testing.T) {
	a := assert.New(t)
	cli := func() client.Client {
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
	}()
	cases := []struct {
		ev      event.DeleteEvent
		expRes  bool
		expData map[string]*epsData
	}{
		{
			ev: event.DeleteEvent{
				Object: &v1beta1.EndpointSlice{},
			},
			expData: map[string]*epsData{},
		},
		{
			ev: event.DeleteEvent{
				Object: &v1beta1.EndpointSlice{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "epslice-1",
						Namespace: "ns-name",
						Labels:    map[string]string{v1beta1.LabelServiceName: "srv-name"},
					},
					AddressType: v1beta1.AddressTypeIPv4,
					Endpoints: []v1beta1.Endpoint{
						{
							Addresses: []string{"10.10.10.9"}, Conditions: v1beta1.EndpointConditions{
								Ready: func() *bool {
									f := false
									return &f
								}(),
							},
						},
						{Addresses: []string{"10.10.10.10", "10.10.10.11"}}},
				},
			},
			expData: map[string]*epsData{
				"ns-name/epslice-1": {
					count: 0,
					srv:   "srv-name",
				},
			},
			expRes: true,
		},
	}
	failed := func(i int) {
		a.FailNow(fmt.Sprintf("case %d failed", i))
	}
	for i, currCase := range cases {
		eps := &EndpointSliceReconciler{
			BaseReconciler: &BaseReconciler{
				Client:             cli,
				Log:                ctrl.Log.WithName("test"),
				AllowedAnnotations: map[string]bool{"yes": true},
				CurrentNsPolicy:    types.AllowList,
			},
			epsDataActions: map[string]*epsData{},
		}

		res := eps.deletePredicate(currCase.ev)
		if !a.Equal(currCase.expRes, res) || !a.Equal(currCase.expData, eps.epsDataActions) {
			failed(i)
		}
	}
}
