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
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

func TestGetGoogleServiceAccountSecret(t *testing.T) {

	anyErr := fmt.Errorf("any")
	cases := []struct {
		kcli   kubernetes.Interface
		expRes []byte
		expErr error
	}{
		{
			kcli:   fake.NewSimpleClientset(),
			expErr: anyErr,
		},
		{
			kcli: func() kubernetes.Interface {
				sec := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      defaultGoogleServiceAccountSecretName,
						Namespace: defaultK8sNamespace,
					},
				}
				return fake.NewSimpleClientset(sec)
			}(),
			expErr: fmt.Errorf(`secret %s/%s has no data`, defaultK8sNamespace, defaultGoogleServiceAccountSecretName),
		},
		{
			kcli: func() kubernetes.Interface {
				sec := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      defaultGoogleServiceAccountSecretName,
						Namespace: defaultK8sNamespace,
					},
					Data: map[string][]byte{
						"test":   []byte("test"),
						"test-1": []byte("test-1"),
					},
				}
				return fake.NewSimpleClientset(sec)
			}(),
			expErr: fmt.Errorf(`secrets  %s/%s has multiple data`, defaultK8sNamespace, defaultGoogleServiceAccountSecretName),
		},
		{
			kcli: func() kubernetes.Interface {
				sec := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      defaultGoogleServiceAccountSecretName,
						Namespace: defaultK8sNamespace,
					},
					Data: map[string][]byte{
						"test": []byte("test"),
					},
				}
				return fake.NewSimpleClientset(sec)
			}(),
			expRes: []byte("test"),
		},
	}

	for i, currCase := range cases {
		a := assert.New(t)
		kcli = currCase.kcli
		res, err := GetGoogleServiceAccountSecret(context.Background())

		if currCase.expErr == anyErr {
			if err == nil {
				a.FailNow("case failed: was expecting error but no error occurred", "i", i)
			}

			continue
		}

		if !a.Equal(currCase.expRes, res) || !a.Equal(currCase.expErr, err) {
			a.FailNow("case failed", "i", i)
		}

		kcli = nil
	}
}

func TestGetEtcdCredentialsSecret(t *testing.T) {

	anyErr := fmt.Errorf("any")
	cases := []struct {
		kcli    kubernetes.Interface
		expUser string
		expPass string
		expErr  error
	}{
		{
			kcli:   fake.NewSimpleClientset(),
			expErr: anyErr,
		},
		{
			kcli: func() kubernetes.Interface {
				sec := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      defaultEtcdCredentialsSecretName,
						Namespace: defaultK8sNamespace,
					},
				}
				return fake.NewSimpleClientset(sec)
			}(),
			expUser: "",
			expPass: "",
		},
		{
			kcli: func() kubernetes.Interface {
				sec := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      defaultEtcdCredentialsSecretName,
						Namespace: defaultK8sNamespace,
					},
					Data: map[string][]byte{
						"username": []byte("test"),
						"password": []byte("test-1"),
					},
				}
				return fake.NewSimpleClientset(sec)
			}(),
			expUser: "test",
			expPass: "test-1",
		},
	}

	for i, currCase := range cases {
		a := assert.New(t)
		kcli = currCase.kcli
		user, pass, err := GetEtcdCredentialsSecret(context.Background())

		if currCase.expErr == anyErr {
			if err == nil {
				a.FailNow("case failed: was expecting error but no error occurred", "i", i)
			}

			continue
		}

		if !a.Equal(currCase.expUser, user) || !a.Equal(currCase.expPass, pass) || !a.Equal(currCase.expErr, err) {
			a.FailNow("case failed", "i", i)
		}

		kcli = nil
	}
}
