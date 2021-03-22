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

package types

const (
	// SDDefaultRegion is the key for service directory region setting
	SDDefaultRegion = "gcloud.servicedirectory.region"
	// SDProject is the key for service directory project setting
	SDProject = "gcloud.servicedirectory.project"
	// NamespaceListPolicy is the key for the namespace list policy setting
	NamespaceListPolicy = "namespace.listpolicy"
	// AllowedAnnotations is the key for the allowed annotations setting
	AllowedAnnotations = "service.annotations"
	// AllowedAnnotationsMap is the key for the allowed annotations map setting
	AllowedAnnotationsMap = "service.annotationsmap"
	// ServiceRegistrySettingsKey is the key for the service registry settings
	ServiceRegistrySettingsKey = "serviceregistry"
	// DeprecatedGcloudServiceDirectoryKey is the key for the old service
	// directory. It is currently only used to check if it is there and warn
	// the user that it is deprecated.
	DeprecatedGcloudServiceDirectoryKey = "gcloud.servicedirectory"
)

// Settings of the application
type Settings struct {
	Namespace                NamespaceSettings `yaml:"namespace"`
	Service                  ServiceSettings   `yaml:"service"`
	*ServiceRegistrySettings `yaml:"serviceRegistry"`

	// DEPRECATED: include this under serviceRegistry instead of here.
	// TODO: remove this on v0.6.0
	Gcloud *GcloudSettings `yaml:"gcloud"`
}

// GcloudSettings holds gcloud settings
// TODO: remove this on v0.6.0
type GcloudSettings struct {
	ServiceDirectory *DeprecatedServiceDirectorySettings `yaml:"serviceDirectory"`
}

// ListPolicy is the list type that must be adopted by the operator
type ListPolicy string

const (
	// AllowList will make the operator only consider resources that have
	// are in the allowlist
	AllowList ListPolicy = "allowlist"
	// AllowedKey is the label key that states that a specific resource is
	// in the allowlist, if allowlist is the current policy type.
	// If the policy type is blocklist, this key is ignored.
	AllowedKey string = "operator.cnwan.io/allowed"
	// BlockList will make the operator consider all resources and ignore
	// those that are in the blocklist
	BlockList ListPolicy = "blocklist"
	// BlockedKey is the label key that states that a specific resource is
	// in the blocklist, if blocklist is the current policy type.
	// If the policy type is allowlist, this key is ignored.
	BlockedKey string = "operator.cnwan.io/blocked"
)

// NamespaceSettings includes settings about namespaces
type NamespaceSettings struct {
	ListPolicy ListPolicy `yaml:"listPolicy"`
}

// ServiceSettings includes settings about services
type ServiceSettings struct {
	Annotations []string `yaml:"annotations"`
}

// ServiceRegistrySettings contains information about the service registry
// that must be used, i.e. etcd or service directory.
type ServiceRegistrySettings struct {
	*ServiceDirectorySettings `yaml:"gcpServiceDirectory"`
	*EtcdSettings             `yaml:"etcd"`
}

// DeprecatedServiceDirectorySettings holds settings about gcloud service directory
// TODO: remove this on v0.6.0
type DeprecatedServiceDirectorySettings struct {
	DefaultRegion string `yaml:"region"`
	ProjectName   string `yaml:"project"`
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
