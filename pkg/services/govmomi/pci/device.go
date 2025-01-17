/*
Copyright 2023 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package pci

import (
	"fmt"

	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
	"golang.org/x/net/context"

	infrav1 "sigs.k8s.io/cluster-api-provider-vsphere/apis/v1beta1"
)

func CalculateDevicesToBeAdded(ctx context.Context, vm *object.VirtualMachine, deviceSpecs []infrav1.PCIDeviceSpec) ([]infrav1.PCIDeviceSpec, error) {
	// store the number of expected devices for each deviceID + vendorID combo
	deviceVendorIDComboMap := map[string]int{}
	for _, spec := range deviceSpecs {
		key := constructKey(spec)
		if _, ok := deviceVendorIDComboMap[key]; !ok {
			deviceVendorIDComboMap[key] = 1
		} else {
			deviceVendorIDComboMap[key]++
		}
	}

	devices, err := vm.Device(ctx)
	if err != nil {
		return nil, err
	}

	specsToBeAdded := []infrav1.PCIDeviceSpec{}
	for _, spec := range deviceSpecs {
		key := constructKey(spec)
		pciDeviceList := devices.SelectByBackingInfo(createBackingInfo(spec))
		expectedDeviceLen := deviceVendorIDComboMap[key]
		if expectedDeviceLen-len(pciDeviceList) > 0 {
			specsToBeAdded = append(specsToBeAdded, spec)
			deviceVendorIDComboMap[key]--
		}
	}
	return specsToBeAdded, nil
}

func ConstructDeviceSpecs(pciDeviceSpecs []infrav1.PCIDeviceSpec) []types.BaseVirtualDevice {
	pciDevices := []types.BaseVirtualDevice{}
	deviceKey := int32(-200)

	for _, pciDevice := range pciDeviceSpecs {
		backingInfo := createBackingInfo(pciDevice)
		pciDevices = append(pciDevices, &types.VirtualPCIPassthrough{
			VirtualDevice: types.VirtualDevice{
				Key:     deviceKey,
				Backing: backingInfo,
			},
		})
		deviceKey--
	}
	return pciDevices
}

func createBackingInfo(spec infrav1.PCIDeviceSpec) *types.VirtualPCIPassthroughDynamicBackingInfo {
	return &types.VirtualPCIPassthroughDynamicBackingInfo{
		AllowedDevice: []types.VirtualPCIPassthroughAllowedDevice{
			{
				VendorId: *spec.VendorID,
				DeviceId: *spec.DeviceID,
			},
		},
	}
}

func constructKey(pciDeviceSpec infrav1.PCIDeviceSpec) string {
	return fmt.Sprintf("%d-%d", *pciDeviceSpec.DeviceID, *pciDeviceSpec.VendorID)
}
