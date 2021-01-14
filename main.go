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

package main

import (
	"context"
	"io/ioutil"
	"os"

	"github.com/CloudNativeSDWAN/cnwan-operator/internal/types"
	"github.com/CloudNativeSDWAN/cnwan-operator/internal/utils"
	sr "github.com/CloudNativeSDWAN/cnwan-operator/pkg/servregistry"
	sd "github.com/CloudNativeSDWAN/cnwan-operator/pkg/servregistry/gcloud/servicedirectory"
	"gopkg.in/yaml.v3"

	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/CloudNativeSDWAN/cnwan-operator/controllers"
	// +kubebuilder:scaffold:imports
)

const (
	opKey                string = "owner"
	opVal                string = "cnwan-operator"
	defaultSettingsPath  string = "./settings/settings.yaml"
	defaultSdServAccPath string = "./credentials/gcloud-credentials.json"
	defaultTimeout       int    = 30
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)

	_ = corev1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func main() {
	//--------------------------------------
	// Inits and defaults
	//--------------------------------------
	ctx, canc := context.WithCancel(context.Background())
	defer canc()

	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	settingsPath := getSettingsPath()

	//--------------------------------------
	// Load the settings
	//--------------------------------------

	settings, err := getSettings(settingsPath)
	if err != nil {
		setupLog.Error(err, "error while unmarshaling settings")
		os.Exit(1)
	}
	setupLog.Info("settings file loaded successfully")

	settings, err = utils.ParseAndValidateSettings(settings)
	if err != nil {
		setupLog.Error(err, "error while unmarshaling options")
		os.Exit(1)
	}
	setupLog.Info("settings parsed successfully")

	viper.SetConfigFile(settingsPath)
	if err := viper.ReadInConfig(); err != nil {
		setupLog.Error(err, "error storing settings")
		os.Exit(1)
	}

	// Load the allowed annotations and put into a map, for better
	// check afterwards
	annotations := settings.Service.Annotations
	allowedAnnotations := map[string]bool{}
	for _, ann := range annotations {
		allowedAnnotations[ann] = true
	}
	viper.Set(types.AllowedAnnotationsMap, allowedAnnotations)

	// Create a handler for gcp service directory
	servreg, err := getServiceDirectoryHandler(ctx, settings.ServiceRegistrySettings.ProjectID, settings.ServiceRegistrySettings.DefaultRegion)
	if err != nil {
		setupLog.Error(err, "fatal error encountered")
		os.Exit(1)
	}
	srBroker, err := sr.NewBroker(servreg, opKey, opVal)
	if err != nil {
		setupLog.Error(err, "fatal error encountered")
		os.Exit(1)
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
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controllers.ServiceReconciler{
		Client:        mgr.GetClient(),
		Log:           ctrl.Log.WithName("controllers").WithName("Service"),
		Scheme:        mgr.GetScheme(),
		ServRegBroker: srBroker,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Service")
		os.Exit(1)
	}
	if err = (&controllers.NamespaceReconciler{
		Client:        mgr.GetClient(),
		Log:           ctrl.Log.WithName("controllers").WithName("Namespace"),
		Scheme:        mgr.GetScheme(),
		ServRegBroker: srBroker,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Namespace")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func getServiceDirectoryHandler(ctx context.Context, projectID, defaultRegion string) (sr.ServiceRegistry, error) {
	// TODO: this will be heavily improved in future versions

	credsPath := defaultSdServAccPath

	// is specified on env?
	if fromEnv := os.Getenv("CNWAN_OPERATOR_SETTINGS_PATH"); len(fromEnv) > 0 {
		credsPath = fromEnv
	}

	sdHandler, err := sd.NewHandler(ctx, projectID, defaultRegion, credsPath, defaultTimeout)
	if err != nil {
		return nil, err
	}

	return sdHandler, nil
}

func getSettingsPath() string {
	args := os.Args

	// is specified as first argument?
	if len(args) > 1 {
		return args[1]
	}

	// is specified on env?
	if fromEnv := os.Getenv("CNWAN_OPERATOR_SETTINGS_PATH"); len(fromEnv) > 0 {
		return fromEnv
	}

	// last resort: just try to load it from a default path...
	return defaultSettingsPath
}

func getSettings(fileName string) (*types.Settings, error) {
	file, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	var settings types.Settings
	if err := yaml.Unmarshal(file, &settings); err != nil {
		return nil, err
	}

	return &settings, nil
}
