// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package ufs

import (
	"context"
	"fmt"
	"strings"

	proto "github.com/golang/protobuf/proto"
	"go.chromium.org/chromiumos/infra/proto/go/device"
	"go.chromium.org/chromiumos/infra/proto/go/lab"
	"go.chromium.org/chromiumos/infra/proto/go/manufacturing"
	"go.chromium.org/luci/common/logging"

	api "infra/appengine/cros/lab_inventory/api/v1"
	"infra/appengine/cros/lab_inventory/app/config"
	"infra/appengine/cros/lab_inventory/app/external"
	ufspb "infra/unifiedfleet/api/v1/models"
	ufschromeoslab "infra/unifiedfleet/api/v1/models/chromeos/lab"
	ufsapi "infra/unifiedfleet/api/v1/rpc"
	ufsutil "infra/unifiedfleet/app/util"
)

// GetUFSDevicesByIds Gets MachineLSEs from UFS by Asset id/Machine id.
func GetUFSDevicesByIds(ctx context.Context, ufsClient external.UFSClient, ids []string) ([]*lab.ChromeOSDevice, []*api.DeviceOpResult) {
	failedDevices := make([]*api.DeviceOpResult, 0, len(ids))
	var devices []*lab.ChromeOSDevice
	for _, id := range ids {
		machine, err := ufsClient.GetMachine(ctx, &ufsapi.GetMachineRequest{
			Name: ufsutil.AddPrefix(ufsutil.MachineCollection, id),
		})
		if err != nil {
			logging.Errorf(ctx, "Machine not found for machine ID %s", id)
			failedDevices = append(failedDevices, &api.DeviceOpResult{
				Id:       id,
				ErrorMsg: err.Error(),
			})
			continue
		}
		machine.Name = ufsutil.RemovePrefix(machine.Name)
		req := &ufsapi.ListMachineLSEsRequest{
			PageSize: 1,
			Filter:   fmt.Sprintf("machine=%s", id),
		}
		res, err := ufsClient.ListMachineLSEs(ctx, req)
		if err != nil || len(res.GetMachineLSEs()) == 0 {
			logging.Errorf(ctx, "MachineLSE not found for machine ID %s", id)
			failedDevices = append(failedDevices, &api.DeviceOpResult{
				Id:       id,
				ErrorMsg: err.Error(),
			})
			continue
		}
		lse := res.GetMachineLSEs()[0]
		lse.Name = ufsutil.RemovePrefix(lse.Name)
		devices = append(devices, ConstructInvV2Device(machine, lse))
	}
	return devices, failedDevices
}

// GetUFSDevicesByHostnames Gets MachineLSEs from UFS by MachineLSE name/hostname.
func GetUFSDevicesByHostnames(ctx context.Context, ufsClient external.UFSClient, names []string) ([]*lab.ChromeOSDevice, []*api.DeviceOpResult) {
	failedDevices := make([]*api.DeviceOpResult, 0, len(names))
	var devices []*lab.ChromeOSDevice
	for _, name := range names {
		lse, err := ufsClient.GetMachineLSE(ctx, &ufsapi.GetMachineLSERequest{
			Name: ufsutil.AddPrefix(ufsutil.MachineLSECollection, name),
		})
		if err != nil {
			logging.Errorf(ctx, "MachineLSE not found for hostname %s", name)
			failedDevices = append(failedDevices, &api.DeviceOpResult{
				Hostname: name,
				ErrorMsg: err.Error(),
			})
			continue
		}
		lse.Name = ufsutil.RemovePrefix(lse.Name)

		if len(lse.GetMachines()) == 0 {
			logging.Errorf(ctx, "Machine not found for hostname %s", name)
			failedDevices = append(failedDevices, &api.DeviceOpResult{
				Hostname: lse.GetName(),
				ErrorMsg: err.Error(),
			})
			continue
		}
		machine, err := ufsClient.GetMachine(ctx, &ufsapi.GetMachineRequest{
			Name: ufsutil.AddPrefix(ufsutil.MachineCollection, lse.GetMachines()[0]),
		})
		if err != nil {
			logging.Errorf(ctx, "Machine not found for machine ID %s", lse.GetMachines()[0])
			failedDevices = append(failedDevices, &api.DeviceOpResult{
				Id:       lse.GetMachines()[0],
				Hostname: lse.GetName(),
				ErrorMsg: err.Error(),
			})
			continue
		}
		machine.Name = ufsutil.RemovePrefix(machine.Name)
		devices = append(devices, ConstructInvV2Device(machine, lse))
	}
	return devices, failedDevices
}

// GetUFSDevicesByModels Gets MachineLSEs from UFS by Asset/Machine model.
func GetUFSDevicesByModels(ctx context.Context, ufsClient external.UFSClient, models []string) ([]*lab.ChromeOSDevice, []*api.DeviceOpResult) {
	var ids []string
	for _, model := range models {
		var pageToken string
		for {
			req := &ufsapi.ListMachinesRequest{
				PageSize:  1000,
				PageToken: pageToken,
				Filter:    fmt.Sprintf("model=%s", model),
			}
			res, err := ufsClient.ListMachines(ctx, req)
			if err != nil {
				logging.Errorf(ctx, "Failed to get MachineLSE for model %s", model)
				break
			}
			if len(res.GetMachines()) == 0 {
				logging.Errorf(ctx, "MachineLSE not found for model %s", model)
				break
			}
			for _, machine := range res.GetMachines() {
				ids = append(ids, ufsutil.RemovePrefix(machine.GetName()))
			}
			pageToken = res.GetNextPageToken()
			if pageToken == "" {
				break
			}
		}
	}
	return GetUFSDevicesByIds(ctx, ufsClient, ids)
}

