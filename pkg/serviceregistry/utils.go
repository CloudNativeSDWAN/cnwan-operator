// Copyright © 2023 Cisco
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

package serviceregistry

import (
	serego "github.com/CloudNativeSDWAN/serego/api/core/types"
)

func getNamespaceNameFromEventObject(event *Event) string {
	switch parsedObject := event.Object.(type) {
	case *serego.Namespace:
		return parsedObject.Name
	case *serego.Service:
		return parsedObject.Namespace
	case *serego.Endpoint:
		return parsedObject.Namespace
	default:
		return ""
	}
}

func isOwnedByOperator(metadata map[string]string) bool {
	owned, exists := metadata["owner"]
	return exists && owned == "cnwan-operator"
}
