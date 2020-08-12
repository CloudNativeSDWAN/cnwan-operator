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
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/CloudNativeSDWAN/cnwan-operator/types"
	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
)

const (
	hashFormat string = "%s:%d"
	hashChars  int    = 10
)

func getIPsFromLoadBalancer(servSpec corev1.ServiceSpec, lb corev1.LoadBalancerStatus) (ips []string) {
	// Append external IPs
	ips = append(ips, servSpec.ExternalIPs...)

	// Get data from load balancers
	for _, ing := range lb.Ingress {
		ips = append(ips, ing.IP)
	}

	return
}

func getHashedName(name, ip string, port int32) string {
	fullName := fmt.Sprintf(hashFormat, ip, port)
	h := sha256.New()

	h.Write([]byte(fullName))
	hash := fmt.Sprintf("%x", h.Sum(nil))
	return fmt.Sprintf("%s-%s", name, hash[:hashChars])
}

// GetSnapshot returns a snapshot of the current service
func GetSnapshot(service *corev1.Service) types.ServiceSnapshot {
	snap := types.ServiceSnapshot{
		Name:      service.Name,
		Namespace: service.Namespace,
		Metadata:  FilterAnnotations(service.Annotations),
	}

	if !service.DeletionTimestamp.IsZero() {
		// If it is deleted, it means that it does not have *valid* endpoitns
		// anymore
		snap.Endpoints = map[string]types.EndpointSnapshot{}
		return snap
	}

	ips := getIPsFromLoadBalancer(service.Spec, service.Status.LoadBalancer)

	// Get the endpoints
	endpoints := map[string]types.EndpointSnapshot{}
	for _, port := range service.Spec.Ports {
		for _, ip := range ips {
			name := getHashedName(service.Name, ip, port.Port)
			endpoints[name] = types.EndpointSnapshot{
				Name:     name,
				Address:  ip,
				Port:     port.Port,
				Metadata: map[string]string{},
			}
		}
	}

	snap.Endpoints = endpoints
	return snap
}

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
