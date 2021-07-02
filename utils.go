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

package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	sd "cloud.google.com/go/servicedirectory/apiv1"
	"github.com/CloudNativeSDWAN/cnwan-operator/internal/types"
	"github.com/CloudNativeSDWAN/cnwan-operator/pkg/cluster"
	"go.etcd.io/etcd/clientv3"
	"google.golang.org/api/option"
)

func getNetworkCfg(network, subnetwork *string) (netCfg *cluster.NetworkConfiguration, err error) {
	netCfg = &cluster.NetworkConfiguration{}
	if network != nil {
		netCfg.NetworkName = *network
	}
	if subnetwork != nil {
		netCfg.SubNetworkName = *subnetwork
	}

	if strings.ToLower(netCfg.NetworkName) == "auto" || strings.ToLower(netCfg.SubNetworkName) == "auto" {
		var res *cluster.NetworkConfiguration
		runningIn := cluster.WhereAmIRunning()
		if runningIn == cluster.UnknownCluster {
			return nil, fmt.Errorf("could not get information about the managed cluster: unsupported or no permissions to do so")
		}

		if runningIn == cluster.GKECluster {
			sa, err := cluster.GetGoogleServiceAccountSecret(context.Background())
			if err != nil {
				return nil, err
			}

			res, err = cluster.GetNetworkFromGKE(context.Background(), option.WithCredentialsJSON(sa))
			if err != nil {
				return nil, err
			}
		}

		// TODO: implement EKS on future versions. Code is ready but just not
		// included in this iteration.

		if strings.ToLower(netCfg.NetworkName) == "auto" {
			netCfg.NetworkName = res.NetworkName
		}
		if strings.ToLower(netCfg.SubNetworkName) == "auto" {
			netCfg.SubNetworkName = res.SubNetworkName
		}
	}

	return
}

func getGSDClient(ctx context.Context) (*sd.RegistrationClient, error) {
	// TODO: next versions will have a flag parsing system. Therefore this will
	// need a change in case service account is provided somewhere else.
	saBytes, err := cluster.GetGoogleServiceAccountSecret(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not load google service account secret: %s", err)
	}

	cli, err := sd.NewRegistrationClient(ctx, option.WithCredentialsJSON(saBytes))
	if err != nil {
		return nil, fmt.Errorf("could not get start service directory client: %s", err)
	}

	return cli, err
}

func parseAndResetGSDSettings(gcSettings *types.ServiceDirectorySettings) (*types.ServiceDirectorySettings, error) {
	newSettings := &types.ServiceDirectorySettings{
		DefaultRegion: "",
		ProjectID:     "",
	}

	if gcSettings != nil && gcSettings.DefaultRegion != "" {
		newSettings.DefaultRegion = gcSettings.DefaultRegion
		setupLog.Info("using region defined in settings", "region", gcSettings.DefaultRegion)
	}

	if gcSettings != nil && gcSettings.ProjectID != "" {
		newSettings.ProjectID = gcSettings.ProjectID
		setupLog.Info("using project ID defined in settings", "project-id", gcSettings.ProjectID)
	}

	if newSettings.DefaultRegion != "" && newSettings.ProjectID != "" {
		return newSettings, nil
	}

	setupLog.Info("attempting to retrieve some data from Google Cloud...")
	if cluster.WhereAmIRunning() != cluster.GKECluster {
		return nil, fmt.Errorf("could not load data from Google Cloud: either platform is not GKE or there are no permissions to do so")
	}

	if newSettings.DefaultRegion == "" {
		_defRegion, err := cluster.GetGCPRegion()
		if err != nil {
			return nil, fmt.Errorf("could not get region from GCP: %s", err)
		}
		newSettings.DefaultRegion = *_defRegion
		setupLog.Info("retrieved region from GCP", "region", newSettings.DefaultRegion)
	}

	if newSettings.ProjectID == "" {
		_projectID, err := cluster.GetGCPProjectID()
		if err != nil {
			return nil, fmt.Errorf("could not get project ID from GCP: %s", err)
		}
		newSettings.ProjectID = *_projectID
		setupLog.Info("retrieved project ID from GCP", "project ID", newSettings.ProjectID)
	}

	return newSettings, nil
}

func getEtcdClient(settings *types.EtcdSettings) (*clientv3.Client, error) {
	endps := []string{}

	for _, endp := range settings.Endpoints {
		endps = append(endps, fmt.Sprintf("%s:%d", endp.Host, *endp.Port))
	}
	cfg := clientv3.Config{
		Endpoints: endps,
	}

	if settings.Authentication == types.EtcdAuthWithNothing {
		return clientv3.New(cfg)
	}

	ctx, canc := context.WithTimeout(context.Background(), time.Duration(15)*time.Second)
	defer canc()

	if settings.Authentication == types.EtcdAuthWithUsernamePassw {
		user, pass, err := cluster.GetEtcdCredentialsSecret(ctx)
		if err != nil {
			return nil, err
		}

		if len(user) > 0 {
			cfg.Username = string(user)
		}
		if len(pass) > 0 {
			cfg.Password = string(pass)
		}

		cfg.Endpoints = endps
		return clientv3.New(cfg)
	}

	// TODO: support for TLS: if authentication is through TLS if Username and Password are both nil, then look
	// for the secrets containing the client's certificate and and key.
	return nil, fmt.Errorf("unsupported etcd authentication method")
}
