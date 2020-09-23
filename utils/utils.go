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
	"fmt"
	"strings"

	"github.com/CloudNativeSDWAN/cnwan-operator/types"
	"github.com/spf13/viper"
)

const (
	hashFormat string = "%s:%d"
	hashChars  int    = 10
)

// FilterAnnotations is used to remove annotations that should be ignored
// by the operator
func FilterAnnotations(annotations map[string]string) map[string]string {
	allowedAnnotations := map[string]bool{}
	if viper.Get(types.AllowedAnnotationsMap) != nil {
		allowedAnnotations = viper.Get(types.AllowedAnnotationsMap).(map[string]bool)
	}

	if _, exists := allowedAnnotations["*/*"]; exists {
		return annotations
	}

	filtered := map[string]string{}
	for key, val := range annotations {

		// Check this key specifically
		if _, exists := allowedAnnotations[key]; exists {
			filtered[key] = val
			continue
		}

		prefixName := strings.Split(key, "/")
		if len(prefixName) != 2 {
			// This key is not in prefix/name format
			continue
		}

		prefixWildcard := fmt.Sprintf("%s/*", prefixName[0])
		if _, exists := allowedAnnotations[prefixWildcard]; exists {
			filtered[key] = val
			continue
		}

		wildcardName := fmt.Sprintf("*/%s", prefixName[1])
		if _, exists := allowedAnnotations[wildcardName]; exists {
			filtered[key] = val
		}
	}

	return filtered
}
