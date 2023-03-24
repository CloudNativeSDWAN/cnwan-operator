// Copyright © 2020, 2021, 2022 Cisco
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

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/CloudNativeSDWAN/cnwan-operator/internal/types"
	"github.com/CloudNativeSDWAN/cnwan-operator/internal/utils"
	"github.com/CloudNativeSDWAN/cnwan-operator/pkg/cluster"
	"github.com/CloudNativeSDWAN/cnwan-operator/pkg/controllers"
	"github.com/CloudNativeSDWAN/cnwan-operator/pkg/serviceregistry"
	serego "github.com/CloudNativeSDWAN/serego/api/core"
	"github.com/CloudNativeSDWAN/serego/api/options/wrapper"
	"github.com/rs/zerolog"
	"gopkg.in/yaml.v3"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	// +kubebuilder:scaffold:imports
)

const (
	opKey                string = "owner"
	opVal                string = "cnwan-operator"
	defaultSettingsPath  string = "./settings/settings.yaml"
	defaultSdServAccPath string = "./credentials/gcloud-credentials.json"
	defaultTimeout       int    = 30
	defaultNsName        string = "cnwan-operator-system"

	// Exit codes
	Success int = iota
	CannotGetConfigmap
	CannotUnmarshalConfigmap
	SettingsValidationError
	CannotEstablishConnectionToEtcd
	CannotGetServiceDirectoryClient
	InvalidServiceDirectorySettings
	CannotGetCloudMapClient
	InvalidCloudMapSettings
	CannotGetBroker
	CannotGetControllerManager
	CannotCreateServiceController
	CannotCreateNamespaceController
	CannotRunControllerManager
)

// var (
// 	// scheme   = k8sruntime.NewScheme()
// 	setupLog = ctrl.Log.WithName("setup")
// )

// func init() {
// 	_ = clientgoscheme.AddToScheme(scheme)

// 	_ = corev1.AddToScheme(scheme)
// 	// +kubebuilder:scaffold:scheme
// }

var log zerolog.Logger

func main() {
	log = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).With().Timestamp().Logger()

	if code, err := run(); err != nil {
		log.Err(err).Msg("error occurred")
		os.Exit(code)
	}
}

