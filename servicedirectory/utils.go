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

package servicedirectory

import (
	"path"
	"reflect"

	"github.com/CloudNativeSDWAN/cnwan-operator/types"
	"github.com/spf13/viper"
)

func (s *sdHandler) getResourcePath(resources ...string) string {
	projectName := viper.GetString(types.SDProject)
	defRegion := viper.GetString(types.SDDefaultRegion)
	res := path.Join("projects", projectName, "locations", defRegion)

	// Add the namespace
	if len(resources) > 0 {
		res = path.Join(res, "namespaces", resources[0])
	}

	// Add the service
	if len(resources) > 1 {
		res = path.Join(res, "services", resources[1])
	}

	// Add the endpoint
	if len(resources) > 2 {
		res = path.Join(res, "endpoints", resources[2])
	}

	return res
}

// deepEqualMetadata compares two metadata maps, ignoring reserved metadata
// without removing it from the maps
func (s *sdHandler) deepEqualMetadata(src, dst map[string]string) bool {
	// Copy the two
	sr := map[string]string{}
	de := map[string]string{}

	for key, val := range src {
		if key == "owner" && val == "cnwan-operator" {
			// Don't copy this one
			continue
		}
		sr[key] = val
	}

	for key, val := range dst {
		if key == "owner" && val == "cnwan-operator" {
			// Don't copy this one
			continue
		}
		de[key] = val
	}

	return reflect.DeepEqual(sr, de)
}
