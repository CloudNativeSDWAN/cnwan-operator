// Copyright © 2020 Cisco
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

package utils

import (
	"testing"

	"github.com/CloudNativeSDWAN/cnwan-operator/internal/types"
	"github.com/spf13/viper"
	. "github.com/stretchr/testify/assert"
)

func TestFilterAnnotations(t *testing.T) {
	annotations := map[string]string{
		"one.prefix.com/first-name":  "one-first-value",
		"one.prefix.com/second-name": "one-second-value",
		"one-no-prefix-label":        "one-no-prefix-value",
		"two-no-prefix-label":        "two-no-prefix-value",
		"two.prefix.com/first-name":  "two-first-value",
		"two.prefix.com/second-name": "two-second-value",
	}

	// Case 1: no annotations
	res := FilterAnnotations(map[string]string{})
	Empty(t, res)

	viper.Set(types.AllowedAnnotationsMap, map[string]bool{"one.prefix.com/first-name": true})
	res = FilterAnnotations(map[string]string{})
	Empty(t, res)

	// Case 2: specific annotations
	allowed := map[string]bool{
		"one.prefix.com/first-name": true,
		"one-no-prefix-label":       true,
		"three-no-prefix-label":     true,
	}
	expected := map[string]string{
		"one.prefix.com/first-name": "one-first-value",
		"one-no-prefix-label":       "one-no-prefix-value",
	}
	viper.Set(types.AllowedAnnotationsMap, allowed)
	res = FilterAnnotations(annotations)
	Equal(t, expected, res)

	// Case 3: with prefix wildcards
	allowed = map[string]bool{
		"one.prefix.com/*":      true,
		"one-no-prefix-label":   true,
		"three-no-prefix-label": true,
	}
	expected = map[string]string{
		"one.prefix.com/first-name":  "one-first-value",
		"one.prefix.com/second-name": "one-second-value",
		"one-no-prefix-label":        "one-no-prefix-value",
	}
	viper.Set(types.AllowedAnnotationsMap, allowed)
	res = FilterAnnotations(annotations)
	Equal(t, expected, res)

	// Case 3: with prefix names
	allowed = map[string]bool{
		"*/first-name":          true,
		"one-no-prefix-label":   true,
		"three-no-prefix-label": true,
	}
	expected = map[string]string{
		"one.prefix.com/first-name": "one-first-value",
		"two.prefix.com/first-name": "two-first-value",
		"one-no-prefix-label":       "one-no-prefix-value",
	}
	viper.Set(types.AllowedAnnotationsMap, allowed)
	res = FilterAnnotations(annotations)
	Equal(t, expected, res)

	// Case 4: all
	allowed = map[string]bool{
		"*/*": true,
	}
	viper.Set(types.AllowedAnnotationsMap, allowed)
	res = FilterAnnotations(annotations)
	Equal(t, annotations, res)
}
