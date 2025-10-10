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
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/ptr"

	"github.com/google/uuid"
)

func enumerateAllPossibleDevices(numGPUs int) (AllocatableDevices, error) {
	seed := os.Getenv("NODE_NAME")
	uuids := generateUUIDs(seed, numGPUs)

	alldevices := make(AllocatableDevices)
	for i, uuid := range uuids {
		device := resourceapi.Device{
			Name:                     fmt.Sprintf("gpu-%d", i),
			AllowMultipleAllocations: ptr.To(true),
			Attributes: map[resourceapi.QualifiedName]resourceapi.DeviceAttribute{
				"index": {
					IntValue: ptr.To(int64(i)),
				},
				"uuid": {
					StringValue: ptr.To(uuid),
				},
				"model": {
					StringValue: ptr.To("LATEST-GPU-MODEL"),
				},
				"driverVersion": {
					VersionValue: ptr.To("1.0.0"),
				},
			},
			Capacity: map[resourceapi.QualifiedName]resourceapi.DeviceCapacity{
				"memory": {
					Value: resource.MustParse("80Gi"), // MPS - defaultPinnedDeviceMemoryLimit
					RequestPolicy: &resourceapi.CapacityRequestPolicy{
						Default: ptr.To(resource.MustParse("80Gi")),
						ValidRange: &resourceapi.CapacityRequestPolicyRange{
							Min:  ptr.To(resource.MustParse("1Gi")),
							Step: ptr.To(resource.MustParse("512Mi")),
						},
					},
				},
				"thread": {
					Value: resource.MustParse("100"), // MPS - defaultActiveThreadPercentage
					RequestPolicy: &resourceapi.CapacityRequestPolicy{
						Default: ptr.To(resource.MustParse("100")),
						ValidRange: &resourceapi.CapacityRequestPolicyRange{
							Min:  ptr.To(resource.MustParse("1")),
							Step: ptr.To(resource.MustParse("1")),
						},
					},
				},
				// count is used for limit device to be allocated by a specific request only once
				// the other request can still request 0 to share.
				"count": {
					Value: resource.MustParse("1"), // MPS - defaultActiveThreadPercentage
					RequestPolicy: &resourceapi.CapacityRequestPolicy{
						Default: ptr.To(resource.MustParse("0")),
						ValidValues: []resource.Quantity{
							resource.MustParse("0"),
							resource.MustParse("1"),
						},
					},
				},
			},
		}
		alldevices[device.Name] = device
	}
	return alldevices, nil
}

func generateUUIDs(seed string, count int) []string {
	rand := rand.New(rand.NewSource(hash(seed)))

	uuids := make([]string, count)
	for i := 0; i < count; i++ {
		charset := make([]byte, 16)
		rand.Read(charset)
		uuid, _ := uuid.FromBytes(charset)
		uuids[i] = "gpu-" + uuid.String()
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
