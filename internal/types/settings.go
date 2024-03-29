// Copyright © 2020, 2021 Cisco
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

package types

// Settings of the application
type Settings struct {
	WatchNamespacesByDefault bool            `yaml:"watchNamespacesByDefault"`
	Service                  ServiceSettings `yaml:",inline"`
	*ServiceRegistrySettings `yaml:"serviceRegistry"`
	CloudMetadata            *CloudMetadata `yaml:"cloudMetadata"`
}

// ServiceSettings includes settings about services
type ServiceSettings struct {
	Annotations []string `yaml:"serviceAnnotations"`
}

// ServiceRegistrySettings contains information about the service registry
// that must be used, i.e. etcd or service directory.
type ServiceRegistrySettings struct {
	*ServiceDirectorySettings `yaml:"gcpServiceDirectory"`
	*EtcdSettings             `yaml:"etcd"`
	*CloudMapSettings         `yaml:"awsCloudMap"`
}

// ServiceDirectorySettings holds settings about gcloud service directory
type ServiceDirectorySettings struct {
	// DefaultRegion is the default region where objects will be registered to
	// in case the region is not mentioned explicitly.
	DefaultRegion string `yaml:"defaultRegion"`
	// ProjectID is the ID of the gcp project as it appears on google cloud
	// console.
	ProjectID string `yaml:"projectID"`
}

// EtcdAuthenticationType specifies how the cnwan operator must authenticate to
// the etcd cluster
type EtcdAuthenticationType string

const (
	// EtcdAuthWithNothing specifies that no authentication must be
	// performed.
	EtcdAuthWithNothing EtcdAuthenticationType = ""
	// EtcdAuthWithUsernamePassw specifies that authentication needs to be done
	// with username and password.
	EtcdAuthWithUsernamePassw EtcdAuthenticationType = "WithUsernameAndPassword"
	// EtcdAuthWithTLS specifies that authentication must be done with TLS.
	EtcdAuthWithTLS EtcdAuthenticationType = "WithTLS"
)

// EtcdSettings holds settings about etcd
type EtcdSettings struct {
	Authentication EtcdAuthenticationType `yaml:"authentication,omitempty"`
	Prefix         *string                `yaml:"prefix,omitempty"`
	Endpoints      []*EtcdEndpoint        `yaml:"endpoints"`
}

// EtcdEndpoint specifies an endpoint where to connect to.
// The port is a pointer, because if it is not specifies the well-known etcd
// port assigned from IANA will be used instead.
type EtcdEndpoint struct {
	Host string `yaml:"host"`
	Port *int   `yaml:"port"`
}

// CloudMetadata contains data and configuration about the cloud provider
// that is hosting the cluster, if any.
type CloudMetadata struct {
	// Network name
	Network *string `yaml:"network"`
	// SubNetwork name
	SubNetwork *string `yaml:"subNetwork"`
}

// CloudMapSettings contains data and configuration about AWS Cloud Map.
type CloudMapSettings struct {
	// DefaultRegion is the region where services will be registered.
	DefaultRegion string `yaml:"defaultRegion"`
	// TODO: support a different profile?
}
