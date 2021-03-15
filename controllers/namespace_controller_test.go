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
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ktypes "k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func TestNsCreatePredicate(t *testing.T) {
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
	a := assert.New(t)
	cases := []struct {
		ev         event.CreateEvent
		currPolicy types.ListPolicy
		expRes     bool
		expCache   map[string]bool
	}{
		{
			currPolicy: types.AllowList,
			ev: event.CreateEvent{
				Meta: &v1.ObjectMeta{
					Name:   "ns-name",
					Labels: map[string]string{"whatever": "whatever"},
				},
			},
			expCache: map[string]bool{},
		},
		{
			currPolicy: types.AllowList,
			ev: event.CreateEvent{
				Meta: &v1.ObjectMeta{
					Name:   "ns-name",
					Labels: map[string]string{string(types.AllowedKey): "whatever"},
				},
			},
			expCache: map[string]bool{ktypes.NamespacedName{Name: "ns-name"}.String(): true},
			expRes:   true,
		},
		{
			currPolicy: types.BlockList,
			ev: event.CreateEvent{
				Meta: &v1.ObjectMeta{
					Name:   "ns-name",
					Labels: map[string]string{"whatever": "whatever"},
				},
			},
			expCache: map[string]bool{ktypes.NamespacedName{Name: "ns-name"}.String(): true},
			expRes:   true,
		},
		{
			currPolicy: types.BlockList,
			ev: event.CreateEvent{
				Meta: &v1.ObjectMeta{
					Name:   "ns-name",
					Labels: map[string]string{string(types.BlockedKey): "whatever"},
				},
			},
			expCache: map[string]bool{},
		},
	}
	failed := func(i int) {
		a.FailNow(fmt.Sprintf("case %d failed", i))
	}
	for i, currCase := range cases {
		n := &NamespaceReconciler{
			BaseReconciler: &BaseReconciler{
				CurrentNsPolicy: currCase.currPolicy,
				Log:             ctrl.Log.WithName("controllers"),
			},
			cacheNsWatch: map[string]bool{},
		}

		res := n.createPredicate(currCase.ev)
		if !a.Equal(currCase.expRes, res) || !a.Equal(currCase.expCache, n.cacheNsWatch) {
			failed(i)
		}
	}
}

func TestNsDeletePredicate(t *testing.T) {
	a := assert.New(t)

	cases := []struct {
		ev         event.DeleteEvent
		currPolicy types.ListPolicy
		expRes     bool
		expCache   map[string]bool
	}{
		{
			currPolicy: types.AllowList,
			ev: event.DeleteEvent{
				Meta: &v1.ObjectMeta{
					Name:   "ns-name",
					Labels: map[string]string{"whatever": "whatever"},
				},
			},
			expCache: map[string]bool{},
		},
		{
			currPolicy: types.AllowList,
			ev: event.DeleteEvent{
				Meta: &v1.ObjectMeta{
					Name:   "ns-name",
					Labels: map[string]string{string(types.AllowedKey): "whatever"},
				},
			},
			expCache: map[string]bool{ktypes.NamespacedName{Name: "ns-name"}.String(): false},
			expRes:   true,
		},
		{
			currPolicy: types.BlockList,
			ev: event.DeleteEvent{
				Meta: &v1.ObjectMeta{
					Name:   "ns-name",
					Labels: map[string]string{"whatever": "whatever"},
				},
			},
			expCache: map[string]bool{ktypes.NamespacedName{Name: "ns-name"}.String(): false},
			expRes:   true,
		},
		{
			currPolicy: types.BlockList,
			ev: event.DeleteEvent{
				Meta: &v1.ObjectMeta{
					Name:   "ns-name",
					Labels: map[string]string{string(types.BlockedKey): "whatever"},
				},
			},
			expCache: map[string]bool{},
		},
	}
	failed := func(i int) {
		a.FailNow(fmt.Sprintf("case %d failed", i))
	}
	for i, currCase := range cases {
		n := &NamespaceReconciler{
			BaseReconciler: &BaseReconciler{CurrentNsPolicy: currCase.currPolicy},
			cacheNsWatch:   map[string]bool{},
		}

		res := n.deletePredicate(currCase.ev)
		if !a.Equal(currCase.expRes, res) || !a.Equal(currCase.expCache, n.cacheNsWatch) {
			failed(i)
		}
	}
}

