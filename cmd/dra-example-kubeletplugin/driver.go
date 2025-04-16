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
	"context"
	"sync"

	resourceapi "k8s.io/api/resource/v1beta1"
	"k8s.io/apimachinery/pkg/types"
	coreclientset "k8s.io/client-go/kubernetes"
	"k8s.io/dynamic-resource-allocation/kubeletplugin"
	"k8s.io/dynamic-resource-allocation/resourceslice"

	"sigs.k8s.io/dra-example-driver/pkg/consts"
)

var _ kubeletplugin.DRAPlugin = &driver{}

type driver struct {
	client coreclientset.Interface
	plugin *kubeletplugin.Helper
	state  *DeviceState

	prepareResourcesFailure   error
	failPrepareResourcesMutex sync.Mutex

	unprepareResourcesFailure   error
	failUnprepareResourcesMutex sync.Mutex
}

func NewDriver(ctx context.Context, config *Config) (*driver, error) {
	driver := &driver{
		client: config.coreclient,
	}

	state, err := NewDeviceState(config)
	if err != nil {
		return nil, err
	}
	driver.state = state

	plugin, err := kubeletplugin.Start(
		ctx,
		driver,
		kubeletplugin.KubeClient(config.coreclient),
		kubeletplugin.NodeName(config.flags.nodeName),
		kubeletplugin.DriverName(consts.DriverName),
	)
	if err != nil {
		return nil, err
	}
	driver.plugin = plugin
	var devices []resourceapi.Device
	for _, device := range state.allocatable {
		devices = append(devices, device)
	}
	var resources resourceslice.DriverResources
	resources.Pools = map[string]resourceslice.Pool{
		config.flags.nodeName: resourceslice.Pool{
			Slices: []resourceslice.Slice{{
				Devices: devices,
			},
			},
		},
	}
	if err := plugin.PublishResources(ctx, resources); err != nil {
		return nil, err
	}

	return driver, nil
}

func (d *driver) Shutdown(ctx context.Context) error {
	d.plugin.Stop()
	return nil
}

func (d *driver) PrepareResourceClaims(ctx context.Context, claims []*resourceapi.ResourceClaim) (result map[types.UID]kubeletplugin.PrepareResult, err error) {
	if failure := d.getPrepareResourcesFailure(); failure != nil {
		return nil, failure
	}

	result = make(map[types.UID]kubeletplugin.PrepareResult)
	for _, claim := range claims {
		devices, err := d.state.Prepare(claim)
		var claimResult kubeletplugin.PrepareResult
		if err != nil {
			claimResult.Err = err
		} else {
			claimResult.Devices = devices
		}
		result[claim.UID] = claimResult
	}
	return result, nil
}

func (d *driver) UnprepareResourceClaims(ctx context.Context, claims []kubeletplugin.NamespacedObject) (result map[types.UID]error, err error) {
	result = make(map[types.UID]error)

	if failure := d.getUnprepareResourcesFailure(); failure != nil {
		return nil, failure
	}

	for _, claimRef := range claims {
		uid := string(claimRef.UID)
		err := d.state.Unprepare(uid)
		result[claimRef.UID] = err
	}
	return result, nil
}

func (d *driver) getPrepareResourcesFailure() error {
	d.failPrepareResourcesMutex.Lock()
	defer d.failPrepareResourcesMutex.Unlock()
	return d.prepareResourcesFailure
}

func (d *driver) getUnprepareResourcesFailure() error {
	d.failUnprepareResourcesMutex.Lock()
	defer d.failUnprepareResourcesMutex.Unlock()
	return d.unprepareResourcesFailure
}
