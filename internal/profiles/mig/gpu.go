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

package mig

import (
	"fmt"

	resourceapi "k8s.io/api/resource/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/dynamic-resource-allocation/resourceslice"
	"k8s.io/utils/ptr"
	cdiapi "tags.cncf.io/container-device-interface/pkg/cdi"
	cdispec "tags.cncf.io/container-device-interface/specs-go"

	configapi "sigs.k8s.io/dra-example-driver/api/example.com/resource/gpu/v1alpha1"
	"sigs.k8s.io/dra-example-driver/internal/profiles"
	"sigs.k8s.io/dra-example-driver/internal/profiles/helpers"
)

const ProfileName = "mig"

type Profile struct {
	nodeName string
	numGPUs  int
}

func NewProfile(nodeName string, numGPUs int) Profile {
	return Profile{
		nodeName: nodeName,
		numGPUs:  numGPUs,
	}
}

func (p Profile) EnumerateDevices() (resourceslice.DriverResources, error) {
	profiles := map[string]struct {
		resourceapi.Device
		Count int
	}{
		"gpu-%d": {
			resourceapi.Device{
				Attributes: map[resourceapi.QualifiedName]resourceapi.DeviceAttribute{
					"profile": {
						StringValue: ptr.To(""),
					},
				},
				ConsumesCounters: []resourceapi.DeviceCounterConsumption{
					{
						CounterSet: "gpu-0-counter-set",
						Counters: map[string]resourceapi.Counter{
							"compute": resourceapi.Counter{
								Value: resource.MustParse("1"),
							},
							"memory": resourceapi.Counter{
								Value: resource.MustParse("10Gi"),
							},
						},
					},
				},
			}, 1,
		},
		"gpu-%d-mig-1g10gb-%d": {
			resourceapi.Device{
				Attributes: map[resourceapi.QualifiedName]resourceapi.DeviceAttribute{
					"profile": {
						StringValue: ptr.To("1g.10gb"),
					},
				},
				ConsumesCounters: []resourceapi.DeviceCounterConsumption{
					{
						CounterSet: "gpu-0-counter-set",
						Counters: map[string]resourceapi.Counter{
							"compute": resourceapi.Counter{
								Value: resource.MustParse("1"),
							},
							"memory": resourceapi.Counter{
								Value: resource.MustParse("10Gi"),
							},
						},
					},
				},
			}, 7,
		},
		"gpu-%d-mig-2g10gb-%d": {
			resourceapi.Device{
				Attributes: map[resourceapi.QualifiedName]resourceapi.DeviceAttribute{
					"profile": {
						StringValue: ptr.To("2g.10gb"),
					},
				},
				ConsumesCounters: []resourceapi.DeviceCounterConsumption{
					{
						CounterSet: "gpu-0-counter-set",
						Counters: map[string]resourceapi.Counter{
							"compute": resourceapi.Counter{
								Value: resource.MustParse("2"),
							},
							"memory": resourceapi.Counter{
								Value: resource.MustParse("10Gi"),
							},
						},
					},
				},
			}, 3,
		},
		"gpu-%d-mig-3g20gb-%d": {
			resourceapi.Device{
				Attributes: map[resourceapi.QualifiedName]resourceapi.DeviceAttribute{
					"profile": {
						StringValue: ptr.To("3g.20gb"),
					},
				},
				ConsumesCounters: []resourceapi.DeviceCounterConsumption{
					{
						CounterSet: "gpu-0-counter-set",
						Counters: map[string]resourceapi.Counter{
							"compute": resourceapi.Counter{
								Value: resource.MustParse("3"),
							},
							"memory": resourceapi.Counter{
								Value: resource.MustParse("20Gi"),
							},
						},
					},
				},
			}, 1,
		},
		"gpu-%d-mig-4g20gb-%d": {
			resourceapi.Device{
				Attributes: map[resourceapi.QualifiedName]resourceapi.DeviceAttribute{
					"profile": {
						StringValue: ptr.To("4g.20gb"),
					},
				},
				ConsumesCounters: []resourceapi.DeviceCounterConsumption{
					{
						CounterSet: "gpu-0-counter-set",
						Counters: map[string]resourceapi.Counter{
							"compute": resourceapi.Counter{
								Value: resource.MustParse("4"),
							},
							"memory": resourceapi.Counter{
								Value: resource.MustParse("20Gi"),
							},
						},
					},
				},
			}, 1,
		},
	}
	seed := p.nodeName
	uuids := helpers.GenerateUUIDs(seed, "gpu", p.numGPUs)

	var devices []resourceapi.Device
	for i, uuid := range uuids {
		for namefmt, profileCount := range profiles {
			for j := range profileCount.Count {
				device := profileCount.Device.DeepCopy()
				if namefmt == "gpu-%d" {
					device.Name = fmt.Sprintf(namefmt, i)
				} else {
					device.Name = fmt.Sprintf(namefmt, i, j)
				}
				device.Attributes["index"] = resourceapi.DeviceAttribute{
					IntValue: ptr.To(int64(i)),
				}
				device.Attributes["uuid"] = resourceapi.DeviceAttribute{
					StringValue: ptr.To(uuid),
				}
				device.Attributes["model"] = resourceapi.DeviceAttribute{
					StringValue: ptr.To("MIG-SUPPROT-GPU-MODEL"),
				}
				device.Attributes["driverVersion"] = resourceapi.DeviceAttribute{
					VersionValue: ptr.To("1.0.0"),
				}
				devices = append(devices, *device)
			}
		}
	}

	resources := resourceslice.DriverResources{
		Pools: map[string]resourceslice.Pool{
			p.nodeName: {
				Slices: []resourceslice.Slice{
					{
						SharedCounters: []resourceapi.CounterSet{
							{
								Name: "gpu-0-counter-set",
								Counters: map[string]resourceapi.Counter{
									"compute": resourceapi.Counter{
										Value: resource.MustParse("8"),
									},
									"memory": resourceapi.Counter{
										Value: resource.MustParse("40Gi"),
									},
								},
							},
						},
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

// DefaultSetup sets mig profile to environmental variables.
func (p Profile) DefaultSetup(results []resourceapi.DeviceRequestAllocationResult) (profiles.PerDeviceCDIContainerEdits, error) {
	perDeviceEdits := make(profiles.PerDeviceCDIContainerEdits)

	for _, result := range results {
		envs := []string{
			fmt.Sprintf("GPU_DEVICE_%s=%s", result.Device[4:], result.Device),
		}

		envs = append(envs, fmt.Sprintf("GPU_DEVICE_MIG_PROFILE=%s.%s", result.Device[10:12], result.Device[12:16]))

		edits := &cdispec.ContainerEdits{
			Env: envs,
		}

		perDeviceEdits[result.Device] = &cdiapi.ContainerEdits{ContainerEdits: edits}
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
