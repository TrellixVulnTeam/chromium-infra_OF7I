// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package labelupdater will handle the logic that update DUT's data
// back to UFS/inventory after task run.

package labelupdater

import (
	"context"
	"log"

	"github.com/golang/protobuf/proto"
	"go.chromium.org/chromiumos/infra/proto/go/lab"
	"go.chromium.org/luci/common/errors"

	"infra/cmd/skylab_swarming_worker/internal/swmbot"
	"infra/libs/skylab/inventory"
	ufspb "infra/unifiedfleet/api/v1/models"
	chromeosLab "infra/unifiedfleet/api/v1/models/chromeos/lab"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	ufsUtil "infra/unifiedfleet/app/util"
)

// labelUpdater implements an update method that is used as a dutinfo.UpdateFunc.
type LabelUpdater struct {
	BotInfo      *swmbot.Info
	TaskName     string
	UpdateLabels bool
}

// update is a dutinfo.UpdateFunc for updating DUT inventory labels.
// If adminServiceURL is empty, this method does nothing.
func (u LabelUpdater) Update(ctx context.Context, dutID string, old *inventory.DeviceUnderTest, new *inventory.DeviceUnderTest) error {
	// WARNING: This is an indirect check of if the job is a repair job.
	// By design, only repair job is allowed to update labels and has updateLabels set.
	// https://chromium.git.corp.google.com/infra/infra/+/7ae58795dd4badcfe9eadf4e109e27a498bed04c/go/src/infra/cmd/skylab_swarming_worker/main.go#207
	// And only repair job sets its local task account.
	// We cannot move this check later as swmbot.WithTaskAccount will fail for non-repair job.
	if u.BotInfo.AdminService == "" || !u.UpdateLabels {
		log.Printf("Skipping label update since no admin service was provided")
		return nil
	}

	ctx, err := swmbot.WithTaskAccount(ctx)
	if err != nil {
		return errors.Annotate(err, "update inventory labels").Err()
	}

	log.Printf("Calling UFS to update dutstate")
	if err := u.updateUFS(ctx, dutID, old, new); err != nil {
		return errors.Annotate(err, "fail to update to DutState in UFS").Err()
	}
	return nil
}

func (u LabelUpdater) updateUFS(ctx context.Context, dutID string, old, new *inventory.DeviceUnderTest) error {
	// Updating dutmeta, labmeta and dutstate to UFS
	ufsDutMeta := getUFSDutMetaFromSpecs(dutID, new.GetCommon())
	ufsLabMeta := getUFSLabMetaFromSpecs(dutID, new.GetCommon())
	ufsDutComponentState := getUFSDutComponentStateFromSpecs(dutID, new.GetCommon())
	ufsClient, err := swmbot.UFSClient(ctx, u.BotInfo)
	if err != nil {
		return errors.Annotate(err, "fail to create ufs client").Err()
	}
	osCtx := swmbot.SetupContext(ctx, ufsUtil.OSNamespace)
	ufsResp, err := ufsClient.UpdateDutState(osCtx, &ufsAPI.UpdateDutStateRequest{
		DutState: ufsDutComponentState,
		DutMeta:  ufsDutMeta,
		LabMeta:  ufsLabMeta,
	})
	log.Printf("resp for UFS update: %#v", ufsResp)
	if err != nil {
		return errors.Annotate(err, "fail to update UFS meta & component states").Err()
	}
	return nil
}

func getUFSDutMetaFromSpecs(dutID string, specs *inventory.CommonDeviceSpecs) *ufspb.DutMeta {
	attr := specs.GetAttributes()
	dutMeta := &ufspb.DutMeta{
		ChromeosDeviceId: dutID,
		Hostname:         specs.GetHostname(),
	}
	for _, kv := range attr {
		if kv.GetKey() == "serial_number" {
			dutMeta.SerialNumber = kv.GetValue()
		}
		if kv.GetKey() == "HWID" {
			dutMeta.HwID = kv.GetValue()
		}
	}
	dutMeta.DeviceSku = specs.GetLabels().GetSku()
	return dutMeta
}

