// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package harness

import (
	"context"
	"fmt"
	"log"

	"github.com/golang/protobuf/proto"
	"go.chromium.org/chromiumos/infra/proto/go/lab"
	"go.chromium.org/luci/common/errors"

	invV2 "infra/appengine/cros/lab_inventory/api/v1"
	"infra/libs/skylab/inventory"
	ufspb "infra/unifiedfleet/api/v1/models"
	chromeosLab "infra/unifiedfleet/api/v1/models/chromeos/lab"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	ufsUtil "infra/unifiedfleet/app/util"

	"infra/libs/skylab/autotest/hostinfo"

	"infra/cmd/skylab_swarming_worker/internal/swmbot"
	h_hostinfo "infra/cmd/skylab_swarming_worker/internal/swmbot/harness/hostinfo"
	"infra/cmd/skylab_swarming_worker/internal/swmbot/harness/localdutinfo"
	"infra/cmd/skylab_swarming_worker/internal/swmbot/harness/resultsdir"
	"infra/cmd/skylab_swarming_worker/internal/swmbot/harness/ufsdutinfo"
)

// DUTHarness holds information about a DUT's harness
type DUTHarness struct {
	BotInfo      *swmbot.Info
	DUTID        string
	DUTHostname  string
	ResultsDir   string
	LocalState   *swmbot.LocalDUTState
	labelUpdater labelUpdater
	// err tracks errors during setup to simplify error handling logic.
	err     error
	closers []closer
}

