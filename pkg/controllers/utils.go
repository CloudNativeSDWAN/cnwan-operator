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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"strings"

	serego "github.com/CloudNativeSDWAN/serego/api/core/types"
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

func getIPsFromService(service *corev1.Service) ([]string, error) {
	ipsMap := map[string]bool{}
	for _, externalIP := range service.Spec.ExternalIPs {
		ipsMap[externalIP] = true
	}

	// Get data from load balancers
	for _, ing := range service.Status.LoadBalancer.Ingress {
		if ing.IP != "" {
			ipsMap[ing.IP] = true
		}

		if ing.Hostname != "" {
			resolvedIPs, err := net.LookupHost(ing.Hostname)
			if err != nil {
				return nil, err
			}

			for _, resolvedIP := range resolvedIPs {
				ipsMap[resolvedIP] = true
			}
		}
	}

	ips := []string{}
	for ip := range ipsMap {
		ips = append(ips, ip)
	}
	return ips, nil
}

func checkNsLabels(labels map[string]string, watchAllByDefault bool) bool {
	switch labels[watchLabel] {
	case watchEnabledLabel:
		return true
	case watchDisabledLabel:
		return false
	default:
		return watchAllByDefault
	}
}

type checkServiceResult struct {
	passed      bool
	reason      string
	err         error
	annotations map[string]string
	ips         []string
	endpoints   []*serego.Endpoint
}

func checkService(service *corev1.Service, annotationsToKeep []string) (result checkServiceResult) {
	if service.Spec.Type != corev1.ServiceTypeLoadBalancer {
		result.reason = "not a LoadBalancer"
		return
	}

	annotations := filterAnnotations(service.Annotations, annotationsToKeep)
	if len(annotations) == 0 {
		result.reason = "no valid annotations"
		return
	}

	ips, err := getIPsFromService(service)
	if len(ips) == 0 {
		result.reason = "no valid hostnames/ips found"
		if err != nil {
			result.err = err
		}

		return
	}

	result = checkServiceResult{
		passed:      true,
		annotations: annotations,
		ips:         ips,
		endpoints:   []*serego.Endpoint{},
	}
	for _, port := range service.Spec.Ports {
		for _, ip := range ips {

			// Create an hashed name for this
			toBeHashed := fmt.Sprintf("%s:%d", ip, port.Port)
			h := sha256.New()
			h.Write([]byte(toBeHashed))
			hash := hex.EncodeToString(h.Sum(nil))

			result.endpoints = append(result.endpoints, &serego.Endpoint{
				Namespace: service.Namespace,
				Service:   service.Name,
				Name:      fmt.Sprintf("%s-%s", service.Name, hash[:10]),
				Address:   ip,
				Port:      port.Port,
				Metadata:  annotations,
			})
		}
	}

	return
}

func getEndpointsMapFromSlice(endpoints []*serego.Endpoint) map[string]*serego.Endpoint {
	epMap := map[string]*serego.Endpoint{}
	for _, ep := range endpoints {
		epMap[ep.Name] = ep
	}
	return epMap
}
