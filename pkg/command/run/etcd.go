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
	"strings"
	"syscall"
	"time"

	"github.com/CloudNativeSDWAN/cnwan-operator/pkg/cluster"
	"github.com/CloudNativeSDWAN/cnwan-operator/pkg/servregistry/etcd"
	"github.com/spf13/cobra"
	clientv3 "go.etcd.io/etcd/client/v3"
	"golang.org/x/term"
	"gopkg.in/yaml.v3"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const (
	defaultEtcdConfigMapName         = "etcd-options"
	defaultEtcdCredentialsSecretName = "etcd-credentials"
)

type EtcdOptions struct {
	Prefix    string   `yaml:"prefix"`
	Endpoints []string `yaml:"endpoints"`

	Username string
	Password string
}

func getRunEtcdCommand(operatorOpts *Options) *cobra.Command {
	// -----------------------------
	// Inits and defaults
	// -----------------------------

	opts := &EtcdOptions{}
	fileOpts := &fileOrK8sResource{}
	credsOpts := &fileOrK8sResource{}

	// This is used for the --password flag option, to signal user has
	// indeed a password for authentication.
	var tmpPassFlag bool

	// -----------------------------
	// The command
	// -----------------------------

	cmd := &cobra.Command{
		Use:   "etcd [COMMAND] [OPTIONS]",
		Short: "Run the program with etcd",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if err := parseEtcdCommand(fileOpts, opts, operatorOpts, cmd); err != nil {
				return fmt.Errorf("error while parsing options: %w", err)
			}

			if credsOpts.k8s != "" {
				err := func() error {
					ctx, canc := context.WithTimeout(context.Background(), 10*time.Second)
					defer canc()

					secret, err := cluster.GetDataFromSecret(ctx, operatorOpts.Namespace, credsOpts.k8s)
					if err != nil {
						return fmt.Errorf("cannot get secret: %w", err)
					}

					username, exists := secret["username"]
					if !exists {
						return fmt.Errorf("secret %s/%s does not contain any username", operatorOpts.Namespace, credsOpts.k8s)
					} else {
						if opts.Username == "" {
							// The user did not overwrite this via flag.
							// So we're going to the value from the secret.
							opts.Username = string(username)
						}
					}

					if password, exists := secret["password"]; exists {
						opts.Password = string(password)
					}

					return nil
				}()
				if err != nil {
					return fmt.Errorf("cannot get etcd credentials: %w", err)
				}
			}

			// -- Ask for password
			if tmpPassFlag {
				if operatorOpts.RunningInK8s {
					return fmt.Errorf("cannot use --password while running in Kubernetes. Please use a Secret instead.")
				}

				fmt.Printf("Please enter password for %s: ", opts.Username)
				bytePassword, err := term.ReadPassword(int(syscall.Stdin))
				if err != nil {
					return fmt.Errorf("cannot read password from terminal: %w", err)
				}
				fmt.Println()

				opts.Password = strings.TrimSpace(string(bytePassword))
			}

			return nil
		},
		RunE: func(_ *cobra.Command, _ []string) error {
			return runWithEtcd(operatorOpts, opts)
		},
		Example: "etcd --username root -p --endpoints 10.10.10.10:2379,10.10.10.11:2379",
	}

	// -----------------------------
	// Flags
	// -----------------------------

	cmd.Flags().StringVarP(&opts.Username, "username", "u", "",
		"username for authentication")
	cmd.Flags().BoolVarP(&tmpPassFlag, "password", "p", false,
		"enter password -- will be done interactively.")
	cmd.Flags().StringVar(&opts.Prefix, "prefix", "",
		"prefix to insert before every key.")
	cmd.Flags().StringSliceVar(&opts.Endpoints, "endpoints", []string{"localhost:2379"},
		"list of endpoints for etcd.")
	cmd.Flags().StringVar(&fileOpts.path, "etcd.options-path", "",
		"path to the file containing etcd options.")
	cmd.Flags().StringVar(&fileOpts.k8s, "etcd.options-configmap", func() string {
		if operatorOpts.RunningInK8s {
			return defaultEtcdConfigMapName
		}

		return ""
	}(),
		"name of the Kubernetes config map containing settings.")
	cmd.Flags().StringVar(&credsOpts.k8s, "credentials-secret", func() string {
		if operatorOpts.RunningInK8s {
			return defaultEtcdCredentialsSecretName
		}

		return ""
	}(),
		"name of the Kubernetes secret containing the credentials.")

	return cmd
}

func parseEtcdCommand(flagOpts *fileOrK8sResource, etcdOpts *EtcdOptions, opts *Options, cmd *cobra.Command) error {
	// -- Get the options from file or ConfigMap
	if flagOpts.path != "" || flagOpts.k8s != "" {
		var (
			fileOptions        []byte
			decodedFileOptions *EtcdOptions
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
		if !cmd.Flag("endpoints").Changed {
			etcdOpts.Endpoints = decodedFileOptions.Endpoints
		}

		if !cmd.Flag("prefix").Changed {
			etcdOpts.Prefix = decodedFileOptions.Prefix
		}
	}

	return nil
}

func runWithEtcd(operatorOpts *Options, opts *EtcdOptions) error {
	// TODO: when #81 is solved an merged this will be replaced
	// with zerolog
	l := ctrl.Log.WithName("etcd")
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
	l.Info("starting...")

	// TODO: support certificates
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   opts.Endpoints,
		Username:    opts.Username,
		Password:    opts.Password,
		DialTimeout: 15 * time.Second,
	})
	if err != nil {
		return fmt.Errorf("cannot get etcd client: %w", err)
	}

	defer cli.Close()

	// TODO: the context should be given by the run function, or explicitly
	// provide a context for each call. This will be fixed with the new API.
	servreg := etcd.NewServiceRegistryWithEtcd(context.Background(), cli, &opts.Prefix)
	return run(servreg, operatorOpts)
}
