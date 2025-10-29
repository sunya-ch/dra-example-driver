/*
 * Copyright 2023 The Kubernetes Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/dra-example-driver/pkg/config"
)

var (
	mockResourceSlicePath = "../../test/mock/slice-config.yaml"
)

func TestEnumerateAllPossibleDevices(t *testing.T) {
	originalConfigPath := config.ResourceSliceConfigPath
	config.ResourceSliceConfigPath = mockResourceSlicePath
	rsConfig, err := config.GetResourceSliceConfig()
	fmt.Println(rsConfig)
	require.NoError(t, err)
	allocatableDevices, err := enumerateAllPossibleDevices(rsConfig, 2)
	require.NoError(t, err)
	assert.Len(t, allocatableDevices, 2)
	for name, device := range allocatableDevices {
		fmt.Println(name, device.Attributes)
		assert.Contains(t, name, rsConfig.Prefix)
		assert.NotNil(t, device.AllowMultipleAllocations)
		assert.True(t, *device.AllowMultipleAllocations)
		assert.Len(t, device.Capacity, 2)
		if name == rsConfig.Prefix+"1" {
			assert.Len(t, device.Attributes, 6)
		} else {
			assert.Len(t, device.Attributes, 5)
		}
	}
	config.ResourceSliceConfigPath = originalConfigPath
}
