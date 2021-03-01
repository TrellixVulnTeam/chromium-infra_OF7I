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
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"

	api "infra/appengine/cros/lab_inventory/api/v1"
	"infra/appengine/cros/lab_inventory/app/config"
	"infra/appengine/cros/lab_inventory/app/external"
	shivasUtil "infra/cmd/shivas/utils"
	ufspb "infra/unifiedfleet/api/v1/models"
	ufschromeoslab "infra/unifiedfleet/api/v1/models/chromeos/lab"
	ufsapi "infra/unifiedfleet/api/v1/rpc"
	ufsutil "infra/unifiedfleet/app/util"
)

// DeviceData holds the invV2 Device and updatetime(of MachineLSE)
type DeviceData struct {
	Device     *lab.ChromeOSDevice
	UpdateTime *timestamppb.Timestamp
}

// DutStateData holds the invV2 DutState and updatetime
type DutStateData struct {
	DutState   *lab.DutState
	UpdateTime *timestamppb.Timestamp
}

// UpdateUFSDutState updates dutmeta, labmeta and dutstate in UFS
func UpdateUFSDutState(ctx context.Context, req *api.UpdateDutsStatusRequest) ([]*api.DeviceOpResult, []*api.DeviceOpResult, error) {
	ufsClient, err := GetUFSClient(ctx)
	if err != nil {
		return nil, nil, err
	}
	ctx = SetupOSNameSpaceContext(ctx)
	dutMetas := req.GetDutMetas()
	labMetas := req.GetLabMetas()
	dutStates := req.GetStates()
	failed := make([]*api.DeviceOpResult, 0, len(dutMetas))
	passed := make([]*api.DeviceOpResult, 0, len(dutMetas))
	for i := range dutMetas {
		req := &ufsapi.ListMachineLSEsRequest{
			PageSize: 1,
			Filter:   fmt.Sprintf("machine=%s", dutMetas[i].GetChromeosDeviceId()),
		}
		res, err := ufsClient.ListMachineLSEs(ctx, req)
		if err != nil {
			logging.Errorf(ctx, "ListMachineLSEs failed for %s", dutMetas[i].GetChromeosDeviceId())
			return nil, nil, err
		}
		if len(res.GetMachineLSEs()) == 0 {
			logging.Errorf(ctx, "No MachineLSE found for %s", dutMetas[i].GetChromeosDeviceId())
			failed = append(failed, &api.DeviceOpResult{
				Id:       dutMetas[i].GetChromeosDeviceId(),
				ErrorMsg: fmt.Sprintf("No MachineLSE found for for %s", dutMetas[i].GetChromeosDeviceId()),
			})
			return nil, failed, nil
		}
		lse := res.GetMachineLSEs()[0]
		lse.Name = ufsutil.RemovePrefix(lse.Name)

		_, err = ufsClient.UpdateDutState(ctx, &ufsapi.UpdateDutStateRequest{
			DutState: CopyInvV2DutStateToUFSDutState(dutStates[i], lse.Name),
			DutMeta:  CopyInvV2DutMetaToUFSDutMeta(dutMetas[i], lse.Name),
			LabMeta:  CopyInvV2LabMetaToUFSLabMeta(labMetas[i], lse.Name),
		})
		if err != nil {
			failed = append(failed, &api.DeviceOpResult{
				Id:       dutMetas[i].GetChromeosDeviceId(),
				Hostname: lse.Name,
				ErrorMsg: err.Error(),
			})
			continue
		}
		passed = append(passed, &api.DeviceOpResult{
			Id:       dutMetas[i].GetChromeosDeviceId(),
			Hostname: lse.Name,
		})
	}
	return passed, failed, nil
}

