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

package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"time"

	"github.com/CloudNativeSDWAN/cnwan-operator/controllers"
	"github.com/CloudNativeSDWAN/cnwan-operator/internal/types"
	"github.com/CloudNativeSDWAN/cnwan-operator/internal/utils"
	"github.com/CloudNativeSDWAN/cnwan-operator/pkg/cluster"
	sr "github.com/CloudNativeSDWAN/cnwan-operator/pkg/servregistry"
	"github.com/CloudNativeSDWAN/cnwan-operator/pkg/servregistry/etcd"
	"github.com/spf13/viper"
	"go.etcd.io/etcd/clientv3"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
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
	// TODO: on next version, this main will be completely changed with a
	// better return code and exiting mechanism. Right now is fine but
	// too cluttered.

	//--------------------------------------
	// Inits and defaults
	//--------------------------------------
	returnCode := 0
	defer os.Exit(returnCode)

	ctx, canc := context.WithCancel(context.Background())
	defer canc()

	var etcdClient *clientv3.Client
	var servreg sr.ServiceRegistry
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	nsName := os.Getenv("CNWAN_OPERATOR_NAMESPACE")
	if len(nsName) == 0 {
		setupLog.Info("CNWAN_OPERATOR_NAMESPACE environment variable does not exist: using default value", "default", defaultNsName)
		nsName = defaultNsName
	}

	settingsPath := getSettingsPath()

	//--------------------------------------
	// Load the settings
	//--------------------------------------

	settings, err := getSettings(settingsPath)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		returnCode = 1
		runtime.Goexit()
	}
	setupLog.Info("settings file loaded successfully")

	settings, err = utils.ParseAndValidateSettings(settings)
	if err != nil {
		setupLog.Error(err, "error while unmarshaling options")
		returnCode = 2
		runtime.Goexit()
	}
	setupLog.Info("settings parsed successfully")

	viper.SetConfigFile(settingsPath)
	if err := viper.ReadInConfig(); err != nil {
		setupLog.Error(err, "error storing settings")
		returnCode = 3
		runtime.Goexit()
	}

	viper.Set(types.CurrentNamespace, nsName)

	// Load the allowed annotations and put into a map, for better
	// check afterwards
	annotations := settings.Service.Annotations
	allowedAnnotations := map[string]bool{}
	for _, ann := range annotations {
		allowedAnnotations[ann] = true
	}
	viper.Set(types.AllowedAnnotationsMap, allowedAnnotations)

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

	var servRegErr error
	if settings.ServiceRegistrySettings.EtcdSettings != nil {
		setupLog.Info("using etcd as a service registry...")
		_cli, err := getEtcdClient(settings.EtcdSettings)
		if err != nil {
			setupLog.Error(err, "error while establishing connection to the etcd cluster")
			returnCode = 4
			runtime.Goexit()
		}
		etcdClient = _cli
		defer etcdClient.Close()
		servreg, servRegErr = etcd.NewServiceRegistryWithEtcd(ctx, etcdClient, settings.EtcdSettings.Prefix)
	}
	if settings.ServiceRegistrySettings.ServiceDirectorySettings != nil {
		setupLog.Info("using gcloud service directory...")

		cli, err := getGSDClient(context.Background())
		if err != nil {
			setupLog.Error(err, "fatal error encountered")
			returnCode = 11
			runtime.Goexit()
		}
		defer cli.Close()

		sdSettings, err := parseAndResetGSDSettings(settings.ServiceRegistrySettings.ServiceDirectorySettings)
		if err != nil {
			setupLog.Error(err, "fatal error encountered")
			returnCode = 11
			runtime.Goexit()
		}

		// TODO: get new handler
		_ = sdSettings
	}

	if servRegErr != nil {
		setupLog.Error(err, "fatal error encountered")
		returnCode = 5
		runtime.Goexit()
	}

	srBroker, err := sr.NewBroker(servreg, sr.MetadataPair{Key: opKey, Value: opVal}, persistentMeta...)
	if err != nil {
		setupLog.Error(err, "fatal error encountered")
		returnCode = 6
		runtime.Goexit()
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
		returnCode = 7
		runtime.Goexit()
	}

	if err = (&controllers.ServiceReconciler{
		Client:        mgr.GetClient(),
		Log:           ctrl.Log.WithName("controllers").WithName("Service"),
		Scheme:        mgr.GetScheme(),
		ServRegBroker: srBroker,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Service")
		returnCode = 8
		runtime.Goexit()
	}
	if err = (&controllers.NamespaceReconciler{
		Client:        mgr.GetClient(),
		Log:           ctrl.Log.WithName("controllers").WithName("Namespace"),
		Scheme:        mgr.GetScheme(),
		ServRegBroker: srBroker,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Namespace")
		returnCode = 9
		runtime.Goexit()
	}
	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		returnCode = 10
		runtime.Goexit()
	}
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

	k8sconf := ctrl.GetConfigOrDie()
	clientset, err := kubernetes.NewForConfig(k8sconf)
	if err != nil {
		return nil, err
	}

	if settings.Authentication == types.EtcdAuthWithUsernamePassw {
		cfg, err := getEtcdConfWithCredentials(clientset)
		if err != nil {
			return nil, err
		}

		cfg.Endpoints = endps
		return clientv3.New(*cfg)
	}

	// TODO: support for TLS: if authentication is through TLS if Username and Password are both nil, then look
	// for the secrets containing the client's certificate and and key.
	return nil, fmt.Errorf("unsupported etcd authentication method")
}

func getEtcdConfWithCredentials(clientset *kubernetes.Clientset) (*clientv3.Config, error) {
	nsName := viper.GetString(types.CurrentNamespace)

	ctx, canc := context.WithTimeout(context.Background(), time.Duration(15)*time.Second)
	defer canc()

	// Get username and password
	secret, err := clientset.CoreV1().Secrets(nsName).Get(ctx, types.EtcdCredentialsSecretName, v1.GetOptions{})
	if err != nil {
		setupLog.Error(err, "error while trying to get secret with etcd credentials, skipping...")
		return nil, err
	}

	// get it
	data := secret.Data

	unameVal, unameExists := data["username"]
	passVal, passExist := data["password"]

	if !unameExists && !passExist {
		return nil, fmt.Errorf("username and password could not be found")
	}

	if len(unameVal) == 0 && len(passVal) == 0 {
		setupLog.V(int(zapcore.WarnLevel)).Info("username and password are both empty")
	}

	cfg := &clientv3.Config{}
	if len(unameVal) > 0 {
		cfg.Username = string(unameVal)
	}
	if len(passVal) > 0 {
		cfg.Password = string(passVal)
	}

	return cfg, nil
}