func getUFSLabMetaFromSpecs(dutID string, specs *inventory.CommonDeviceSpecs) (labconfig *ufspb.LabMeta) {
	labMeta := &ufspb.LabMeta{
		ChromeosDeviceId: dutID,
		Hostname:         specs.GetHostname(),
	}
	p := specs.GetLabels().GetPeripherals()
	if p != nil {
		labMeta.ServoType = p.GetServoType()
		labMeta.SmartUsbhub = p.GetSmartUsbhub()
		labMeta.ServoTopology = copyServoTopology(convertServoTopology(p.GetServoTopology()))
	}

	return labMeta
}

func getUFSDutComponentStateFromSpecs(dutID string, specs *inventory.CommonDeviceSpecs) *chromeosLab.DutState {
	state := &chromeosLab.DutState{
		Id:       &chromeosLab.ChromeOSDeviceID{Value: dutID},
		Hostname: specs.GetHostname(),
	}
	l := specs.GetLabels()
	p := l.GetPeripherals()
	if p != nil {
		state.Servo = chromeosLab.PeripheralState(p.GetServoState())
		state.RpmState = chromeosLab.PeripheralState(p.GetRpmState())
		if p.GetChameleon() {
			state.Chameleon = chromeosLab.PeripheralState_WORKING
		}
		if p.GetAudioLoopbackDongle() {
			state.AudioLoopbackDongle = chromeosLab.PeripheralState_WORKING
		}
		state.WorkingBluetoothBtpeer = p.GetWorkingBluetoothBtpeer()
		switch l.GetCr50Phase() {
		case inventory.SchedulableLabels_CR50_PHASE_PVT:
			state.Cr50Phase = chromeosLab.DutState_CR50_PHASE_PVT
		case inventory.SchedulableLabels_CR50_PHASE_PREPVT:
			state.Cr50Phase = chromeosLab.DutState_CR50_PHASE_PREPVT
		}
		switch l.GetCr50RoKeyid() {
		case "prod":
			state.Cr50KeyEnv = chromeosLab.DutState_CR50_KEYENV_PROD
		case "dev":
			state.Cr50KeyEnv = chromeosLab.DutState_CR50_KEYENV_DEV
		}

		state.StorageState = chromeosLab.HardwareState(int32(p.GetStorageState()))
		state.ServoUsbState = chromeosLab.HardwareState(int32(p.GetServoUsbState()))
		state.BatteryState = chromeosLab.HardwareState(int32(p.GetBatteryState()))
		state.WifiState = chromeosLab.HardwareState(int32(p.GetWifiState()))
		state.BluetoothState = chromeosLab.HardwareState(int32(p.GetBluetoothState()))
	}
	return state
}

func copyServoTopology(topology *lab.ServoTopology) *chromeosLab.ServoTopology {
	if topology == nil {
		return nil
	}
	s := proto.MarshalTextString(topology)
	var newTopology chromeosLab.ServoTopology
	err := proto.UnmarshalText(s, &newTopology)
	if err != nil {
		log.Printf("cannot unmarshal servo topology: %s", err.Error())
		return nil
	}
	return &newTopology
}

func newServoTopologyItem(i *inventory.ServoTopologyItem) *lab.ServoTopologyItem {
	if i == nil {
		return nil
	}
	return &lab.ServoTopologyItem{
		Type:         i.GetType(),
		SysfsProduct: i.GetSysfsProduct(),
		Serial:       i.GetSerial(),
		UsbHubPort:   i.GetUsbHubPort(),
	}
}

func convertServoTopology(st *inventory.ServoTopology) *lab.ServoTopology {
	var t *lab.ServoTopology
	if st != nil {
		var children []*lab.ServoTopologyItem
		for _, child := range st.GetChildren() {
			children = append(children, newServoTopologyItem(child))
		}
		t = &lab.ServoTopology{
			Main:     newServoTopologyItem(st.Main),
			Children: children,
		}
	}
	return t
}
