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

package controllers

import (
	"sync"

	ktypes "k8s.io/apimachinery/pkg/types"
)

type counter struct {
	lock   sync.RWMutex
	counts map[string]map[string]int
}

func newCounter() *counter {
	return &counter{
		counts: map[string]map[string]int{},
	}
}

func (c *counter) getSrvCount(nsName, srvName string) int {
	c.lock.RLock()
	defer c.lock.RUnlock()

	fullName := ktypes.NamespacedName{Namespace: nsName, Name: srvName}.String()
	if _, exists := c.counts[fullName]; !exists {
		return 0
	}

	totalCount := 0
	for _, c := range c.counts[fullName] {
		totalCount += c
	}

	return totalCount
}

func (c *counter) resetCounterTo(nsName, srvName string, data map[string]int) {
	c.lock.Lock()
	defer c.lock.Unlock()

	fullName := ktypes.NamespacedName{Namespace: nsName, Name: srvName}.String()
	c.counts[fullName] = data
}

func (c *counter) getSrvData(nsName, srvName string) map[string]int {
	c.lock.RLock()
	defer c.lock.RUnlock()

	fullName := ktypes.NamespacedName{Namespace: nsName, Name: srvName}.String()
	val, exists := c.counts[fullName]
	if !exists {
		return map[string]int{}
	}

	return val
}

func (c *counter) putSrvCount(nsName, srvName, epSliceName string, count int) {
	c.lock.Lock()
	defer c.lock.Unlock()

	fullName := ktypes.NamespacedName{Namespace: nsName, Name: srvName}.String()
	if _, exists := c.counts[fullName]; !exists {
		c.counts[fullName] = map[string]int{}
	}

	c.counts[fullName][epSliceName] = count
}