// GetUFSDutStateForDevices Gets DutStates from UFS by Asset id/Machine id.
func GetUFSDutStateForDevices(ctx context.Context, ufsClient external.UFSClient, devices []*lab.ChromeOSDevice) ([]*api.ExtendedDeviceData, []*api.DeviceOpResult) {
	extendedData := make([]*api.ExtendedDeviceData, 0, len(devices))
	failedDevices := make([]*api.DeviceOpResult, 0, len(devices))
	for _, d := range devices {
		dutState, err := ufsClient.GetDutState(ctx, &ufsapi.GetDutStateRequest{
			ChromeosDeviceId: d.GetId().GetValue(),
		})
		if err != nil {
			failedDevices = append(failedDevices, &api.DeviceOpResult{
				Id:       d.GetId().GetValue(),
				ErrorMsg: err.Error(),
			})
			continue
		}
		data := &api.ExtendedDeviceData{
			LabConfig: d,
			DutState:  CopyUFSDutStateToInvV2DutState(dutState),
		}
		extendedData = append(extendedData, data)
	}
	return extendedData, failedDevices
}

// CopyUFSDutStateToInvV2DutState converts UFS DutState to InvV2 DutState proto format.
func CopyUFSDutStateToInvV2DutState(oldS *ufschromeoslab.DutState) *lab.DutState {
	if oldS == nil {
		return nil
	}
	s := proto.MarshalTextString(oldS)
	var newS lab.DutState
	proto.UnmarshalText(s, &newS)
	return &newS
}

// CopyUFSDutToInvV2Dut converts UFS DUT to InvV2 DUT proto format.
func CopyUFSDutToInvV2Dut(dut *ufschromeoslab.DeviceUnderTest) *lab.DeviceUnderTest {
	if dut == nil {
		return nil
	}
	s := proto.MarshalTextString(dut)
	var newDUT lab.DeviceUnderTest
	proto.UnmarshalText(s, &newDUT)
	return &newDUT
}

// CopyUFSLabstationToInvV2Labstation converts UFS Labstation to InvV2 Labstation proto format.
func CopyUFSLabstationToInvV2Labstation(labstation *ufschromeoslab.Labstation) *lab.Labstation {
	if labstation == nil {
		return nil
	}
	s := proto.MarshalTextString(labstation)
	var newL lab.Labstation
	proto.UnmarshalText(s, &newL)
	return &newL
}

func getDeviceConfigIDFromMachine(machine *ufspb.Machine) *device.ConfigId {
	buildTarget := strings.ToLower(machine.GetChromeosMachine().GetBuildTarget())
	model := strings.ToLower(machine.GetChromeosMachine().GetModel())
	devConfigID := &device.ConfigId{
		PlatformId: &device.PlatformId{
			Value: buildTarget,
		},
		ModelId: &device.ModelId{
			Value: model,
		},
	}
	sku := strings.ToLower(machine.GetChromeosMachine().GetSku())
	if sku != "" {
		devConfigID.VariantId = &device.VariantId{
			Value: sku,
		}
	}
	return devConfigID
}

// ConstructInvV2Device constructs a InvV2 Device from UFs MachineLSE and Machine.
func ConstructInvV2Device(machine *ufspb.Machine, lse *ufspb.MachineLSE) *lab.ChromeOSDevice {
	crosDevice := &lab.ChromeOSDevice{
		Id:              &lab.ChromeOSDeviceID{Value: machine.GetName()},
		SerialNumber:    machine.GetSerialNumber(),
		ManufacturingId: &manufacturing.ConfigID{Value: machine.GetChromeosMachine().GetHwid()},
		DeviceConfigId:  getDeviceConfigIDFromMachine(machine),
	}
	if lse.GetChromeosMachineLse().GetDeviceLse().GetDut() != nil {
		crosDevice.Device = &lab.ChromeOSDevice_Dut{
			Dut: CopyUFSDutToInvV2Dut(lse.GetChromeosMachineLse().GetDeviceLse().GetDut()),
		}
	} else {
		crosDevice.Device = &lab.ChromeOSDevice_Labstation{
			Labstation: CopyUFSLabstationToInvV2Labstation(lse.GetChromeosMachineLse().GetDeviceLse().GetLabstation()),
		}
	}
	return crosDevice
}

// GetUFSClient gets the UFS clien.
func GetUFSClient(ctx context.Context) (external.UFSClient, error) {
	es, err := external.GetServerInterface(ctx)
	if err != nil {
		return nil, err
	}
	return es.NewUFSInterfaceFactory(ctx, config.Get(ctx).GetUfsService())
}
