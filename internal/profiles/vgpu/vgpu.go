/*
 * Copyright The Kubernetes Authors.
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

package vgpu

import (
	"fmt"

	resourceapi "k8s.io/api/resource/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/dynamic-resource-allocation/resourceslice"
	"k8s.io/klog/v2"
	"k8s.io/utils/ptr"
	cdiapi "tags.cncf.io/container-device-interface/pkg/cdi"
	cdispec "tags.cncf.io/container-device-interface/specs-go"

	configapi "sigs.k8s.io/dra-example-driver/api/example.com/resource/gpu/v1alpha1"
	"sigs.k8s.io/dra-example-driver/internal/profiles"
	"sigs.k8s.io/dra-example-driver/internal/profiles/helpers"
)

const ProfileName = "vgpu"

type Profile struct {
	nodeName  string
	numGPUs   int
	modelName string
}

func NewProfile(nodeName string, numGPUs int, modelName string) Profile {
	return Profile{
		nodeName:  nodeName,
		numGPUs:   numGPUs,
		modelName: modelName,
	}
}

func (p Profile) EnumerateDevices() (resourceslice.DriverResources, error) {
	seed := p.nodeName
	uuids := helpers.GenerateUUIDs(seed, "gpu", p.numGPUs)

	var devices []resourceapi.Device
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
					StringValue: ptr.To(p.modelName),
				},
				"driverVersion": {
					VersionValue: ptr.To("1.0.0"),
				},
			},
			Capacity: map[resourceapi.QualifiedName]resourceapi.DeviceCapacity{
				"compute": {
					Value: resource.MustParse("100"),
					RequestPolicy: &resourceapi.CapacityRequestPolicy{
						Default: ptr.To(resource.MustParse("10")),
						ValidRange: &resourceapi.CapacityRequestPolicyRange{
							Min:  ptr.To(resource.MustParse("10")),
							Step: ptr.To(resource.MustParse("10")),
						},
					},
				},
				"memory": {
					Value: resource.MustParse("80Gi"),
					RequestPolicy: &resourceapi.CapacityRequestPolicy{
						Default: ptr.To(resource.MustParse("4Gi")),
						ValidRange: &resourceapi.CapacityRequestPolicyRange{
							Min:  ptr.To(resource.MustParse("4Gi")),
							Step: ptr.To(resource.MustParse("4Gi")),
						},
					},
				},
			},
		}
		devices = append(devices, device)
	}

	resources := resourceslice.DriverResources{
		Pools: map[string]resourceslice.Pool{
			p.nodeName: {
				Slices: []resourceslice.Slice{
					{
						Devices: devices,
					},
				},
			},
		},
	}

	return resources, nil
}

// SchemeBuilder implements [profiles.ConfigHandler].
func (p Profile) SchemeBuilder() runtime.SchemeBuilder {
	return runtime.NewSchemeBuilder(
		configapi.AddToScheme,
	)
}

// Validate implements [profiles.ConfigHandler].
func (p Profile) Validate(config runtime.Object) error {
	gpuConfig, ok := config.(*configapi.GpuConfig)
	if !ok {
		return fmt.Errorf("expected v1alpha1.GpuConfig but got: %T", config)
	}
	return gpuConfig.Validate()
}

// DefaultSetup sets MPS environmental variables.
func (p Profile) DefaultSetup(results []resourceapi.DeviceRequestAllocationResult) (profiles.PerDeviceCDIContainerEdits, error) {
	perDeviceEdits := make(profiles.PerDeviceCDIContainerEdits)

	for _, result := range results {
		shareId := (*string)(result.ShareID)
		deviceId := helpers.GetCDIDeviceID(result.Device, shareId)
		gpuIndex := result.Device[4:]

		envs := []string{
			fmt.Sprintf("GPU_MODEL_NAME_%s=%s", gpuIndex, p.modelName),
			fmt.Sprintf("GPU_DEVICE_%s=%s", gpuIndex, result.Device),
		}
		if computePercent, found := result.ConsumedCapacity[resourceapi.QualifiedName("compute")]; found {
			envs = append(envs, fmt.Sprintf("GPU_DEVICE_%s_ACTIVE_THREAD_PERCENTAGE=%s", gpuIndex, computePercent.String()))
		} else {
			return nil, fmt.Errorf("error setting GPU sharing: no consumed compute capacity")
		}
		if memory, found := result.ConsumedCapacity[resourceapi.QualifiedName("memory")]; found {
			envs = append(envs, fmt.Sprintf("GPU_DEVICE_%s_MEMORY_LIMIT=%v", gpuIndex, memory.String()))
		} else {
			return nil, fmt.Errorf("error setting GPU sharing: no consumed compute capacity")
		}

		edits := &cdispec.ContainerEdits{
			Env: envs,
		}

		klog.Background().Info("default setup env", "envs", envs)

		perDeviceEdits[deviceId] = &cdiapi.ContainerEdits{ContainerEdits: edits}
	}

	return perDeviceEdits, nil
}

// ApplyConfig implements [profiles.ConfigHandler].
func (p Profile) ApplyConfig(config runtime.Object, results []*resourceapi.DeviceRequestAllocationResult) (profiles.PerDeviceCDIContainerEdits, error) {
	if config == nil {
		config = configapi.DefaultGpuConfig()
	}
	if config, ok := config.(*configapi.GpuConfig); ok {
		return applyGpuConfig(config, results)
	}
	return nil, fmt.Errorf("runtime object is not a recognized configuration")
}

// In this example driver there is no actual configuration applied. We simply
// define a set of environment variables to be injected into the containers
// that include a given device. A real driver would likely need to do some sort
// of hardware configuration as well, based on the config passed in.
func applyGpuConfig(config *configapi.GpuConfig, results []*resourceapi.DeviceRequestAllocationResult) (profiles.PerDeviceCDIContainerEdits, error) {
	perDeviceEdits := make(profiles.PerDeviceCDIContainerEdits)

	// Normalize the config to set any implied defaults.
	if err := config.Normalize(); err != nil {
		return nil, fmt.Errorf("error normalizing GPU config: %w", err)
	}

	// Validate the config to ensure its integrity.
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("error validating GPU config: %w", err)
	}

	return perDeviceEdits, nil
}
