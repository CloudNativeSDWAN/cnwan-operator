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
	"strings"

	"github.com/CloudNativeSDWAN/cnwan-operator/internal/types"
)

// Utils contains the options that are used to set up and define
// the behavior of the controllers and performs some utility functions
type Utils struct {
	AllowedAnnotations []string
	CurrentNsPolicy    types.ListPolicy
}

// ShouldWatchNs returns true a namespace should be watched according to the
// provided labels and the list policy currently implemented.
func (c *Utils) ShouldWatchNs(labels map[string]string) (watch bool) {
	switch c.CurrentNsPolicy {
	case types.AllowList:
		if _, exists := labels[types.AllowedKey]; exists {
			watch = true
		}
	case types.BlockList:
		if _, exists := labels[types.BlockedKey]; !exists {
			watch = true
		}
	}

	return
}

// FilterAnnotations takes a map of annotations and returnes a new one
// stripped from the ones that should not be registered on the service
// registry.
func (c *Utils) FilterAnnotations(annotations map[string]string) map[string]string {
	if len(annotations) == 0 {
		return map[string]string{}
	}
	if len(c.AllowedAnnotations) == 0 {
		return map[string]string{}
	}

	allowedAnns := map[string]bool{}
	for _, val := range c.AllowedAnnotations {
		allowedAnns[val] = true
	}

	if _, exists := allowedAnns["*/*"]; exists {
		return annotations
	}

	filtered := map[string]string{}
	for key, val := range annotations {

		// Check this key specifically
		if _, exists := allowedAnns[key]; exists {
			filtered[key] = val
			continue
		}

		prefixName := strings.Split(key, "/")
		if len(prefixName) != 2 {
			// This key is not in prefix/name format
			continue
		}

		prefixWildcard := fmt.Sprintf("%s/*", prefixName[0])
		if _, exists := allowedAnns[prefixWildcard]; exists {
			filtered[key] = val
			continue
		}

		wildcardName := fmt.Sprintf("*/%s", prefixName[1])
		if _, exists := allowedAnns[wildcardName]; exists {
			filtered[key] = val
		}
	}

	return filtered
}
