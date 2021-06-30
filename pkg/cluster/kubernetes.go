// Copyright © 2021 Cisco
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

package cluster

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	defaultK8sNamespace                   string = "cnwan-operator-system"
	defaultGoogleServiceAccountSecretName string = "google-service-account"
	defaultEtcdCredentialsSecretName      string = "etcd-credentials"
)

var (
	kcli kubernetes.Interface
)

func getK8sClientSet() (kubernetes.Interface, error) {
	if kcli != nil {
		return kcli, nil
	}

	k8sconf, err := ctrl.GetConfig()
	if err != nil {
		return nil, err
	}

	kcli, err = kubernetes.NewForConfig(k8sconf)
	if err != nil {
		return nil, err
	}

	return kcli, nil
}

func getSecret(ctx context.Context, name string) (*corev1.Secret, error) {
	cli, err := getK8sClientSet()
	if err != nil {
		return nil, err
	}

	// TODO: May change this on future to use a different namespace.
	secret, err := cli.CoreV1().Secrets(defaultK8sNamespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return secret, nil
}

// GetGoogleServiceAccountSecret tries to retrieve the Google Service Account
// secret from Kubernetes, so that it could be used to login to Google Cloud
// services such as Service Directory or to pull cloud metadata/configuration.
func GetGoogleServiceAccountSecret(ctx context.Context) ([]byte, error) {
	secret, err := getSecret(ctx, defaultGoogleServiceAccountSecretName)
	if err != nil {
		return nil, err
	}

	switch l := len(secret.Data); {
	case l == 0:
		return nil, fmt.Errorf(`secret %s/%s has no data`, defaultK8sNamespace, defaultGoogleServiceAccountSecretName)
	case l > 1:
		return nil, fmt.Errorf(`secrets  %s/%s has multiple data`, defaultK8sNamespace, defaultGoogleServiceAccountSecretName)
	}

	var data []byte
	for _, d := range secret.Data {
		data = d
		break
	}

	return data, nil
}

// GetEtcdCredentials tries to retrieve the etcd credentials secret from
// Kubernetes, so that it could be used to start an etcd client.
func GetEtcdCredentialsSecret(ctx context.Context) (string, string, error) {
	secret, err := getSecret(ctx, defaultEtcdCredentialsSecretName)
	if err != nil {
		return "", "", err
	}

	return string(secret.Data["username"]), string(secret.Data["password"]), nil
}