// Close closes and flushes out the harness resources.  This is safe
// to call multiple times.
func (dh *DUTHarness) Close(ctx context.Context) error {
	log.Printf("Wrapping up harness for %s", dh.DUTHostname)
	var errs []error
	for n := len(dh.closers) - 1; n >= 0; n-- {
		if err := dh.closers[n].Close(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errors.Annotate(errors.MultiError(errs), "close harness").Err()
	}
	return nil
}

func makeDUTHarness(b *swmbot.Info) *DUTHarness {
	return &DUTHarness{
		BotInfo: b,
		labelUpdater: labelUpdater{
			botInfo: b,
		},
	}
}

func (dh *DUTHarness) loadLocalDUTInfo(ctx context.Context) {
	if dh.err != nil {
		return
	}
	if dh.DUTHostname == "" {
		dh.err = fmt.Errorf("DUTHostname cannot be blank")
		return
	}
	ldi, err := localdutinfo.Open(ctx, dh.BotInfo, dh.DUTHostname, dh.DUTID)
	if err != nil {
		dh.err = err
		return
	}
	dh.closers = append(dh.closers, ldi)
	dh.LocalState = &ldi.LocalDUTState
}

func (dh *DUTHarness) loadUFSDUTInfo(ctx context.Context) (*inventory.DeviceUnderTest, map[string]string) {
	if dh.err != nil {
		return nil, nil
	}
	var s *ufsdutinfo.Store
	if dh.DUTID != "" {
		s, dh.err = ufsdutinfo.LoadByID(ctx, dh.BotInfo, dh.DUTID, dh.labelUpdater.update)
	} else if dh.DUTHostname != "" {
		s, dh.err = ufsdutinfo.LoadByHostname(ctx, dh.BotInfo, dh.DUTHostname, dh.labelUpdater.update)
	} else {
		dh.err = fmt.Errorf("Both DUTID and DUTHostname field is empty.")
	}
	if dh.err != nil {
		return nil, nil
	}
	// We overwrite both DUTHostname and DUTID based on UFS data because in
	// single DUT tasks we don't have DUTHostname when we start, and in the
	// scheduling_unit (multi-DUT) tasks we don't have DUTID when we start.
	dh.DUTHostname = s.DUT.GetCommon().GetHostname()
	dh.DUTID = s.DUT.GetCommon().GetId()
	dh.closers = append(dh.closers, s)
	return s.DUT, s.StableVersions
}

func (dh *DUTHarness) makeHostInfo(d *inventory.DeviceUnderTest, stableVersion map[string]string) *hostinfo.HostInfo {
	if dh.err != nil {
		return nil
	}
	hip := h_hostinfo.FromDUT(d, stableVersion)
	dh.closers = append(dh.closers, hip)
	return hip.HostInfo
}

func (dh *DUTHarness) addLocalStateToHostInfo(hi *hostinfo.HostInfo) {
	if dh.err != nil {
		return
	}
	hib := h_hostinfo.BorrowLocalDUTState(hi, dh.LocalState)
	dh.closers = append(dh.closers, hib)
}

func (dh *DUTHarness) makeDUTResultsDir(d *resultsdir.Dir) {
	if dh.err != nil {
		return
	}
	path, err := d.OpenSubDir(dh.DUTHostname)
	if err != nil {
		dh.err = err
		return
	}
	log.Printf("Created DUT level results sub-dir %s", path)
	dh.ResultsDir = path
}

func (dh *DUTHarness) exposeHostInfo(hi *hostinfo.HostInfo) {
	if dh.err != nil {
		return
	}
	hif, err := h_hostinfo.Expose(hi, dh.ResultsDir, dh.DUTHostname)
	if err != nil {
		dh.err = err
		return
	}
	dh.closers = append(dh.closers, hif)
}

// labelUpdater implements an update method that is used as a dutinfo.UpdateFunc.
type labelUpdater struct {
	botInfo      *swmbot.Info
	taskName     string
	updateLabels bool
}

// update is a dutinfo.UpdateFunc for updating DUT inventory labels.
// If adminServiceURL is empty, this method does nothing.
func (u labelUpdater) update(ctx context.Context, dutID string, old *inventory.DeviceUnderTest, new *inventory.DeviceUnderTest) error {
	// WARNING: This is an indirect check of if the job is a repair job.
	// By design, only repair job is allowed to update labels and has updateLabels set.
	// https://chromium.git.corp.google.com/infra/infra/+/7ae58795dd4badcfe9eadf4e109e27a498bed04c/go/src/infra/cmd/skylab_swarming_worker/main.go#207
	// And only repair job sets its local task account.
	// We cannot move this check later as swmbot.WithTaskAccount will fail for non-repair job.
	if u.botInfo.AdminService == "" || !u.updateLabels {
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

// TODO(xixuan): move it to lib.
func getStatesFromLabel(dutID string, l *inventory.SchedulableLabels) *lab.DutState {
	state := lab.DutState{
		Id: &lab.ChromeOSDeviceID{Value: dutID},
	}
	p := l.GetPeripherals()
	if p != nil {
		state.Servo = lab.PeripheralState(p.GetServoState())
		state.RpmState = lab.PeripheralState(p.GetRpmState())
		if p.GetChameleon() {
			state.Chameleon = lab.PeripheralState_WORKING
		}
		if p.GetAudioLoopbackDongle() {
			state.AudioLoopbackDongle = lab.PeripheralState_WORKING
		}
		state.WorkingBluetoothBtpeer = p.GetWorkingBluetoothBtpeer()
		switch l.GetCr50Phase() {
		case inventory.SchedulableLabels_CR50_PHASE_PVT:
			state.Cr50Phase = lab.DutState_CR50_PHASE_PVT
		case inventory.SchedulableLabels_CR50_PHASE_PREPVT:
			state.Cr50Phase = lab.DutState_CR50_PHASE_PREPVT
		}
		switch l.GetCr50RoKeyid() {
		case "prod":
			state.Cr50KeyEnv = lab.DutState_CR50_KEYENV_PROD
		case "dev":
			state.Cr50KeyEnv = lab.DutState_CR50_KEYENV_DEV
		}

		state.StorageState = lab.HardwareState(int32(p.GetStorageState()))
		state.ServoUsbState = lab.HardwareState(int32(p.GetServoUsbState()))
		state.BatteryState = lab.HardwareState(int32(p.GetBatteryState()))
	}
	return &state
}

func getMetaFromSpecs(dutID string, specs *inventory.CommonDeviceSpecs) *invV2.DutMeta {
	attr := specs.GetAttributes()
	dutMeta := invV2.DutMeta{
		ChromeosDeviceId: dutID,
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
	return &dutMeta
}

func getLabMetaFromLabel(dutID string, l *inventory.SchedulableLabels) (labconfig *invV2.LabMeta) {
	labMeta := invV2.LabMeta{
		ChromeosDeviceId: dutID,
	}
	p := l.GetPeripherals()
	if p != nil {
		labMeta.ServoType = p.GetServoType()
		labMeta.SmartUsbhub = p.GetSmartUsbhub()
		labMeta.ServoTopology = convertServoTopology(p.GetServoTopology())
	}

	return &labMeta
}

// TODO (xixuan): will remove the above duplicated functions when UFS feature for OS lab is launched.
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

func (u labelUpdater) updateUFS(ctx context.Context, dutID string, old, new *inventory.DeviceUnderTest) error {
	// Updating dutmeta, labmeta and dutstate to UFS
	ufsDutMeta := getUFSDutMetaFromSpecs(dutID, new.GetCommon())
	ufsLabMeta := getUFSLabMetaFromSpecs(dutID, new.GetCommon())
	ufsDutComponentState := getUFSDutComponentStateFromSpecs(dutID, new.GetCommon())
	ufsClient, err := swmbot.UFSClient(ctx, u.botInfo)
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
