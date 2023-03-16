// Copyright © 2021 - 2023 Cisco
//
// SPDX-License-Identifier: Apache-2.0
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
	"net"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

// filterAnnotations is used to remove annotations that should be ignored
// by the operator
func filterAnnotations(currentAnnotations map[string]string, filter []string) map[string]string {
	filterMap := map[string]bool{}
	for _, ann := range filter {
		filterMap[ann] = true
	}

	if _, exists := filterMap["*/*"]; exists {
		return currentAnnotations
	}

	filtered := map[string]string{}
	for key, val := range currentAnnotations {

		// Check this key specifically
		if _, exists := filterMap[key]; exists {
			filtered[key] = val
			continue
		}

		prefixName := strings.Split(key, "/")
		if len(prefixName) != 2 {
			// This key is not in prefix/name format
			continue
		}

		prefixWildcard := fmt.Sprintf("%s/*", prefixName[0])
		if _, exists := filterMap[prefixWildcard]; exists {
			filtered[key] = val
			continue
		}

		wildcardName := fmt.Sprintf("*/%s", prefixName[1])
		if _, exists := filterMap[wildcardName]; exists {
			filtered[key] = val
		}
	}

	return filtered
}

func shouldWatchLabel(labels map[string]string, watchAllByDefault bool) bool {
	switch labels[watchLabel] {
	case watchEnabledLabel:
		return true
	case watchDisabledLabel:
		return false
	default:
		return watchAllByDefault
	}
}

func getIPsFromService(service *corev1.Service) ([]string, error) {
	ips := []string{}
	ips = append(ips, service.Spec.ExternalIPs...)

	// Get data from load balancers
	for _, ing := range service.Status.LoadBalancer.Ingress {
		if ing.IP != "" {
			ips = append(ips, ing.IP)
		}

		if ing.Hostname != "" {
			if resolvedIPs, err := net.LookupHost(ing.Hostname); err != nil {
				return nil, err
			} else {
				ips = append(ips, resolvedIPs...)
			}
		}
		ips = append(ips, ing.IP)
	}

	return ips, nil
}
