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

	"github.com/CloudNativeSDWAN/cnwan-operator/controllers"
	"github.com/CloudNativeSDWAN/cnwan-operator/internal/types"
	"github.com/CloudNativeSDWAN/cnwan-operator/internal/utils"
	"github.com/CloudNativeSDWAN/cnwan-operator/pkg/cluster"
	sr "github.com/CloudNativeSDWAN/cnwan-operator/pkg/servregistry"
	"github.com/CloudNativeSDWAN/cnwan-operator/pkg/servregistry/aws/cloudmap"
	"github.com/CloudNativeSDWAN/cnwan-operator/pkg/servregistry/etcd"
	sd "github.com/CloudNativeSDWAN/cnwan-operator/pkg/servregistry/gcloud/servicedirectory"
	clientv3 "go.etcd.io/etcd/client/v3"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
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

var (
	scheme   = k8sruntime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)

	_ = corev1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func main() {
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	if code, err := run(); err != nil {
		setupLog.Error(err, "error occurred")
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
		setupLog.Info("CNWAN_OPERATOR_NAMESPACE environment variable does not exist: using default value", "default", defaultNsName)
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
		setupLog.Info("settings file loaded successfully")

		var _settings *types.Settings
		if err := yaml.Unmarshal(settingsByte, &_settings); err != nil {
			return CannotUnmarshalConfigmap, fmt.Errorf("cannot unmarshal settings: %w", err)
		}

		settings, err = utils.ParseAndValidateSettings(_settings)
		if err != nil {
			return SettingsValidationError, fmt.Errorf("invalid settings provided: %w", err)
		}
	}
	setupLog.Info("settings parsed successfully")

	persistentMeta := []sr.MetadataPair{}
	if settings.CloudMetadata != nil {
		// No need to check for network and subnetwork nil as it was already
		// validate previously.
		netCfg, err := getNetworkCfg(settings.CloudMetadata.Network, settings.CloudMetadata.SubNetwork)
		if err != nil {
			setupLog.Error(err, "could not get cloud network information, skipping...")
		} else {
			setupLog.Info("got network configuration", "cnwan.io/network", netCfg.NetworkName, "cnwan.io/sub-network", netCfg.SubNetworkName)
			if runningIn := cluster.WhereAmIRunning(); runningIn != cluster.UnknownCluster {
				persistentMeta = append(persistentMeta, sr.MetadataPair{Key: "cnwan.io/platform", Value: string(runningIn)})
			}
			if netCfg.NetworkName != "" {
				persistentMeta = append(persistentMeta, sr.MetadataPair{Key: "cnwan.io/network", Value: netCfg.NetworkName})
			}
			if netCfg.NetworkName != "" {
				persistentMeta = append(persistentMeta, sr.MetadataPair{Key: "cnwan.io/sub-network", Value: netCfg.SubNetworkName})
			}
		}
	}

	//--------------------------------------
	// Get the service registry
	//--------------------------------------

	var etcdClient *clientv3.Client
	var servreg sr.ServiceRegistry

	if settings.ServiceRegistrySettings.EtcdSettings != nil {
		setupLog.Info("using etcd as a service registry...")
		_cli, err := getEtcdClient(settings.EtcdSettings)
		if err != nil {
			return CannotEstablishConnectionToEtcd, fmt.Errorf("cannot establish connection to etcd: %w", err)
		}

		etcdClient = _cli
		defer etcdClient.Close()
		servreg = etcd.NewServiceRegistryWithEtcd(ctx, etcdClient, settings.EtcdSettings.Prefix)
	}

	if settings.ServiceRegistrySettings.ServiceDirectorySettings != nil {
		setupLog.Info("using gcloud service directory...")
		cli, err := getGSDClient(context.Background())
		if err != nil {
			return CannotGetServiceDirectoryClient, fmt.Errorf("cannot get service directory client: %w", err)
		}
		defer cli.Close()

		sdSettings, err := parseAndResetGSDSettings(settings.ServiceRegistrySettings.ServiceDirectorySettings)
		if err != nil {
			return InvalidServiceDirectorySettings, fmt.Errorf("invalid service directory: %w", err)
		}

		servreg = &sd.Handler{
			ProjectID:     sdSettings.ProjectID,
			DefaultRegion: sdSettings.DefaultRegion,
			Log:           setupLog.WithName("ServiceDirectory"),
			Context:       ctx,
			Client:        cli,
		}
	}

	if settings.ServiceRegistrySettings.CloudMapSettings != nil {
		setupLog.Info("using aws cloud map...")

		cmSettings, err := parseAndResetAWSCloudMapSettings(settings.CloudMapSettings)
		if err != nil {
			return InvalidCloudMapSettings, fmt.Errorf("invalid cloud map settings: %w", err)
		}

		cli, err := getAWSClient(context.Background(), &cmSettings.DefaultRegion)
		if err != nil {
			return CannotGetCloudMapClient, fmt.Errorf("cannot get cloud map client: %w", err)
		}

		servreg = cloudmap.NewHandler(ctx, cli, setupLog)
	}

	srBroker, err := sr.NewBroker(servreg, sr.MetadataPair{Key: opKey, Value: opVal}, persistentMeta...)
	if err != nil {
		return CannotGetBroker, fmt.Errorf("cannot get service registry broker: %w", err)
	}

	//--------------------------------------
	// Init manager
	//--------------------------------------

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		LeaderElection:     false,
		MetricsBindAddress: "0",
	})
	if err != nil {
		return CannotGetControllerManager, fmt.Errorf("cannot create controller manager: %w", err)
	}

	if err = (&controllers.ServiceReconciler{
		Client:                   mgr.GetClient(),
		Log:                      ctrl.Log.WithName("controllers").WithName("Service"),
		Scheme:                   mgr.GetScheme(),
		ServRegBroker:            srBroker,
		WatchNamespacesByDefault: settings.WatchNamespacesByDefault,
		AllowedAnnotations:       settings.Service.Annotations,
	}).SetupWithManager(mgr); err != nil {
		return CannotCreateServiceController, fmt.Errorf("cannot create service controller: %w", err)
	}

	if err = (&controllers.NamespaceReconciler{
		Client:                   mgr.GetClient(),
		Log:                      ctrl.Log.WithName("controllers").WithName("Namespace"),
		Scheme:                   mgr.GetScheme(),
		ServRegBroker:            srBroker,
		WatchNamespacesByDefault: settings.WatchNamespacesByDefault,
		AllowedAnnotations:       settings.Service.Annotations,
	}).SetupWithManager(mgr); err != nil {
		return CannotCreateNamespaceController, fmt.Errorf("cannot create namespace controller: %w", err)
	}
	// +kubebuilder:scaffold:builder

	setupLog.Info("starting controller manager...")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		return CannotRunControllerManager, fmt.Errorf("cannot run controller manager: %w", err)
	}

	return Success, nil
}
