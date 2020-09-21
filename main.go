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
	"flag"
	"fmt"
	"os"

	sd "github.com/CloudNativeSDWAN/cnwan-operator/pkg/servregistry/gcloud/servicedirectory"

	sr "github.com/CloudNativeSDWAN/cnwan-operator/pkg/servregistry"

	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/CloudNativeSDWAN/cnwan-operator/controllers"
	"github.com/CloudNativeSDWAN/cnwan-operator/types"
	// +kubebuilder:scaffold:imports
)

const (
	opKey = "owner"
	opVal = "cnwan-operator"
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
	var metricsAddr string
	var enableLeaderElection bool
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.Parse()
	ctx, canc := context.WithCancel(context.Background())
	defer canc()

	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	//--------------------------------------
	// Load the settings
	//--------------------------------------

	viper.SetConfigName("settings")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./settings/")
	err := viper.ReadInConfig()
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err := validateSettings(); err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Load the allowed annotations and put into a map, for better
	// check afterwards
	annotations := viper.GetStringSlice(types.AllowedAnnotations)
	allowedAnnotations := map[string]bool{}
	for _, ann := range annotations {
		allowedAnnotations[ann] = true
	}
	viper.Set(types.AllowedAnnotationsMap, allowedAnnotations)

	// Create a handler for gcp service directory
	credsPath := "./credentials/gcloud-credentials.json"
	sdHandler, err := sd.NewHandler(ctx, viper.GetString(types.SDProject), viper.GetString(types.SDDefaultRegion), credsPath, 30)
	if err != nil {
		setupLog.Error(err, "fatal error encountered")
		os.Exit(1)
	}

	srBroker, err := sr.NewBroker(sdHandler, opKey, opVal)
	if err != nil {
		setupLog.Error(err, "fatal error encountered")
		os.Exit(1)
	}

	//--------------------------------------
	// Init manager
	//--------------------------------------

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		Port:               9443,
		LeaderElection:     enableLeaderElection,
		LeaderElectionID:   "ebb9afc0.operator.cnwan.io",
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

func validateSettings() error {
	if len(viper.GetString(types.NamespaceListPolicy)) == 0 {
		viper.Set(types.NamespaceListPolicy, types.AllowList)
	}

	if len(viper.GetString(types.SDDefaultRegion)) == 0 {
		return fmt.Errorf("%s", "fatal: service directory region not provided")
	}

	if len(viper.GetString(types.SDProject)) == 0 {
		return fmt.Errorf("%s", "fatal: service directory project name not provided")
	}

	return nil
}