func TestNsUpdatePredicate(t *testing.T) {
	a := assert.New(t)

	cases := []struct {
		ev         event.UpdateEvent
		currPolicy types.ListPolicy
		expRes     bool
		expCache   map[string]bool
	}{
		{
			currPolicy: types.AllowList,
			ev: event.UpdateEvent{
				MetaOld: &v1.ObjectMeta{
					Name:   "ns-name",
					Labels: map[string]string{"whatever": "whatever"},
				},
				MetaNew: &v1.ObjectMeta{
					Name:   "ns-name",
					Labels: map[string]string{"whatever": "whatever"},
				},
			},
			expCache: map[string]bool{},
		},
		{
			currPolicy: types.AllowList,
			ev: event.UpdateEvent{
				MetaOld: &v1.ObjectMeta{
					Name:   "ns-name",
					Labels: map[string]string{types.AllowedKey: "whatever"},
				},
				MetaNew: &v1.ObjectMeta{
					Name:   "ns-name",
					Labels: map[string]string{types.AllowedKey: "whatever"},
				},
			},
			expCache: map[string]bool{},
		},
		{
			currPolicy: types.BlockList,
			ev: event.UpdateEvent{
				MetaOld: &v1.ObjectMeta{
					Name:   "ns-name",
					Labels: map[string]string{"whatever": "whatever"},
				},
				MetaNew: &v1.ObjectMeta{
					Name:   "ns-name",
					Labels: map[string]string{"whatever": "whatever"},
				},
			},
			expCache: map[string]bool{},
		},
		{
			currPolicy: types.BlockList,
			ev: event.UpdateEvent{
				MetaOld: &v1.ObjectMeta{
					Name:   "ns-name",
					Labels: map[string]string{types.BlockedKey: "whatever"},
				},
				MetaNew: &v1.ObjectMeta{
					Name:   "ns-name",
					Labels: map[string]string{types.BlockedKey: "whatever"},
				},
			},
			expCache: map[string]bool{},
		},
		{
			currPolicy: types.AllowList,
			ev: event.UpdateEvent{
				MetaOld: &v1.ObjectMeta{
					Name:   "ns-name",
					Labels: map[string]string{"whatever": "whatever"},
				},
				MetaNew: &v1.ObjectMeta{
					Name:   "ns-name",
					Labels: map[string]string{types.AllowedKey: "whatever"},
				},
			},
			expCache: map[string]bool{ktypes.NamespacedName{Name: "ns-name"}.String(): true},
			expRes:   true,
		},
		{
			currPolicy: types.AllowList,
			ev: event.UpdateEvent{
				MetaOld: &v1.ObjectMeta{
					Name:   "ns-name",
					Labels: map[string]string{types.AllowedKey: "whatever"},
				},
				MetaNew: &v1.ObjectMeta{
					Name:   "ns-name",
					Labels: map[string]string{"whatever": "whatever"},
				},
			},
			expCache: map[string]bool{ktypes.NamespacedName{Name: "ns-name"}.String(): false},
			expRes:   true,
		},
		{
			currPolicy: types.BlockList,
			ev: event.UpdateEvent{
				MetaOld: &v1.ObjectMeta{
					Name:   "ns-name",
					Labels: map[string]string{"whatever": "whatever"},
				},
				MetaNew: &v1.ObjectMeta{
					Name:   "ns-name",
					Labels: map[string]string{types.BlockedKey: "whatever"},
				},
			},
			expCache: map[string]bool{ktypes.NamespacedName{Name: "ns-name"}.String(): false},
			expRes:   true,
		},
		{
			currPolicy: types.BlockList,
			ev: event.UpdateEvent{
				MetaOld: &v1.ObjectMeta{
					Name:   "ns-name",
					Labels: map[string]string{types.BlockedKey: "whatever"},
				},
				MetaNew: &v1.ObjectMeta{
					Name:   "ns-name",
					Labels: map[string]string{"whatever": "whatever"},
				},
			},
			expCache: map[string]bool{ktypes.NamespacedName{Name: "ns-name"}.String(): true},
			expRes:   true,
		},
	}
	failed := func(i int) {
		a.FailNow(fmt.Sprintf("case %d failed", i))
	}
	for i, currCase := range cases {
		n := &NamespaceReconciler{
			BaseReconciler: &BaseReconciler{CurrentNsPolicy: currCase.currPolicy},
			cacheNsWatch:   map[string]bool{},
		}

		res := n.updatePredicate(currCase.ev)
		if !a.Equal(currCase.expRes, res) || !a.Equal(currCase.expCache, n.cacheNsWatch) {
			failed(i)
		}
	}
}