// GetUFSDevicesByIds Gets MachineLSEs from UFS by Asset id/Machine id.
func GetUFSDevicesByIds(ctx context.Context, ufsClient external.UFSClient, ids []string) ([]*lab.ChromeOSDevice, []*api.DeviceOpResult) {
	ctx = SetupOSNameSpaceContext(ctx)
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
		if err != nil {
			logging.Errorf(ctx, "ListMachineLSEs failed for %s", id)
			failedDevices = append(failedDevices, &api.DeviceOpResult{
				Id:       id,
				ErrorMsg: err.Error(),
			})
			continue
		}
		if len(res.GetMachineLSEs()) == 0 {
			logging.Errorf(ctx, "MachineLSE not found for machine ID %s", id)
			failedDevices = append(failedDevices, &api.DeviceOpResult{
				Id:       id,
				ErrorMsg: fmt.Sprintf("No MachineLSE found for for %s", id),
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
	ctx = SetupOSNameSpaceContext(ctx)
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
				ErrorMsg: fmt.Sprintf("Machine not found for hostname %s", name),
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
	ctx = SetupOSNameSpaceContext(ctx)
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
	ctx = SetupOSNameSpaceContext(ctx)
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

// GetAllUFSDevicesData Gets all the MachineLSEs and Machines from UFS and returns invV2 Devices and updatedtime.
func GetAllUFSDevicesData(ctx context.Context, ufsClient external.UFSClient) ([]*DeviceData, error) {
	ctx = SetupOSNameSpaceContext(ctx)
	var devicesData []*DeviceData
	idToMachine := make(map[string]*ufspb.Machine, 0)
	for curPageToken := ""; ; {
		req := &ufsapi.ListMachinesRequest{
			PageSize:  1000,
			PageToken: curPageToken,
		}
		res, err := ufsClient.ListMachines(ctx, req)
		if err != nil {
			return nil, errors.Annotate(err, "Failed to get Machines from UFS").Err()
		}
		for _, machine := range res.GetMachines() {
			machine.Name = ufsutil.RemovePrefix(machine.Name)
			idToMachine[machine.GetName()] = machine
		}
		if res.GetNextPageToken() == "" {
			break
		}
		curPageToken = res.GetNextPageToken()
	}
	for curPageToken := ""; ; {
		req := &ufsapi.ListMachineLSEsRequest{
			PageSize:  1000,
			PageToken: curPageToken,
		}
		res, err := ufsClient.ListMachineLSEs(ctx, req)
		if err != nil {
			return nil, errors.Annotate(err, "Failed to get MachineLSEs from UFS").Err()
		}
		for _, lse := range res.GetMachineLSEs() {
			lse.Name = ufsutil.RemovePrefix(lse.Name)
			if len(lse.GetMachines()) == 0 {
				logging.Errorf(ctx, "no Machine in LSE %s", lse.GetName())
				continue
			}
			if machine, found := idToMachine[lse.GetMachines()[0]]; found {
				deviceData := &DeviceData{
					Device:     ConstructInvV2Device(machine, lse),
					UpdateTime: lse.GetUpdateTime(),
				}
				devicesData = append(devicesData, deviceData)
				continue
			}
			logging.Errorf(ctx, "no Machine found %s", lse.GetMachines()[0])
		}
		if res.GetNextPageToken() == "" {
			break
		}
		curPageToken = res.GetNextPageToken()
	}
	return devicesData, nil
}

// GetAllUFSDutStatesData Gets all the DutStateLSEs and DutStates from UFS and returns invV2 DutStates and updatedtime.
func GetAllUFSDutStatesData(ctx context.Context, ufsClient external.UFSClient) ([]*DutStateData, error) {
	ctx = SetupOSNameSpaceContext(ctx)
	var dutStatesData []*DutStateData
	for curPageToken := ""; ; {
		req := &ufsapi.ListDutStatesRequest{
			PageSize:  1000,
			PageToken: curPageToken,
		}
		res, err := ufsClient.ListDutStates(ctx, req)
		if err != nil {
			return nil, errors.Annotate(err, "Failed to get DutStates from UFS").Err()
		}
		for _, dutState := range res.GetDutStates() {
			dutStateData := &DutStateData{
				DutState:   CopyUFSDutStateToInvV2DutState(dutState),
				UpdateTime: dutState.GetUpdateTime(),
			}
			dutStatesData = append(dutStatesData, dutStateData)
		}
		if res.GetNextPageToken() == "" {
			break
		}
		curPageToken = res.GetNextPageToken()
	}
	return dutStatesData, nil
}

// CreateMachineLSEs creates machine LSEs in UFS from given cros devices and returns AddCrosDevicesResponse.
// Intended to be used in piping IV2 API to UFS.
func CreateMachineLSEs(iv2ctx context.Context, devices []*lab.ChromeOSDevice, pickServoPort bool) *api.AddCrosDevicesResponse {
	// Response to be returned for AddCrosDevices API
	resp := &api.AddCrosDevicesResponse{
		PassedDevices: []*api.DeviceOpResult{},
		FailedDevices: []*api.DeviceOpResult{},
	}
	// Helper function to record errors on failure to AddCrosDevicesResponse
	recErr := func(id, hostname string, err error) {
		resp.FailedDevices = append(resp.FailedDevices, &api.DeviceOpResult{
			Id:       id,
			Hostname: hostname,
			ErrorMsg: err.Error(),
		})
	}
	ctx := SetupOSNameSpaceContext(iv2ctx)
	// Create a UFS client
	ufsClient, err := GetUFSClient(ctx)
	if err != nil {
		for _, device := range devices {
			recErr(device.GetId().GetValue(), "", errors.Annotate(err, "CreateMachineLSEs - [UFS] Failed to create machine lse.").Err())
		}
		return resp
	}
	// Iterate through all the devices and update UFS with the required data
	for _, device := range devices {
		// Determine the hostname for the LSE.
		var hostname string
		if device.GetDut() != nil {
			hostname = device.GetDut().GetHostname()
		} else {
			hostname = device.GetLabstation().GetHostname()
		}
		// Check if the machine exists
		_, err := ufsClient.GetMachine(ctx, &ufsapi.GetMachineRequest{
			Name: ufsutil.AddPrefix(ufsutil.MachineCollection, device.GetId().GetValue()),
		})
		if err != nil {
			// Check if it was a not found error and create an asset &| rack.
			if ufsutil.IsNotFoundError(err) {
				// Create an asset after verifying that the rack exists.
				var loc *ufspb.Location
				if shivasUtil.IsLocation(hostname) {
					loc, err = shivasUtil.GetLocation(hostname)
					if err != nil {
						recErr(device.GetId().GetValue(), hostname, errors.Annotate(err, "CreateMachineLSEs - Failed to determine location").Err())
						continue
					}
				} else {
					recErr(device.GetId().GetValue(), hostname, errors.Reason("CreateMachineLSEs - Cannot determine location for %s", hostname).Err())
					continue
				}
				// Check if rack exists
				_, err = ufsClient.GetRack(ctx, &ufsapi.GetRackRequest{
					Name: ufsutil.AddPrefix(ufsutil.RackCollection, loc.GetRack()),
				})
				if err != nil && ufsutil.IsNotFoundError(err) {
					_, err = ufsClient.RackRegistration(ctx, &ufsapi.RackRegistrationRequest{
						Rack: &ufspb.Rack{
							Name:        loc.GetRack(),
							Location:    loc,
							Description: "Added from IV2 as a part of UFS migration",
							Tags:        []string{"UFS-migration", "source=IV2"},
						},
					})
					if err != nil {
						recErr(device.GetId().GetValue(), hostname, errors.Annotate(err, "CreateMachineLSEs - Unable to create rack in UFS").Err())
						continue
					}
				} else if err != nil {
					recErr(device.GetId().GetValue(), hostname, errors.Annotate(err, "CreateMachineLSEs - Unable to check if the rack exists").Err())
					continue
				}
				// Construct and add an asset based on the cros device.
				var model, board, variant string
				if dconfigID := device.GetDeviceConfigId(); dconfigID != nil {
					if dconfigID.GetPlatformId() != nil {
						board = dconfigID.GetPlatformId().GetValue()
					} else {
						recErr(device.GetId().GetValue(), hostname, errors.Reason("CreateMachineLSEs - Cannot create host without board").Err())
						continue
					}
					if dconfigID.GetModelId() != nil {
						model = dconfigID.GetModelId().GetValue()
					} else {
						recErr(device.GetId().GetValue(), hostname, errors.Reason("CreateMachineLSEs - Cannot create host without model").Err())
						continue
					}
					if dconfigID.GetVariantId() != nil {
						variant = dconfigID.GetVariantId().GetValue()
					}
				}
				asset := &ufspb.Asset{
					Name:     ufsutil.AddPrefix(ufsutil.AssetCollection, device.GetId().GetValue()),
					Location: loc,
					Info: &ufspb.AssetInfo{
						AssetTag:    device.GetId().GetValue(),
						Model:       model,
						BuildTarget: board,
						Sku:         variant,
					},
					Model: model,
				}
				if device.GetDut() != nil {
					asset.Type = ufspb.AssetType_DUT
				} else {
					asset.Type = ufspb.AssetType_LABSTATION
				}
				// Create the asset and update to UFS
				_, err = ufsClient.CreateAsset(ctx, &ufsapi.CreateAssetRequest{
					Asset: asset,
				})
				if err != nil {
					recErr(device.GetId().GetValue(), hostname, errors.Annotate(err, "CreateMachineLSEs - Failed to create asset in UFS").Err())
					continue
				}
			} else {
				// Error was due to some other issue. Log and continue
				recErr(device.GetId().GetValue(), hostname, errors.Annotate(err, "CreateMachineLSEs - Failed to determine if the asset exists").Err())
				continue
			}
		}
		// Create and update machine lse to UFS.
		var mlse *ufspb.MachineLSE
		if device.GetDut() != nil {
			// Copy the dut to UFS dut.
			mlse = ufsutil.DUTToLSE(device.GetDut(), device.GetId().GetValue(), nil)
			if pickServoPort {
				// UFS assigns a servo port if its set to 0
				mlse.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetServo().ServoPort = int32(0)
			}
		} else {
			mlse = ufsutil.LabstationToLSE(device.GetLabstation(), device.GetId().GetValue(), nil)
		}
		_, err = ufsClient.CreateMachineLSE(ctx, &ufsapi.CreateMachineLSERequest{
			MachineLSE:   mlse,
			MachineLSEId: mlse.GetName(),
		})
		if err != nil {
			recErr(device.GetId().GetValue(), hostname, errors.Annotate(err, "CreateMachineLSEs - Unable to create host %s", hostname).Err())
			continue
		}
		resp.PassedDevices = append(resp.PassedDevices, &api.DeviceOpResult{
			Id:       device.GetId().GetValue(),
			Hostname: hostname,
		})
	}
	return resp
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

// CopyInvV2DutStateToUFSDutState converts InvV2 DutState to UFS DutState
func CopyInvV2DutStateToUFSDutState(oldS *lab.DutState, hostname string) *ufschromeoslab.DutState {
	if oldS == nil {
		return nil
	}
	s := proto.MarshalTextString(oldS)
	var newS ufschromeoslab.DutState
	proto.UnmarshalText(s, &newS)
	newS.Hostname = hostname
	return &newS
}

// CopyInvV2DutMetaToUFSDutMeta converts InvV2 DutMeta to UFS DutMeta
func CopyInvV2DutMetaToUFSDutMeta(oldDm *api.DutMeta, hostname string) *ufspb.DutMeta {
	if oldDm == nil {
		return nil
	}
	s := proto.MarshalTextString(oldDm)
	var newDm ufspb.DutMeta
	proto.UnmarshalText(s, &newDm)
	newDm.Hostname = hostname
	return &newDm
}

// CopyInvV2LabMetaToUFSLabMeta converts InvV2 LabMeta to UFS LabMeta
func CopyInvV2LabMetaToUFSLabMeta(oldLm *api.LabMeta, hostname string) *ufspb.LabMeta {
	if oldLm == nil {
		return nil
	}
	s := proto.MarshalTextString(oldLm)
	var newLm ufspb.LabMeta
	proto.UnmarshalText(s, &newLm)
	newLm.Hostname = hostname
	return &newLm
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

// SetupOSNameSpaceContext sets up context with namespace
func SetupOSNameSpaceContext(ctx context.Context) context.Context {
	md := metadata.Pairs(ufsutil.Namespace, ufsutil.OSNamespace)
	return metadata.NewOutgoingContext(ctx, md)
}
