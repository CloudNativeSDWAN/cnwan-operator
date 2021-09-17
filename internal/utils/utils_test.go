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

package utils

import (
	"fmt"
	"testing"

	"github.com/CloudNativeSDWAN/cnwan-operator/internal/types"
	. "github.com/stretchr/testify/assert"
)

func TestParseAndValidateSettings(t *testing.T) {
	a := New(t)
	pref := "/whatever"
	port2800 := 2800
	portDef := 2379
	port2810 := 2810
	cases := []struct {
		id     string
		arg    *types.Settings
		expRes *types.Settings
		expErr error
	}{
		{
			id:     "nil-settings",
			expErr: fmt.Errorf("no settings provided"),
		},
		{
			id:     "no-service-registry-settings",
			arg:    &types.Settings{MonitorNamespacesByDefault: true},
			expErr: fmt.Errorf("no service registry provided"),
		},
		{
			id: "no-service-registry-fields",
			arg: &types.Settings{
				MonitorNamespacesByDefault: true,
				ServiceRegistrySettings:    &types.ServiceRegistrySettings{},
			},
			expErr: fmt.Errorf("no service registry provided"),
		},
		{
			id: "etcd-unknown-auth",
			arg: &types.Settings{
				ServiceRegistrySettings: &types.ServiceRegistrySettings{
					EtcdSettings: &types.EtcdSettings{
						Authentication: types.EtcdAuthenticationType("nothing"),
						Endpoints: []*types.EtcdEndpoint{
							{Host: "10.10.10.10"},
						},
					},
				},
			},
			expErr: fmt.Errorf("unrecognized authentication method for etcd"),
		},
		{
			id: "etcd-uname-pass-auth",
			arg: &types.Settings{
				MonitorNamespacesByDefault: true,
				ServiceRegistrySettings: &types.ServiceRegistrySettings{
					EtcdSettings: &types.EtcdSettings{
						Authentication: types.EtcdAuthWithUsernamePassw,
						Endpoints: []*types.EtcdEndpoint{
							{Host: "10.10.10.10"},
						},
					},
				},
			},
			expRes: &types.Settings{
				MonitorNamespacesByDefault: true,
				ServiceRegistrySettings: &types.ServiceRegistrySettings{
					EtcdSettings: &types.EtcdSettings{
						Authentication: types.EtcdAuthWithUsernamePassw,
						Endpoints: []*types.EtcdEndpoint{
							{Host: "10.10.10.10", Port: &portDef},
						},
					},
				},
			},
		},
		{
			id: "etcd-tls-auth",
			arg: &types.Settings{
				ServiceRegistrySettings: &types.ServiceRegistrySettings{
					EtcdSettings: &types.EtcdSettings{
						Authentication: types.EtcdAuthWithTLS,
						Endpoints: []*types.EtcdEndpoint{
							{Host: "10.10.10.10"},
						},
					},
				},
			},
			expErr: fmt.Errorf("etcd authentication with TLS is not supported yet"),
		},
		{
			id: "only-etcd-not-empty",
			arg: &types.Settings{
				ServiceRegistrySettings: &types.ServiceRegistrySettings{
					EtcdSettings: &types.EtcdSettings{
						Prefix: &pref,
						Endpoints: []*types.EtcdEndpoint{
							{Host: "10.10.10.10"},
							{Host: "10.10.10.10", Port: &port2800},
							{Host: "10.10.10.10", Port: &port2800},
							{Port: &port2810},
							{Host: "11.11.11.11", Port: &port2810},
						},
					},
				},
			},
			expRes: &types.Settings{
				ServiceRegistrySettings: &types.ServiceRegistrySettings{
					EtcdSettings: &types.EtcdSettings{
						Authentication: types.EtcdAuthWithNothing,
						Prefix:         &pref,
						Endpoints: []*types.EtcdEndpoint{
							{Host: "10.10.10.10", Port: &portDef},
							{Host: "10.10.10.10", Port: &port2800},
							{Host: "11.11.11.11", Port: &port2810},
						},
					},
				},
			},
		},
		{
			id: "only-etcd-empty",
			arg: &types.Settings{
				ServiceRegistrySettings: &types.ServiceRegistrySettings{
					EtcdSettings: &types.EtcdSettings{},
				},
			},
			expErr: fmt.Errorf("no etcd endpoints provided"),
		},
		{
			id: "both-but-only-etcd-is-there",
			arg: &types.Settings{
				MonitorNamespacesByDefault: true,
				Service: types.ServiceSettings{
					Annotations: []string{"one", "two"},
				},
				ServiceRegistrySettings: &types.ServiceRegistrySettings{
					EtcdSettings: &types.EtcdSettings{
						Endpoints: []*types.EtcdEndpoint{
							{Host: "10.10.10.10"},
						},
					},
					ServiceDirectorySettings: &types.ServiceDirectorySettings{},
				},
			},
			expRes: &types.Settings{
				MonitorNamespacesByDefault: true,
				Service: types.ServiceSettings{
					Annotations: []string{"one", "two"},
				},
				ServiceRegistrySettings: &types.ServiceRegistrySettings{
					EtcdSettings: &types.EtcdSettings{
						Endpoints: []*types.EtcdEndpoint{
							{Host: "10.10.10.10", Port: &portDef},
						},
					},
				},
			},
		},
		{
			id: "convert-deprecated-sd",
			arg: &types.Settings{
				Gcloud: &types.GcloudSettings{
					ServiceDirectory: &types.DeprecatedServiceDirectorySettings{
						ProjectName:   "test",
						DefaultRegion: "ca",
					},
				},
				MonitorNamespacesByDefault: true,
				Service: types.ServiceSettings{
					Annotations: []string{"one", "two"},
				},
				ServiceRegistrySettings: &types.ServiceRegistrySettings{},
			},
			expRes: &types.Settings{
				MonitorNamespacesByDefault: true,
				Service: types.ServiceSettings{
					Annotations: []string{"one", "two"},
				},
				ServiceRegistrySettings: &types.ServiceRegistrySettings{
					ServiceDirectorySettings: &types.ServiceDirectorySettings{
						ProjectID:     "test",
						DefaultRegion: "ca",
					},
				},
			},
		},
		{
			id: "do-not-convert-deprecated-sd",
			arg: &types.Settings{
				Gcloud: &types.GcloudSettings{
					ServiceDirectory: &types.DeprecatedServiceDirectorySettings{
						ProjectName:   "old",
						DefaultRegion: "old",
					},
				},
				MonitorNamespacesByDefault: true,
				Service: types.ServiceSettings{
					Annotations: []string{"one", "two"},
				},
				ServiceRegistrySettings: &types.ServiceRegistrySettings{
					ServiceDirectorySettings: &types.ServiceDirectorySettings{
						ProjectID:     "new",
						DefaultRegion: "new",
					},
				},
			},
			expRes: &types.Settings{
				MonitorNamespacesByDefault: true,
				Service: types.ServiceSettings{
					Annotations: []string{"one", "two"},
				},
				ServiceRegistrySettings: &types.ServiceRegistrySettings{
					ServiceDirectorySettings: &types.ServiceDirectorySettings{
						ProjectID:     "new",
						DefaultRegion: "new",
					},
				},
			},
		},
		{
			id: "successful-with-cloud-cfg",
			arg: &types.Settings{
				Gcloud: &types.GcloudSettings{
					ServiceDirectory: &types.DeprecatedServiceDirectorySettings{
						ProjectName:   "old",
						DefaultRegion: "old",
					},
				},
				Service: types.ServiceSettings{
					Annotations: []string{"one", "two"},
				},
				ServiceRegistrySettings: &types.ServiceRegistrySettings{
					ServiceDirectorySettings: &types.ServiceDirectorySettings{
						ProjectID:     "new",
						DefaultRegion: "new",
					},
				},
				CloudMetadata: &types.CloudMetadata{
					SubNetwork: func() *string {
						test := "subnetwork"
						return &test
					}(),
					Network: func() *string {
						test := "network"
						return &test
					}(),
				},
			},
			expRes: &types.Settings{
				Service: types.ServiceSettings{
					Annotations: []string{"one", "two"},
				},
				ServiceRegistrySettings: &types.ServiceRegistrySettings{
					ServiceDirectorySettings: &types.ServiceDirectorySettings{
						ProjectID:     "new",
						DefaultRegion: "new",
					},
				},
				CloudMetadata: func() *types.CloudMetadata {
					nname := "network"
					snname := "subnetwork"
					return &types.CloudMetadata{
						SubNetwork: &snname,
						Network:    &nname,
					}
				}(),
			},
		},
		{
			id: "successful-with-empty-cloud-cfg",
			arg: &types.Settings{
				Gcloud: &types.GcloudSettings{
					ServiceDirectory: &types.DeprecatedServiceDirectorySettings{
						ProjectName:   "old",
						DefaultRegion: "old",
					},
				},
				Service: types.ServiceSettings{
					Annotations: []string{"one", "two"},
				},
				ServiceRegistrySettings: &types.ServiceRegistrySettings{
					ServiceDirectorySettings: &types.ServiceDirectorySettings{
						ProjectID:     "new",
						DefaultRegion: "new",
					},
				},
				CloudMetadata: &types.CloudMetadata{},
			},
			expRes: &types.Settings{
				Service: types.ServiceSettings{
					Annotations: []string{"one", "two"},
				},
				ServiceRegistrySettings: &types.ServiceRegistrySettings{
					ServiceDirectorySettings: &types.ServiceDirectorySettings{
						ProjectID:     "new",
						DefaultRegion: "new",
					},
				},
				CloudMetadata: nil,
			},
		},
	}

	for _, currCase := range cases {
		res, err := ParseAndValidateSettings(currCase.arg)

		if currCase.expRes != nil {
			if !a.NotNil(res) {
				a.FailNow(fmt.Sprintf("case %s failed", currCase.id))
			}

			if currCase.expRes.ServiceRegistrySettings != nil {
				if !a.NotNil(res.ServiceRegistrySettings) {
					a.FailNow(fmt.Sprintf("case %s failed", currCase.id))
				}

				if currCase.expRes.EtcdSettings != nil {
					if !a.Equal(*currCase.expRes.EtcdSettings, *res.EtcdSettings) {
						a.FailNow(fmt.Sprintf("case %s failed", currCase.id))
					}
				}

				if currCase.expRes.ServiceDirectorySettings != nil {
					if !a.Equal(*currCase.expRes.ServiceDirectorySettings, *res.ServiceDirectorySettings) {
						a.FailNow(fmt.Sprintf("case %s failed", currCase.id))
					}
				}
			}
		}

		if !a.Equal(currCase.expErr, err) {
			a.FailNow(fmt.Sprintf("case %s failed", currCase.id))
		} else {
			continue
		}

		if !a.Equal(currCase.expRes, currCase.arg) {
			a.FailNow(fmt.Sprintf("case %s failed", currCase.id))
		}
	}
}
