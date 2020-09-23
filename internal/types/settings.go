// Copyright © 2020 Cisco
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
)

// Settings of the application
type Settings struct {
	Namespace NamespaceSettings `yaml:"namespace"`
	Gcloud    *GcloudSettings   `yaml:"gcloud"`
}

// GcloudSettings holds gcloud settings
type GcloudSettings struct {
	ServiceDirectory *ServiceDirectorySettings `yaml:"serviceDirectory"`
}

// ServiceDirectorySettings holds settings about gcloud service directory
type ServiceDirectorySettings struct {
	DefaultRegion string `yaml:"region"`
	ProjectName   string `yaml:"project"`
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
