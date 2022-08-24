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
	"time"

	servicedirectory "cloud.google.com/go/servicedirectory/apiv1"
	sd "github.com/CloudNativeSDWAN/cnwan-operator/pkg/servregistry/gcloud/servicedirectory"
	"github.com/spf13/cobra"
	"google.golang.org/api/option"
	"gopkg.in/yaml.v3"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const (
	defaultServiceDirectoryConfigMapName            = "service-directory-options"
	defaultServiceDirectoryServiceAccountSecretName = "service-directory-service-account"
)

type ServiceDirectoryOptions struct {
	DefaultRegion string `yaml:"defaultRegion"`
	ProjectID     string `yaml:"projectID"`

	serviceAccountBytes []byte
}

func getRunServiceDirectoryCommand(runOpts *RunOptions, fileOptsFlags *fileOrK8sResource) *cobra.Command {
	// -----------------------------
	// Inits and defaults
	// -----------------------------

	runOpts.ServiceDirectoryOptions = &ServiceDirectoryOptions{}
	servAccOpts := &fileOrK8sResource{}

	// -----------------------------
	// The command
	// -----------------------------

	cmd := &cobra.Command{
		Use:     "servicedirectory [COMMAND] [OPTIONS]",
		Aliases: []string{"sd", "google-service-directory", "with-service-directory"},
		Short:   "Run the program with Google Service Directory.",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if err := parseServiceDirectoryCommand(runOpts.ServiceDirectoryOptions, fileOptsFlags, cmd); err != nil {
				return fmt.Errorf("error while parsing service directory options: %w", err)
			}

			// -- Get the service account
			if servAccOpts.path == "" && servAccOpts.k8s == "" {
				return fmt.Errorf("no service account provided")
			}

			var k8sres *k8sResource
			if servAccOpts.k8s != "" {
				k8sres = &k8sResource{
					Type:      "secret",
					Namespace: getCurrentNamespace(),
					Name:      servAccOpts.k8s,
				}
			}

			serviceAccount, err := getFileFromPathOrK8sResource(servAccOpts.path, k8sres)
			if err != nil {
				return fmt.Errorf("error while retrieving GCP service account: %w", err)
			}
			runOpts.ServiceDirectoryOptions.serviceAccountBytes = serviceAccount

			return nil
		},
		RunE: func(_ *cobra.Command, _ []string) error {
			return runWithServiceDirectory(runOpts)
		},
		Example: "servicedirectory --project-id my-project-id --default-region us-east1",
	}

	// -----------------------------
	// Flags
	// -----------------------------

	cmd.Flags().StringVar(&runOpts.ServiceDirectoryOptions.DefaultRegion, "default-region", "",
		"region/location where to register resources. Write auto to try to get it automatically.")
	cmd.Flags().StringVar(&runOpts.ServiceDirectoryOptions.ProjectID, "project-id", "",
		"Google Cloud project ID. Write auto to try to get it automatically.")
	cmd.Flags().StringVar(&servAccOpts.path, "service-account-path", "",
		"path to the service account file.")
	cmd.Flags().StringVar(&servAccOpts.k8s, "service-account-secret", func() string {
		if runOpts.RunningInK8s {
			return defaultServiceDirectoryServiceAccountSecretName
		}

		return ""
	}(),
		"name of the Kubernetes secret containing the service account.")

	return cmd
}

func parseServiceDirectoryCommand(sdOpts *ServiceDirectoryOptions, flagOpts *fileOrK8sResource, cmd *cobra.Command) error {
	// --------------------------------
	// Get options from path or configmap
	// --------------------------------

	if flagOpts.path != "" || flagOpts.k8s != "" {
		var (
			fileOptions        []byte
			decodedFileOptions struct {
				*ServiceDirectoryOptions `yaml:"serviceDirectory"`
			}
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
		if !cmd.Flag("default-region").Changed {
			sdOpts.DefaultRegion = decodedFileOptions.DefaultRegion
		}

		if !cmd.Flag("project-id").Changed {
			sdOpts.ProjectID = decodedFileOptions.ProjectID
		}
	}

	// --------------------------------
	// Validation
	// --------------------------------

	if sdOpts.DefaultRegion == "" {
		return fmt.Errorf("no region provided")
	}

	if sdOpts.ProjectID == "" {
		return fmt.Errorf("no project ID provided")
	}

	return nil
}

func runWithServiceDirectory(opts *RunOptions) error {
	sdOpts := opts.ServiceDirectoryOptions

	// TODO: when #81 is solved an merged this will be replaced
	// with zerolog
	l := ctrl.Log.WithName("ServiceDirectory")
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
	l.Info("starting...")

	ctx, canc := context.WithTimeout(context.Background(), 15*time.Second)
	defer canc()

	cli, err := servicedirectory.
		NewRegistrationClient(ctx, option.WithCredentialsJSON(sdOpts.serviceAccountBytes))
	if err != nil {
		return fmt.Errorf("could not get start service directory client: %s", err)
	}

	defer cli.Close()

	// TODO: the context should be given by the run function, or explicitly
	// provide a context for each call. This will be fixed with the new API.
	sr := &sd.Handler{
		ProjectID:     sdOpts.ProjectID,
		DefaultRegion: sdOpts.DefaultRegion,
		Log:           l,
		Context:       context.Background(),
		Client:        cli,
	}

	return run(sr, opts)
}
