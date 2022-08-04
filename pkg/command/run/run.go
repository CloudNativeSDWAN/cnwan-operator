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

// RunOptions is a sort of "global" options container, holding options for the
// run command, its children commands and internal options that are nott
// exposed to the user.
type RunOptions struct {
	OperatorOptions          `yaml:",inline"`
	*ServiceDirectoryOptions `yaml:"serviceDirectory"`

	PersistentMetadata map[string]string
	RunningInK8s       bool
}

// OperatorOptions contains options only for the run command. It contains
// options that are exposed externally and defines marshalling/unmarshalling
// rules.
type OperatorOptions struct {
	WatchNamespacesByDefault bool                   `yaml:"watchNamespacesByDefault"`
	ServiceSettings          *ServiceSettings       `yaml:",inline"`
	CloudMetadata            *CloudMetadataSettings `yaml:"cloudMetadata"`
}

// ServiceSettings contains options/settings only for services.
type ServiceSettings struct {
	Annotations []string `yaml:"serviceAnnotations"`
}

// CloudMetadataSettings contains options/settings for the cloud provider's
// metadata, such as the network and/or subnetwork names where our cluster is
// running in.
type CloudMetadataSettings struct {
	Network    string `yaml:"network"`
	SubNetwork string `yaml:"subNetwork"`
}

func GetRunCommand() *cobra.Command {
	// -----------------------------
	// Inits and defaults
	// -----------------------------

	runOpts := &RunOptions{
		OperatorOptions: OperatorOptions{
			ServiceSettings: &ServiceSettings{
				Annotations: []string{},
			},
			CloudMetadata: &CloudMetadataSettings{},
		},
		RunningInK8s: func() bool {
			_, err := rest.InClusterConfig()
			if err != nil && errors.Is(err, rest.ErrNotInCluster) {
				return false
			}

			return true
		}(),
		PersistentMetadata: map[string]string{},
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

			if _, exists := os.LookupEnv(namespaceEnvName); !exists {
				// This is here just to warn the user that we could not find
				// any environment variable containing the current namespace,
				// and thus loading kubernetes resources may not work if the
				// current namespace is not the default one.
				log.Warn().Str("default", defaultNamespaceName).
					Msg("CNWAN_OPERATOR_NAMESPACE environment variable does not exist: using default value")
			}

			// -- Parse the operator commands
			if err := parseOperatorCommand(&runOpts.OperatorOptions, optsFile, cmd); err != nil {
				return fmt.Errorf("error while parsing options: %w", err)
			}

			// -- Persistent metadata
			if runOpts.OperatorOptions.CloudMetadata.Network != "" {
				runOpts.PersistentMetadata[networkNameMetadataKey] = runOpts.OperatorOptions.CloudMetadata.Network
			}
			if runOpts.OperatorOptions.CloudMetadata.SubNetwork != "" {
				runOpts.PersistentMetadata[subNetworkNameMetadataKey] = runOpts.OperatorOptions.CloudMetadata.SubNetwork
			}

			// Should we get cloud metadata automatically?
			if runOpts.CloudMetadata.Network == autoValue ||
				runOpts.CloudMetadata.SubNetwork == autoValue {
				netCfg, err := retrieveCloudNetworkCfg(&runOpts.OperatorOptions, cloudMetadataCredsFile)
				if err != nil {
					return fmt.Errorf("error while parsing cloud metadata options: %w", err)
				}

				// Re-update persistent metadata
				if netCfg.PlatformName != "" {
					runOpts.PersistentMetadata[platformNameMetadataKey] = netCfg.PlatformName
				}
				if netCfg.NetworkName != "" {
					runOpts.PersistentMetadata[networkNameMetadataKey] = netCfg.NetworkName
				}
				if netCfg.SubNetworkName != "" {
					runOpts.PersistentMetadata[subNetworkNameMetadataKey] = netCfg.SubNetworkName
				}
			}

			return nil
		},
		Example: "run etcd --watch-namespaces-by-default --service-annotations=traffic-profile,hash-commit",
	}

	// -----------------------------
	// Flags
	// -----------------------------

	cmd.PersistentFlags().BoolVar(&runOpts.WatchNamespacesByDefault,
		"watch-namespaces-by-default", false,
		"whether to watch all namespaces unless explictly disabled.")
	cmd.PersistentFlags().StringSliceVar(&runOpts.ServiceSettings.Annotations,
		"service-annotations", []string{},
		"comma-separated list of service annotation keys to watch.")
	cmd.PersistentFlags().StringVar(&optsFile.path, "options-path", "",
		"path to the options file.")
	cmd.PersistentFlags().StringVar(&optsFile.k8s, "options-configmap", func() string {
		if runOpts.RunningInK8s {
			return defaultSettingsConfigMapName
		}

		return ""
	}(),
		"name of the configmap with operator options. Must be in the same namespace. "+
			"Will be ignored if operator.options-path is also provided")
	cmd.PersistentFlags().StringVar(&runOpts.CloudMetadata.Network,
		flagCloudMetadataNetwork, "",
		"network's name that will be registered with the metadata.")
	cmd.PersistentFlags().StringVar(&runOpts.CloudMetadata.SubNetwork,
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

	cmd.AddCommand(getRunServiceDirectoryCommand(runOpts, optsFile))

	return cmd
}

func parseOperatorCommand(unparsedOpOpts *OperatorOptions, flagOpts *fileOrK8sResource, cmd *cobra.Command) error {
	// --------------------------------
	// Get options from path or configmap
	// --------------------------------

	if flagOpts.path != "" || flagOpts.k8s != "" {
		var (
			fileOptions        []byte
			decodedFileOptions *OperatorOptions
		)

		var k8sres *k8sResource
		if flagOpts.k8s != "" {
			k8sres = &k8sResource{
				Type:      "configmap",
				Namespace: getCurrentNamespace(),
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
			unparsedOpOpts.WatchNamespacesByDefault = decodedFileOptions.WatchNamespacesByDefault
		}
		if !cmd.Flag("service-annotations").Changed {
			unparsedOpOpts.ServiceSettings.Annotations = decodedFileOptions.ServiceSettings.Annotations
		}
		if decodedFileOptions.CloudMetadata != nil {
			if !cmd.Flag(flagCloudMetadataNetwork).Changed {
				unparsedOpOpts.CloudMetadata.Network = decodedFileOptions.CloudMetadata.Network
			}
			if !cmd.Flag(flagCloudMetadataSubNetwork).Changed {
				unparsedOpOpts.CloudMetadata.SubNetwork = decodedFileOptions.CloudMetadata.SubNetwork
			}
		}
	}

	if len(unparsedOpOpts.ServiceSettings.Annotations) == 0 {
		return fmt.Errorf("no service annotations provided")
	}

	return nil
}

func run(sr servregistry.ServiceRegistry, opts *RunOptions) error {
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
