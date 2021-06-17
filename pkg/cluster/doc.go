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

// Package cluster contains code that gets information about the cluster where
// we are running and if it is a managed cluster, e.g. GKE or EKS.
// For example: VPC, SubNetwork, Cluster Name, etc.
//
// Additionally, it also retrieves common objects from Kubernetes, e.g.:
// ConfigMap and Secrets.
package cluster
