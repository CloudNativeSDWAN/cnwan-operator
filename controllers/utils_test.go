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
		c := &Utils{CurrentNsPolicy: currCase.policy}
		res := c.ShouldWatchNs(currCase.labels)

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
		allowed     []string
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
			allowed:     []string{"*/*"},
			expRes:      annotations,
		},
		{
			annotations: annotations,
			allowed:     []string{"one-no-prefix-label", "two-no-prefix-label"},
			expRes:      map[string]string{"one-no-prefix-label": "one-no-prefix-value", "two-no-prefix-label": "two-no-prefix-value"},
		},
		{
			annotations: annotations,
			allowed:     []string{"one.prefix.com/*"},
			expRes:      map[string]string{"one.prefix.com/first-name": "one-first-value", "one.prefix.com/second-name": "one-second-value"},
		},
		{
			annotations: annotations,
			allowed:     []string{"*/first-name"},
			expRes:      map[string]string{"one.prefix.com/first-name": "one-first-value", "two.prefix.com/first-name": "two-first-value"},
		},
		{
			annotations: annotations,
			allowed:     []string{"*/first-name", "one-no-prefix-label"},
			expRes:      map[string]string{"one.prefix.com/first-name": "one-first-value", "two.prefix.com/first-name": "two-first-value", "one-no-prefix-label": "one-no-prefix-value"},
		},
	}

	failed := func(i int) {
		a.FailNow(fmt.Sprintf("case %d failed", i))
	}
	for i, currCase := range cases {
		c := &Utils{AllowedAnnotations: currCase.allowed}
		res := c.FilterAnnotations(currCase.annotations)

		if !a.Equal(currCase.expRes, res) {
			failed(i)
		}
	}
}
