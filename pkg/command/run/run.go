// Copyright © 2022 Cisco
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

package run

import (
	"errors"
	"fmt"
	"os"

	"github.com/CloudNativeSDWAN/cnwan-operator/controllers"
	"github.com/CloudNativeSDWAN/cnwan-operator/pkg/servregistry"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
)

// TODO: when #81 is fixed ctrl.Log will be removed in favor of zerolog.
var log zerolog.Logger

const (
	defaultNamespaceName         string = "cnwan-operator-system"
	namespaceEnvName             string = "CNWAN_OPERATOR_NAMESPACE"
	defaultSettingsConfigMapName string = "cnwan-operator-settings"
	opKey                        string = "owner"
	opVal                        string = "cnwan-operator"
	autoValue                    string = "auto"

	flagCloudMetadataNetwork    string = "cloud-metadata.network"
	flagCloudMetadataSubNetwork string = "cloud-metadata.subnetwork"
	networkNameMetadataKey      string = "cnwan.io/network"
	subNetworkNameMetadataKey   string = "cnwan.io/sub-network"
	platformNameMetadataKey     string = "cnwan.io/platform"
)

type Options struct {
	WatchNamespacesByDefault bool                   `yaml:"watchNamespacesByDefault"`
	ServiceSettings          *ServiceSettings       `yaml:",inline"`
	CloudMetadata            *CloudMetadataSettings `yaml:"cloudMetadata"`

	PersistentMetadata map[string]string
	RunningInK8s       bool
	Namespace          string
}

type ServiceSettings struct {
	Annotations []string `yaml:"serviceAnnotations"`
}

type CloudMetadataSettings struct {
	Network    string `yaml:"network"`
	SubNetwork string `yaml:"subNetwork"`
}

type fileOrK8sResource struct {
	path string
	k8s  string
}

func GetRunCommand() *cobra.Command {
	// -----------------------------
	// Inits and defaults
	// -----------------------------

	opts := &Options{
		ServiceSettings: &ServiceSettings{},
		RunningInK8s: func() bool {
			_, err := rest.InClusterConfig()
			if err != nil && errors.Is(err, rest.ErrNotInCluster) {
				return false
			}

			return true
		}(),
		PersistentMetadata: map[string]string{},
		CloudMetadata:      &CloudMetadataSettings{},
	}
	optsFile := &fileOrK8sResource{}
	cloudMetadataCredsFile := &fileOrK8sResource{}

	// -----------------------------
	// The command
	// -----------------------------

	cmd := &cobra.Command{
		Use:   "run [COMMAND] [OPTIONS]",
		Short: "Run the program.",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			log = zerolog.New(os.Stderr).With().Timestamp().Logger()

			if err := parseOperatorCommand(optsFile, opts, cmd); err != nil {
				return fmt.Errorf("error while parsing options: %w", err)
			}

			if opts.CloudMetadata.Network == autoValue ||
				opts.CloudMetadata.SubNetwork == autoValue {
				if err := retrieveCloudNetworkCfg(cloudMetadataCredsFile, opts); err != nil {
					return fmt.Errorf("error while parsing cloud metadata options: %w", err)
				}
			}

			return nil
		},
		Example: "run --watch-namespaces-by-default --service-annotations=traffic-profile,hash-commit",
	}

	// -----------------------------
	// Flags
	// -----------------------------

	cmd.PersistentFlags().BoolVar(&opts.WatchNamespacesByDefault,
		"watch-namespaces-by-default", false,
		"whether to watch all namespaces unless explictly disabled.")
	cmd.PersistentFlags().StringSliceVar(&opts.ServiceSettings.Annotations,
		"service-annotations", []string{},
		"comma-separated list of service annotation keys to watch.")
	cmd.PersistentFlags().StringVar(&optsFile.path, "options.path", "",
		"path to the options file.")
	cmd.PersistentFlags().StringVar(&optsFile.k8s, "options.configmap", func() string {
		if opts.RunningInK8s {
			return defaultSettingsConfigMapName
		}

		return ""
	}(),
		"name of the configmap with operator options. Must be in the same namespace. "+
			"Will be ignored if options.path is also provided")
	cmd.PersistentFlags().StringVar(&opts.CloudMetadata.Network,
		flagCloudMetadataNetwork, "",
		"network's name that will be registered with the metadata.")
	cmd.PersistentFlags().StringVar(&opts.CloudMetadata.SubNetwork,
		flagCloudMetadataSubNetwork, "",
		"subnetwork's name that will be registered with the metadata.")
	cmd.PersistentFlags().StringVar(&cloudMetadataCredsFile.path,
		"cloud-metadata.credentials-path", "",
		"path to the credentials of the running cluster. "+
			"Used only if the other cloud-metadata flags are set to auto.")
	cmd.PersistentFlags().StringVar(&cloudMetadataCredsFile.k8s,
		"cloud-metadata.credentials-secret", "",
		"name of the Kubernetes secret to the credentials of the running cluster. "+
			"Used only if the other cloud-metadata flags are set to auto. "+
			"Will be ignored if cloud-metadata.credentials-path is provided.")

	// -----------------------------
	// Sub commands
	// -----------------------------

	// TODO: add commands

	return cmd
}