func run() (int, error) {
	//--------------------------------------
	// Inits and defaults
	//--------------------------------------

	ctx, canc := context.WithCancel(context.Background())
	defer canc()

	nsName := os.Getenv("CNWAN_OPERATOR_NAMESPACE")
	if len(nsName) == 0 {
		log.Info().
			Str("default", defaultNsName).
			Msg("CNWAN_OPERATOR_NAMESPACE environment variable does not " +
				"exist: using default value")
		nsName = defaultNsName
	}

	//--------------------------------------
	// Load and parse settings
	//--------------------------------------

	var settings *types.Settings
	{
		settingsByte, err := cluster.GetOperatorSettingsConfigMap(ctx)
		if err != nil {
			return CannotGetConfigmap, fmt.Errorf("unable to retrieve settings from configmap: %w", err)
		}
		log.Info().Msg("settings file loaded successfully")

		var _settings *types.Settings
		if err := yaml.Unmarshal(settingsByte, &_settings); err != nil {
			return CannotUnmarshalConfigmap, fmt.Errorf("cannot unmarshal settings: %w", err)
		}

		settings, err = utils.ParseAndValidateSettings(_settings)
		if err != nil {
			return SettingsValidationError, fmt.Errorf("invalid settings provided: %w", err)
		}
	}
	log.Info().Msg("settings parsed successfully")

	persistentMeta := map[string]string{
		"owner": "cnwan-operator",
	}
	if settings.CloudMetadata != nil {
		// No need to check for network and subnetwork nil as it was already
		// validate previously.
		netCfg, err := getNetworkCfg(settings.CloudMetadata.Network, settings.CloudMetadata.SubNetwork)
		if err != nil {
			log.Err(err).Msg("could not get cloud network information, skipping...")
		} else {
			log.Info().
				Str("cnwan.io/network", netCfg.NetworkName).
				Str("cnwan.io/sub-network", netCfg.SubNetworkName).
				Msg("got network configuration")
			if runningIn := cluster.WhereAmIRunning(); runningIn != cluster.UnknownCluster {
				persistentMeta["cnwan.io/platform"] = string(runningIn)
			}
			if netCfg.NetworkName != "" {
				persistentMeta["cnwan.io/network"] = netCfg.NetworkName
			}
			if netCfg.NetworkName != "" {
				persistentMeta["cnwan.io/sub-network"] = netCfg.SubNetworkName
			}
		}
	}

	//--------------------------------------
	// Get the service registry
	//--------------------------------------

	var seregoClient *serego.ServiceRegistry

	switch {

	// Etcd
	case settings.ServiceRegistrySettings.EtcdSettings != nil:
		log.Info().Msg("using etcd")
		cli, err := getEtcdClient(settings.EtcdSettings)
		if err != nil {
			return CannotEstablishConnectionToEtcd, fmt.Errorf("cannot establish connection to etcd: %w", err)
		}
		defer cli.Close()

		seregoClient, err = serego.NewServiceRegistryFromEtcd(cli)
		if err != nil {
			return CannotEstablishConnectionToEtcd, fmt.Errorf("cannot establish connection to etcd: %w", err)
		}

		// Service directory
	case settings.ServiceRegistrySettings.ServiceDirectorySettings != nil:
		log.Info().Msg("using Service Directory")
		cli, err := getGSDClient(ctx)
		if err != nil {
			return CannotGetServiceDirectoryClient, fmt.Errorf("cannot get service directory client: %w", err)
		}
		defer cli.Close()

		sdSettings, err := parseAndResetGSDSettings(settings.ServiceRegistrySettings.ServiceDirectorySettings)
		if err != nil {
			return InvalidServiceDirectorySettings, fmt.Errorf("invalid service directory: %w", err)
		}

		seregoClient, err = serego.NewServiceRegistryFromServiceDirectory(cli,
			wrapper.WithProjectID(sdSettings.ProjectID),
			wrapper.WithRegion(sdSettings.DefaultRegion))
		if err != nil {
			return CannotGetServiceDirectoryClient, fmt.Errorf("cannot get service directory client: %w", err)
		}

		// Cloud Map
	case settings.ServiceRegistrySettings.CloudMapSettings != nil:
		log.Info().Msg("using Cloud Map")
		cmSettings, err := parseAndResetAWSCloudMapSettings(settings.CloudMapSettings)
		if err != nil {
			return InvalidCloudMapSettings, fmt.Errorf("invalid cloud map settings: %w", err)
		}

		cli, err := getAWSClient(ctx, &cmSettings.DefaultRegion)
		if err != nil {
			return CannotGetCloudMapClient, fmt.Errorf("cannot get cloud map client: %w", err)
		}

		seregoClient, _ = serego.NewServiceRegistryFromCloudMap(cli)
	}

	manager, err := controllers.NewManager("")
	if err != nil {
		log.Err(err).Msg("cannot create manager")
		return 1, err
	}

	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM)
	watchCtx, watchCanc := context.WithCancel(ctx)
	exitChan := make(chan struct{})
	go func() {
		defer close(exitChan)
		eventsChan := make(chan *serviceregistry.Event, 100)
		eventHandler := serviceregistry.NewEventHandler(seregoClient, persistentMeta, log)

		go func() {
			eventHandler.WatchForEvents(watchCtx, eventsChan)
			close(exitChan)
		}()

		controllers.NewNamespaceController(manager, &controllers.ControllerOptions{
			EventsChan:         eventsChan,
			ServiceAnnotations: settings.Service.Annotations,
		}, log)
		controllers.NewServiceController(manager, &controllers.ControllerOptions{
			EventsChan:         eventsChan,
			ServiceAnnotations: settings.Service.Annotations,
		}, log)

		manager.Start(ctx)
		log.Info().Msg("closing")
	}()

	<-stopChan
	watchCanc()
	<-exitChan

	log.Info().Msg("goodbye!")
	return Success, nil
}
