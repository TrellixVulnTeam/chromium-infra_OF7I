// Copyright 2018 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package harness manages the setup and teardown of various Swarming
// bot resources for running lab tasks, like results directories and
// host info.
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

	"infra/cmd/skylab_swarming_worker/internal/swmbot/harness/botinfo"
	"infra/cmd/skylab_swarming_worker/internal/swmbot/harness/dutinfo"
	h_hostinfo "infra/cmd/skylab_swarming_worker/internal/swmbot/harness/hostinfo"
	"infra/cmd/skylab_swarming_worker/internal/swmbot/harness/resultsdir"
)

// closer interface to wrap Close method with providing context.
type closer interface {
	Close(ctx context.Context) error
}

// Info holds information about the Swarming harness.
type Info struct {
	*swmbot.Info

	ResultsDir string
	DUTName    string
	BotInfo    *swmbot.LocalState

	labelUpdater labelUpdater

	// err tracks errors during setup to simplify error handling
	// logic.
	err error

	closers []closer
}

// Close closes and flushes out the harness resources.  This is safe
// to call multiple times.
func (i *Info) Close(ctx context.Context) error {
	var errs []error
	for n := len(i.closers) - 1; n >= 0; n-- {
		if err := i.closers[n].Close(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errors.Annotate(errors.MultiError(errs), "close harness").Err()
	}
	return nil
}

// Open opens and sets up the bot and task harness needed for Autotest
// jobs.  An Info struct is returned with necessary fields, which must
// be closed.
func Open(ctx context.Context, b *swmbot.Info, o ...Option) (i *Info, err error) {
	i = &Info{
		Info: b,
		labelUpdater: labelUpdater{
			botInfo: b,
		},
	}
	defer func(i *Info) {
		if err != nil {
			_ = i.Close(ctx)
		}
	}(i)
	for _, o := range o {
		o(i)
	}
	d, sv := i.loadDUTInfo(ctx, b)
	i.DUTName = d.GetCommon().GetHostname()
	i.BotInfo = i.loadBotInfo(ctx, b)

	hi := i.makeHostInfo(d, sv)
	i.addBotInfoToHostInfo(hi, i.BotInfo)
	i.ResultsDir = i.makeResultsDir(b)
	i.exposeHostInfo(hi, i.ResultsDir, i.DUTName)
	if i.err != nil {
		return nil, errors.Annotate(i.err, "open harness").Err()
	}
	return i, nil
}

func (i *Info) loadBotInfo(ctx context.Context, b *swmbot.Info) *swmbot.LocalState {
	if i.err != nil {
		return nil
	}
	if i.DUTName == "" {
		i.err = fmt.Errorf("DUT Name cannot be blank")
	}
	bi, err := botinfo.Open(ctx, b, i.DUTName)
	if err != nil {
		i.err = err
		return nil
	}
	i.closers = append(i.closers, bi)
	return &bi.LocalState
}

func (i *Info) loadDUTInfo(ctx context.Context, b *swmbot.Info) (*inventory.DeviceUnderTest, map[string]string) {
	if i.err != nil {
		return nil, nil
	}
	var s *dutinfo.Store
	s, i.err = dutinfo.Load(ctx, b, i.labelUpdater.update)
	if i.err != nil {
		return nil, nil
	}
	i.closers = append(i.closers, s)
	return s.DUT, s.StableVersions
}

func (i *Info) makeHostInfo(d *inventory.DeviceUnderTest, stableVersion map[string]string) *hostinfo.HostInfo {
	if i.err != nil {
		return nil
	}
	hip := h_hostinfo.FromDUT(d, stableVersion)
	i.closers = append(i.closers, hip)
	return hip.HostInfo
}

func (i *Info) addBotInfoToHostInfo(hi *hostinfo.HostInfo, bi *swmbot.LocalState) {
	if i.err != nil {
		return
	}
	hib := h_hostinfo.BorrowBotInfo(hi, bi)
	i.closers = append(i.closers, hib)
}

func (i *Info) makeResultsDir(b *swmbot.Info) string {
	if i.err != nil {
		return ""
	}
	path := b.ResultsDir()
	rdc, err := resultsdir.Open(path)
	if err != nil {
		i.err = err
		return ""
	}
	log.Printf("Created results directory %s", path)
	i.closers = append(i.closers, rdc)
	return path
}

func (i *Info) exposeHostInfo(hi *hostinfo.HostInfo, resultsDir string, dutName string) {
	if i.err != nil {
		return
	}
	hif, err := h_hostinfo.Expose(hi, resultsDir, dutName)
	if err != nil {
		i.err = err
		return
	}
	i.closers = append(i.closers, hif)
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