func parseOperatorCommand(flagOpts *fileOrK8sResource, opts *Options, cmd *cobra.Command) error {
	// --------------------------------
	// Get the current namespace
	// --------------------------------

	opts.Namespace = func() string {
		if nsName := os.Getenv(namespaceEnvName); nsName != "" {
			return nsName
		}

		log.Warn().Str("default", defaultNamespaceName).
			Msg("CNWAN_OPERATOR_NAMESPACE environment variable does not exist: using default value")
		return defaultNamespaceName
	}()

	// --------------------------------
	// Get options from path or configmap
	// --------------------------------

	if flagOpts.path != "" || flagOpts.k8s != "" {
		var (
			fileOptions        []byte
			decodedFileOptions *Options
		)

		var k8sres *k8sResource
		if flagOpts.k8s != "" {
			k8sres = &k8sResource{
				Type:      "configmap",
				Namespace: opts.Namespace,
				Name:      flagOpts.k8s,
			}
		}

		fileOptions, err := getFileFromPathOrK8sResource(flagOpts.path, k8sres)
		if err != nil {
			return fmt.Errorf("could not load options: %w", err)
		}

		// Unmarshal the file and put the values found inside into our main
		// options struct...
		if err := yaml.Unmarshal(fileOptions, &decodedFileOptions); err != nil {
			return fmt.Errorf("cannot decode options %s: %w", flagOpts.path, err)
		}

		// ... unless cmd flags are provided. In which case they (the flags)
		// take precedence.
		if !cmd.Flag("watch-namespaces-by-default").Changed {
			opts.WatchNamespacesByDefault = decodedFileOptions.WatchNamespacesByDefault
		}
		if !cmd.Flag("service-annotations").Changed {
			opts.ServiceSettings.Annotations = decodedFileOptions.ServiceSettings.Annotations
		}
		if decodedFileOptions.CloudMetadata != nil {
			if !cmd.Flag(flagCloudMetadataNetwork).Changed {
				opts.CloudMetadata.Network = decodedFileOptions.CloudMetadata.Network
			}
			if !cmd.Flag(flagCloudMetadataSubNetwork).Changed {
				opts.CloudMetadata.SubNetwork = decodedFileOptions.CloudMetadata.SubNetwork
			}
		}
	}

	if len(opts.ServiceSettings.Annotations) == 0 {
		return fmt.Errorf("no service annotations provided")
	}

	// --------------------------------
	// Persistent metadata
	// --------------------------------

	if opts.CloudMetadata.Network != "" {
		opts.PersistentMetadata[networkNameMetadataKey] = opts.CloudMetadata.Network
	}
	if opts.CloudMetadata.SubNetwork != "" {
		opts.PersistentMetadata[subNetworkNameMetadataKey] = opts.CloudMetadata.SubNetwork
	}

	return nil
}

func run(sr servregistry.ServiceRegistry, opts *Options) error {
	persistentMeta := []servregistry.MetadataPair{}
	for key, val := range opts.PersistentMetadata {
		persistentMeta = append(persistentMeta, servregistry.MetadataPair{
			Key:   key,
			Value: val,
		})
	}

	srBroker, err := servregistry.NewBroker(sr, servregistry.MetadataPair{Key: opKey, Value: opVal}, persistentMeta...)
	if err != nil {
		return fmt.Errorf("cannot start service registry broker: %w", err)
	}

	scheme := k8sruntime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme

	// Controller manager
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		LeaderElection:     false,
		MetricsBindAddress: "0",
	})
	if err != nil {
		return fmt.Errorf("cannot start controller manager: %w", err)
	}

	// Service controller
	if err = (&controllers.ServiceReconciler{
		Client:                   mgr.GetClient(),
		Log:                      ctrl.Log.WithName("controllers").WithName("Service"),
		Scheme:                   mgr.GetScheme(),
		ServRegBroker:            srBroker,
		WatchNamespacesByDefault: opts.WatchNamespacesByDefault,
		AllowedAnnotations:       opts.ServiceSettings.Annotations,
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("cannot create service controller: %w", err)
	}

	// Namespace controller
	if err = (&controllers.NamespaceReconciler{
		Client:                   mgr.GetClient(),
		Log:                      ctrl.Log.WithName("controllers").WithName("Namespace"),
		Scheme:                   mgr.GetScheme(),
		ServRegBroker:            srBroker,
		WatchNamespacesByDefault: opts.WatchNamespacesByDefault,
		AllowedAnnotations:       opts.ServiceSettings.Annotations,
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("cannot create namespace controller: %w", err)
	}
	// +kubebuilder:scaffold:builder

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		return fmt.Errorf("error while starting controller manager: %w", err)
	}

	return nil
}
