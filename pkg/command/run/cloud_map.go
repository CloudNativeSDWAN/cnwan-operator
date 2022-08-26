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
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/CloudNativeSDWAN/cnwan-operator/pkg/servregistry/aws/cloudmap"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const (
	defaultCloudMapConfigMapName         = "cloud-map-options"
	defaultCloudMapCredentialsSecretName = "cloud-map-credentials"
)

type CloudMapOptions struct {
	DefaultRegion string `yaml:"defaultRegion"`

	credentialsBytes []byte
}

func getRunCloudMapCommand(operatorOpts *Options) *cobra.Command {
	// -----------------------------
	// Inits and defaults
	// -----------------------------

	opts := &CloudMapOptions{}
	cmFileOpts := &fileOrK8sResource{}
	credsOpts := &fileOrK8sResource{}

	// -----------------------------
	// The command
	// -----------------------------

	cmd := &cobra.Command{
		Use:     "cloudmap [OPTIONS]",
		Aliases: []string{"cm", "aws-cloud-map", "with-cloud-map"},
		Short:   "Run the program with AWS Cloud Map",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// -- Parse the options
			if err := parseCloudMapCommand(cmFileOpts, opts, operatorOpts, cmd); err != nil {
				return fmt.Errorf("error while parsing cloud map command: %w", err)
			}

			// -- Get the credentials file
			if credsOpts.path == "" && credsOpts.k8s == "" {
				homeDir, err := os.UserHomeDir()
				if err != nil {
					return fmt.Errorf("cannot detect user home directory: %w", err)
				}

				credsOpts.path = path.Join(homeDir, ".aws", "credentials")
			}

			var k8sres *k8sResource
			if credsOpts.k8s != "" {
				k8sres = &k8sResource{
					Type:      "secret",
					Namespace: operatorOpts.Namespace,
					Name:      credsOpts.k8s,
				}
			}
			credentialsBytes, err := getFileFromPathOrK8sResource(credsOpts.path, k8sres)
			if err != nil {
				return fmt.Errorf("cannot get credentials for cloud metadata: %w", err)
			}

			opts.credentialsBytes = credentialsBytes

			return nil
		},
		RunE: func(_ *cobra.Command, _ []string) error {
			return runWithCloudMap(operatorOpts, opts)
		},
		Example: "cloudmap --default-region us-east-1",
	}

	// -----------------------------
	// Flags
	// -----------------------------

	cmd.Flags().StringVar(&opts.DefaultRegion, "default-region", "",
		"region/location where to register resources.")
	cmd.Flags().StringVar(&cmFileOpts.path, "cloud-map.options-path", "",
		"path to the file containing cloud map options.")
	cmd.Flags().StringVar(&cmFileOpts.k8s, "cloud-map.options-configmap", func() string {
		if operatorOpts.RunningInK8s {
			return defaultCloudMapConfigMapName
		}

		return ""
	}(),
		"name of the Kubernetes config map containing settings.")
	cmd.Flags().StringVar(&credsOpts.path, "cloud-map.credentials-path", "",
		"path to the credentials file.")
	cmd.Flags().StringVar(&credsOpts.k8s, "cloud-map.credentials-secret", func() string {
		if operatorOpts.RunningInK8s {
			return defaultCloudMapCredentialsSecretName
		}

		return ""
	}(),
		"name of the Kubernetes secret containing the credentials.")

	return cmd
}

func parseCloudMapCommand(flagOpts *fileOrK8sResource, cmOpts *CloudMapOptions, operatorOpts *Options, cmd *cobra.Command) error {
	// --------------------------------
	// Get options from path or configmap
	// --------------------------------

	if flagOpts.path != "" || flagOpts.k8s != "" {
		var (
			fileOptions        []byte
			decodedFileOptions *CloudMapOptions
		)

		var k8sres *k8sResource
		if flagOpts.k8s != "" {
			k8sres = &k8sResource{
				Type:      "configmap",
				Namespace: operatorOpts.Namespace,
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
		if !cmd.Flag("default-region").Changed {
			cmOpts.DefaultRegion = decodedFileOptions.DefaultRegion
		}
	}

	// -- Are the settings provided at least?
	if cmOpts.DefaultRegion == "" {
		return fmt.Errorf("no region provided")
	}

	return nil
}

func runWithCloudMap(operatorOpts *Options, cmOpts *CloudMapOptions) error {
	// TODO: when #81 is solved an merged this will be replaced
	// with zerolog
	l := ctrl.Log.WithName("CloudMap")
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
	l.Info("starting...")

	ctx, canc := context.WithTimeout(context.Background(), 15*time.Second)
	defer canc()
	const tempPath = "/tmp/cnwan-operator-aws-credentials"

	opts := []func(*config.LoadOptions) error{config.WithRegion(cmOpts.DefaultRegion)}
	if err := ioutil.WriteFile(tempPath, cmOpts.credentialsBytes, 0644); err != nil {
		return fmt.Errorf("error while trying to write credentials to temporary path: %w", err)
	}

	opts = append(opts, config.WithSharedCredentialsFiles([]string{tempPath}))
	defer os.Remove(tempPath)

	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return fmt.Errorf("error while trying to load AWS configuration: %w", err)
	}

	// TODO: the context should be given by the run function, or explicitly
	// provide a context for each call. This will be fixed with the new API
	servreg := cloudmap.NewHandler(context.Background(), servicediscovery.NewFromConfig(cfg), l)
	return run(servreg, operatorOpts)
}
