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
	"math/rand"
	"os"

	resourceapi "k8s.io/api/resource/v1"
	"k8s.io/utils/ptr"

	"sigs.k8s.io/dra-example-driver/pkg/config"

	"github.com/google/uuid"
)

func enumerateAllPossibleDevices(rsConfig config.ResourceSliceConfig, numGPUs int) (AllocatableDevices, error) {
	seed := os.Getenv("NODE_NAME")
	uuids := generateUUIDs(rsConfig.Prefix, seed, numGPUs)
	alldevices := make(AllocatableDevices)
	fmt.Println("enumerateAllPossibleDevices", rsConfig)
	for i, uuid := range uuids {
		name := fmt.Sprintf("%s%d", rsConfig.Prefix, i)
		device := &resourceapi.Device{
			Name: name,
			Attributes: map[resourceapi.QualifiedName]resourceapi.DeviceAttribute{
				"index": {
					IntValue: ptr.To(int64(i)),
				},
				"uuid": {
					StringValue: ptr.To(uuid),
				},
				"name": {
					StringValue: ptr.To(name),
				},
			},
		}
		fmt.Println(">", uuid, name, rsConfig.Patch.Devices[name])
		applyDeviceConfig(device, rsConfig.Common)

		if patch, found := rsConfig.Patch.Devices[name]; found {
			applyDeviceConfig(device, patch)
		}
		alldevices[device.Name] = *device
	}
	return alldevices, nil
}

func generateUUIDs(prefix, seed string, count int) []string {
	rand := rand.New(rand.NewSource(hash(seed)))

	uuids := make([]string, count)
	for i := 0; i < count; i++ {
		charset := make([]byte, 16)
		rand.Read(charset)
		uuid, _ := uuid.FromBytes(charset)
		uuids[i] = prefix + uuid.String()
	}

	return uuids
}

func hash(s string) int64 {
	h := int64(0)
	for _, c := range s {
		h = 31*h + int64(c)
	}
	return h
}

func applyDeviceConfig(device *resourceapi.Device, deviceConfig config.Device) {
	if deviceConfig.AllowMultipleAllocations != nil {
		device.AllowMultipleAllocations = deviceConfig.AllowMultipleAllocations
	}
	if len(device.Attributes) > 0 && device.Attributes == nil {
		device.Attributes = make(map[resourceapi.QualifiedName]resourceapi.DeviceAttribute)
	}
	for name, attribute := range deviceConfig.Attributes {
		device.Attributes[resourceapi.QualifiedName(name)] = resourceapi.DeviceAttribute{
			IntValue:     attribute.IntValue,
			BoolValue:    attribute.BoolValue,
			StringValue:  attribute.StringValue,
			VersionValue: attribute.VersionValue,
		}
	}
	if len(deviceConfig.Capacity) > 0 && device.Capacity == nil {
		device.Capacity = make(map[resourceapi.QualifiedName]resourceapi.DeviceCapacity)
	}
	for name, capacity := range deviceConfig.Capacity {
		device.Capacity[resourceapi.QualifiedName(name)] = capacity.ConvertToResourceAPI()
	}
}
