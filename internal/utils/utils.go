// Copyright © 2020, 2021 Cisco
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

	"github.com/CloudNativeSDWAN/cnwan-operator/internal/types"
	"github.com/spf13/viper"
	"go.uber.org/zap/zapcore"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	log = zap.New(zap.UseDevMode(false))
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

// ParseAndValidateSettings parses the settings and validates them.
//
// In case of any errors, the settings returned is nil and the error
// occurred is returned.
// TODO: remove this in favor a independent ConfigMaps/Secrets validation.
func ParseAndValidateSettings(settings *types.Settings) (*types.Settings, error) {
	if settings == nil {
		return nil, fmt.Errorf("no settings provided")
	}

	finalSettings := &types.Settings{}
	if settings.CloudMetadata != nil {
		clCfg := settings.CloudMetadata
		finalCfg := &types.CloudMetadata{}

		if clCfg.Network != nil && *clCfg.Network != "" {
			finalCfg.Network = clCfg.Network
		}
		if clCfg.SubNetwork != nil && *clCfg.SubNetwork != "" {
			finalCfg.SubNetwork = clCfg.SubNetwork
		}

		if finalCfg.Network != nil || finalCfg.SubNetwork != nil {
			finalSettings.CloudMetadata = finalCfg
		}
	}

	if settings.Namespace.ListPolicy != types.AllowList && settings.Namespace.ListPolicy != types.BlockList {
		// Probably we could revert to using a default value here, but I think
		// it's better not to confuse the user with unexpected behaviors and
		// just return an error.
		return nil, fmt.Errorf("namespace list policy is neither AllowList nor BlockList")
	}
	finalSettings.Namespace = settings.Namespace

	if len(settings.Service.Annotations) == 0 {
		log.V(int(zapcore.WarnLevel)).Info("no allowed annotations provided: no service will be registered")
	}
	finalSettings.Service = settings.Service

	if settings.ServiceRegistrySettings == nil {
		return nil, fmt.Errorf("no service registry provided")
	}

	finalSettings.ServiceRegistrySettings = &types.ServiceRegistrySettings{}

	// Only one service registry can be chosen at this time

	// TODO: remove this in v0.6.0
	if settings.Gcloud != nil {
		if settings.Gcloud.ServiceDirectory != nil && settings.ServiceDirectorySettings == nil {
			// Convert the deprecated service directory settings into the new structure,
			// but only if the new one doesn't already exist.
			log.V(int(zapcore.WarnLevel)).Info(`DEPRECATED: current service directory settings is under gcloud field.
				This is deprecated and will be removed on v0.6.0.
				Please place it under service registry as defined in the documentation.`)

			sd := settings.Gcloud.ServiceDirectory
			settings.ServiceDirectorySettings = &types.ServiceDirectorySettings{
				DefaultRegion: sd.DefaultRegion,
				ProjectID:     sd.ProjectName,
			}
		}
	}

	if settings.EtcdSettings == nil && settings.ServiceDirectorySettings == nil {
		// Both are nil
		return nil, fmt.Errorf("no service registry provided")
	}

	// Just to display the warning
	if settings.EtcdSettings != nil && settings.ServiceDirectorySettings != nil {
		log.V(int(zapcore.WarnLevel)).Info("UNSUPPORTED: multiple service registries are not supported yet. Only etcd will be used.")
	}

	if settings.EtcdSettings != nil {
		parsedSettings, err := parseEtcdSettings(settings.EtcdSettings)
		if err != nil {
			return nil, err
		}

		finalSettings.EtcdSettings = parsedSettings
		// Nothing else to check

		return finalSettings, nil
	}

	// service directory settings is parsed on another function now.
	finalSettings.ServiceDirectorySettings = settings.ServiceDirectorySettings
	settings = finalSettings

	return finalSettings, nil
}

func parseEtcdSettings(settings *types.EtcdSettings) (*types.EtcdSettings, error) {
	if len(settings.Endpoints) == 0 {
		return nil, fmt.Errorf("no etcd endpoints provided")
	}

	if settings.Authentication != types.EtcdAuthWithNothing &&
		settings.Authentication != types.EtcdAuthWithUsernamePassw &&
		settings.Authentication != types.EtcdAuthWithTLS {
		return nil, fmt.Errorf("unrecognized authentication method for etcd")
	}

	if settings.Authentication == types.EtcdAuthWithTLS {
		return nil, fmt.Errorf("etcd authentication with TLS is not supported yet")
	}

	finalSettings := &types.EtcdSettings{
		Authentication: settings.Authentication,
		Prefix:         settings.Prefix,
		Endpoints:      []*types.EtcdEndpoint{},
	}

	dups := map[string]int{}
	for i, endp := range settings.Endpoints {
		if len(endp.Host) == 0 {
			continue
		}

		// https://www.iana.org/assignments/service-names-port-numbers/service-names-port-numbers.txt
		port := 2379
		if endp.Port != nil {
			port = *settings.Endpoints[i].Port
		}

		if val, exists := dups[endp.Host]; exists && val == port {
			// skip this
			continue
		}

		newEndp := &types.EtcdEndpoint{
			Host: endp.Host,
			Port: &port,
		}
		finalSettings.Endpoints = append(finalSettings.Endpoints, newEndp)
		dups[endp.Host] = port
	}

	return finalSettings, nil
}
