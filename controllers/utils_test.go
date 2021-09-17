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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilterAnnotations(t *testing.T) {
	annotations := map[string]string{
		"stand-alone":           "alone",
		"another-stand-alone":   "another-alone",
		"prefix.io/one":         "one",
		"prefix.io/two":         "two",
		"another.io/first":      "another-first",
		"another.io/second":     "another-second",
		"yet-another.io/first":  "yet-first",
		"yet-another.io/second": "yet-second",
	}

	cases := []struct {
		annotations map[string]string
		filter      []string
		expRes      map[string]string
	}{
		{
			annotations: map[string]string{},
			filter:      []string{"whatever"},
			expRes:      map[string]string{},
		},
		{
			annotations: map[string]string{"whatever": "whatever"},
			filter:      []string{},
			expRes:      map[string]string{},
		},
		{
			annotations: annotations,
			filter:      []string{"*/*"},
			expRes:      annotations,
		},
		{
			annotations: annotations,
			filter:      []string{"stand-alone", "prefix.io/*", "*/first"},
			expRes: map[string]string{
				"stand-alone":          "alone",
				"prefix.io/one":        "one",
				"prefix.io/two":        "two",
				"another.io/first":     "another-first",
				"yet-another.io/first": "yet-first",
			},
		},
	}

	a := assert.New(t)
	for _, currCase := range cases {
		res := filterAnnotations(currCase.annotations, currCase.filter)
		a.Equal(currCase.expRes, res)
	}
}
