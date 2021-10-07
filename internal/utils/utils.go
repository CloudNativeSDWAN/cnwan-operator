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

	"github.com/CloudNativeSDWAN/cnwan-operator/internal/types"
	"go.uber.org/zap/zapcore"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	log = zap.New(zap.UseDevMode(false))
)

// ParseAndValidateSettings parses the settings and validates them.
//
// In case of any errors, the settings returned is nil and the error
// occurred is returned.
// TODO: remove this in favor a independent ConfigMaps/Secrets validation.
func ParseAndValidateSettings(settings *types.Settings) (*types.Settings, error) {
	if settings == nil {
		return nil, fmt.Errorf("no settings provided")
	}

	finalSettings := &types.Settings{WatchNamespacesByDefault: settings.WatchNamespacesByDefault}
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

	if len(settings.Service.Annotations) == 0 {
		log.V(int(zapcore.WarnLevel)).Info("no allowed annotations provided: no service will be registered")
	}
	finalSettings.Service = settings.Service

	if settings.ServiceRegistrySettings == nil {
		return nil, fmt.Errorf("no service registry provided")
	}

	finalSettings.ServiceRegistrySettings = &types.ServiceRegistrySettings{}

	// Make sure only one service registry is provided.
	// TODO: this won't be necessary in future anymore as the CLI will take
	// care of it.
	srs := func() int {
		n := 0

		if settings.EtcdSettings != nil {
			n++
		}

		if settings.ServiceDirectorySettings != nil {
			n++
		}

		if settings.CloudMapSettings != nil {
			n++
		}

		return n
	}()

	if srs == 0 {
		return nil, fmt.Errorf("no service registry provided")
	}

	if srs > 1 {
		return nil, fmt.Errorf("UNSUPPORTED: multiple service registries have been provided")
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

	if settings.ServiceDirectorySettings != nil {
		// service directory settings is parsed on another function now.
		finalSettings.ServiceDirectorySettings = settings.ServiceDirectorySettings
	}

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
