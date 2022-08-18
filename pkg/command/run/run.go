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
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/CloudNativeSDWAN/cnwan-operator/controllers"
	"github.com/CloudNativeSDWAN/cnwan-operator/pkg/cluster"
	"github.com/CloudNativeSDWAN/cnwan-operator/pkg/servregistry"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"google.golang.org/api/option"
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
	Network    *string `yaml:"network"`
	SubNetwork *string `yaml:"subNetwork"`
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
	}

	var (
		cloudMetadataNetwork     string
		cloudMetadataSubNetwork  string
		cloudMetadataCredsPath   string
		cloudMetadataCredsSecret string
		optsPath                 string
		optsConfigMap            string
	)

	// -----------------------------
	// The command
	// -----------------------------

	cmd := &cobra.Command{
		Use:   "run [COMMAND] [OPTIONS]",
		Short: "Run the program.",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			log = zerolog.New(os.Stderr).With().Timestamp().Logger()

			// -- Get the current namespace
			if nsName := os.Getenv(namespaceEnvName); nsName != "" {
				opts.Namespace = nsName
			} else {
				log.Warn().
					Str("default", defaultNamespaceName).
					Msg("CNWAN_OPERATOR_NAMESPACE environment variable does not exist: using default value")
				opts.Namespace = defaultNamespaceName
			}

			// -- Get the options from file or ConfigMap
			if optsPath != "" || optsConfigMap != "" {
				var (
					fileOptions        []byte
					decodedFileOptions *Options
				)

				// -- Get the options from path
				if optsPath != "" {
					if optsConfigMap != "" {
						log.Warn().Msg("both path and configmap flags are provided: only the path will be used")
						optsConfigMap = ""
					}

					log.Debug().Str("path", optsPath).
						Msg("getting options from file...")
					byteOpts, err := ioutil.ReadFile(optsPath)
					if err != nil {
						return fmt.Errorf("cannot open file %s: %w", optsPath, err)
					}

					fileOptions = byteOpts
				}

				// -- Get options from configmap
				if optsConfigMap != "" {
					log.Debug().
						Str("namespace", opts.Namespace).
						Str("name", optsConfigMap).
						Msg("getting options from configmap...")

					ctx, canc := context.WithTimeout(context.Background(), 10*time.Second)
					cfgs, err := cluster.GetFilesFromConfigMap(ctx, opts.Namespace, optsConfigMap)
					if err != nil {
						canc()
						return fmt.Errorf("cannot get configmap: %w", err)
					}
					canc()

					fileOptions = cfgs[0]
				}

				if len(fileOptions) > 0 {
					if err := yaml.Unmarshal(fileOptions, &decodedFileOptions); err != nil {
						return fmt.Errorf("cannot decode options %s: %w", optsPath, err)
					}
				}

				// -- Parse the cmd flags
				if !cmd.Flag("watch-namespaces-by-default").Changed {
					opts.WatchNamespacesByDefault = decodedFileOptions.WatchNamespacesByDefault
				}

				if !cmd.Flag("service-annotations").Changed {
					opts.ServiceSettings.Annotations = decodedFileOptions.ServiceSettings.Annotations
				}

				if decodedFileOptions.CloudMetadata != nil {
					opts.CloudMetadata = decodedFileOptions.CloudMetadata
				}
			}

			if len(opts.ServiceSettings.Annotations) == 0 {
				return fmt.Errorf("no service annotations provided")
			}

			// -- Parse the cloud metadata
			if cmd.Flag("cloud-metadata.network").Changed || cmd.Flag("cloud-metadata.subnetwork").Changed {
				if opts.CloudMetadata == nil {
					opts.CloudMetadata = &CloudMetadataSettings{}
				}

				if cmd.Flag("cloud-metadata.network").Changed {
					opts.CloudMetadata.Network = &cloudMetadataNetwork
				}

				if cmd.Flag("cloud-metadata.subnetwork").Changed {
					opts.CloudMetadata.SubNetwork = &cloudMetadataSubNetwork
				}
			}

			if opts.CloudMetadata != nil {
				// -- Get the credentials
				credentialsBytes, err := func() ([]byte, error) {
					if cloudMetadataCredsPath == "" && cloudMetadataCredsSecret == "" {
						return nil, nil
					}

					if opts.CloudMetadata.Network != nil && *opts.CloudMetadata.Network != "auto" &&
						opts.CloudMetadata.SubNetwork != nil && *opts.CloudMetadata.SubNetwork != "auto" {
						// No auto values to take. Let's stop here.
						log.Info().Msg("neither network nor subnetwork are set to auto, no credential will be loaded")
						return nil, nil
					}

					var creds []byte
					if cloudMetadataCredsPath != "" {
						if cloudMetadataCredsSecret != "" {
							log.Warn().Msg("both path and secret cloud-metadata flags are provided: only the path will be used")
							cloudMetadataCredsSecret = ""
						}

						log.Debug().Str("cloud-metadata.credentials-path", cloudMetadataCredsPath).
							Msg("getting credentials from file...")
						byteOpts, err := ioutil.ReadFile(cloudMetadataCredsPath)
						if err != nil {
							return nil, fmt.Errorf("cannot open file %s: %w", cloudMetadataCredsPath, err)
						}

						return byteOpts, nil
					}

					if cloudMetadataCredsSecret != "" {
						log.Debug().
							Str("namespace", opts.Namespace).
							Str("name", cloudMetadataCredsSecret).
							Msg("getting cloud map credentials from secret...")

						ctx, canc := context.WithTimeout(context.Background(), 10*time.Second)
						defer canc()

						secrets, err := cluster.GetFilesFromSecret(ctx, opts.Namespace, cloudMetadataCredsSecret)
						if err != nil {
							return nil, fmt.Errorf("cannot get secret: %w", err)
						}

						return secrets[0], nil
					}

					return creds, nil
				}()
				if err != nil {
					return err
				}

				// -- Get data automatically?
				netwCfg, err := func() (*cluster.NetworkConfiguration, error) {
					if len(credentialsBytes) == 0 {
						if (opts.CloudMetadata.Network != nil && *opts.CloudMetadata.Network == "auto") ||
							(opts.CloudMetadata.SubNetwork != nil && *opts.CloudMetadata.SubNetwork == "auto") {
							return nil, fmt.Errorf("cannot infer network and/or subnetwork without credentials file. Please provide it via flags or option.")
						}
					}

					ctx, canc := context.WithTimeout(context.Background(), 15*time.Second)
					defer canc()

					switch cluster.WhereAmIRunning() {
					case cluster.GKECluster:
						opts.PersistentMetadata["cnwan.io/platform"] = string(cluster.GKECluster)
						nw, err := cluster.GetNetworkFromGKE(ctx, option.WithCredentialsJSON(credentialsBytes))
						if err != nil {
							return nil, fmt.Errorf("cannot get network configuration from GKE: %w", err)
						}

						return nw, nil
					case cluster.EKSCluster:
						opts.PersistentMetadata["cnwan.io/platform"] = string(cluster.EKSCluster)
						nw, err := cluster.GetNetworkFromEKS(ctx)
						if err != nil {
							return nil, fmt.Errorf("cannot get network configuration from EKS: %w", err)
						}

						return nw, nil
					default:
						return nil, fmt.Errorf("cannot get network configuration: unsupported cluster")
					}
				}()
				if err != nil {
					return err
				}

				if opts.CloudMetadata.Network != nil {
					if *opts.CloudMetadata.Network == "auto" {
						opts.CloudMetadata.Network = &netwCfg.NetworkName
					}

					opts.PersistentMetadata["cnwan.io/network"] = *opts.CloudMetadata.Network
				}

				if opts.CloudMetadata.SubNetwork != nil {
					if *opts.CloudMetadata.SubNetwork == "auto" {
						opts.CloudMetadata.SubNetwork = &netwCfg.SubNetworkName
					}
					opts.PersistentMetadata["cnwan.io/sub-network"] = *opts.CloudMetadata.SubNetwork
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
	cmd.PersistentFlags().StringVar(&cloudMetadataNetwork,
		"cloud-metadata.network", "",
		"network's name that will be registered with the metadata.")
	cmd.PersistentFlags().StringVar(&cloudMetadataSubNetwork,
		"cloud-metadata.subnetwork", "",
		"subnetwork's name that will be registered with the metadata.")
	cmd.PersistentFlags().StringVar(&optsPath, "options.path", "",
		"path to the options file.")
	cmd.PersistentFlags().StringVar(&optsConfigMap, "options.configmap", func() string {
		if opts.RunningInK8s {
			return defaultSettingsConfigMapName
		}

		return ""
	}(),
		"name of the configmap with operator options. Must be in the same namespace.")
	cmd.PersistentFlags().StringVar(&cloudMetadataCredsPath, "cloud-metadata.credentials-path", "",
		`path to the credentials of the running cluster. `+
			`Used only if the other cloud-metadata flags are set to auto.`)
	cmd.PersistentFlags().StringVar(&cloudMetadataCredsSecret, "cloud-metadata.credentials-secret", "",
		`name of the Kubernetes secret to the credentials of the running cluster. `+
			`Used only if the other cloud-metadata flags are set to auto.`)

	// -----------------------------
	// Sub commands
	// -----------------------------

	// TODO: add commands

	return cmd
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
