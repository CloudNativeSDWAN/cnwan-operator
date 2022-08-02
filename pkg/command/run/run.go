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

	"github.com/CloudNativeSDWAN/cnwan-operator/pkg/cluster"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"google.golang.org/api/option"
	"gopkg.in/yaml.v3"
	"k8s.io/client-go/rest"
)

var log zerolog.Logger

const (
	defaultNamespaceName         string = "cnwan-operator-system"
	namespaceEnvName             string = "CNWAN_OPERATOR_NAMESPACE"
	defaultSettingsConfigMapName string = "cnwan-operator-settings"
)

type Options struct {
	WatchNamespacesByDefault bool                   `yaml:"watchNamespacesByDefault"`
	ServiceSettings          *ServiceSettings       `yaml:",inline"`
	CloudMetadata            *CloudMetadataSettings `yaml:"cloudMetadata"`

	RunningInK8s bool
	Namespace    string
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
							Msg("getting service account from secret...")

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
						return nil, nil
					}

					ctx, canc := context.WithTimeout(context.Background(), 15*time.Second)
					defer canc()

					switch cluster.WhereAmIRunning() {
					case cluster.GKECluster:
						nw, err := cluster.GetNetworkFromGKE(ctx, option.WithCredentialsJSON(credentialsBytes))
						if err != nil {
							return nil, fmt.Errorf("cannot get network configuration from GKE: %w", err)
						}

						return nw, nil
					case cluster.EKSCluster:
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

				if opts.CloudMetadata.Network != nil && *opts.CloudMetadata.Network == "auto" {
					opts.CloudMetadata.Network = &netwCfg.NetworkName
				}

				if opts.CloudMetadata.SubNetwork != nil && *opts.CloudMetadata.SubNetwork == "auto" {
					opts.CloudMetadata.SubNetwork = &netwCfg.SubNetworkName
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
