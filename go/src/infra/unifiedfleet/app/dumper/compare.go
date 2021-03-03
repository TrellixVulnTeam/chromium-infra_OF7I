// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dumper

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/google/go-cmp/cmp"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/grpc/prpc"
	crimson "go.chromium.org/luci/machine-db/api/crimson/v1"
	"go.chromium.org/luci/server/auth"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/testing/protocmp"

	invV2Api "infra/appengine/cros/lab_inventory/api/v1"
	ufspb "infra/unifiedfleet/api/v1/models"
	chromeosLab "infra/unifiedfleet/api/v1/models/chromeos/lab"
	"infra/unifiedfleet/app/config"
	"infra/unifiedfleet/app/model/configuration"
	fleetds "infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/model/inventory"
	"infra/unifiedfleet/app/model/registration"
	"infra/unifiedfleet/app/model/state"
	"infra/unifiedfleet/app/util"
)

func compareInventoryV2(ctx context.Context, crosInventoryHost string) error {
	writer, err := getCloudStorageWriter(ctx, fmt.Sprintf("inv2_ufs_diff/%s.log", getCurTimeFilename()))
	if err != nil {
		return err
	}
	defer func() {
		if writer != nil {
			if err := writer.Close(); err != nil {
				logging.Warningf(ctx, "failed to close cloud storage writer: %s", err)
			}
		}
	}()

	t, err := auth.GetRPCTransport(ctx, auth.AsSelf)
	if err != nil {
		return err
	}
	inv2Client := invV2Api.NewInventoryPRPCClient(&prpc.Client{
		C:    &http.Client{Transport: t},
		Host: crosInventoryHost,
	})
	resp, err := inv2Client.ListCrosDevicesLabConfig(ctx, &invV2Api.ListCrosDevicesLabConfigRequest{})
	if err != nil {
		return err
	}
	logging.Infof(ctx, "length of configs from Inv2: %d", len(resp.GetLabConfigs()))

	// The ctx here is in OS namespace
	hostRes, err := inventory.GetAllMachineLSEs(ctx)
	if hostRes == nil {
		return errors.New("OS host entity corrupted")
	}
	dutStateRes, err := state.GetAllDutStates(ctx)
	if dutStateRes == nil {
		return errors.New("OS DUT state entity corrupted")
	}
	if err := compareDuts(ctx, writer, resp.GetLabConfigs(), hostRes, dutStateRes); err != nil {
		return err
	}
	return nil
}

func compareDuts(ctx context.Context, writer *storage.Writer, oldHosts []*invV2Api.ListCrosDevicesLabConfigResponse_LabConfig, newHosts *fleetds.OpResults, newDutStates *fleetds.OpResults) error {
	onlyShowNum := 10
	logs := []string{fmt.Sprintf("\n\n######## OS DUTs diff (Only show the first %d) ############", onlyShowNum)}
	inv2LSEs, _ := util.ToOSMachineLSEs(oldHosts)
	inv2States := util.ToOSDutStates(oldHosts)
	inv2Hosts := make(map[string]*ufspb.MachineLSE)
	inv2DutStates := make(map[string]*chromeosLab.DutState)

	for _, lse := range inv2LSEs {
		inv2Hosts[lse.GetName()] = lse
	}
	for _, s := range inv2States {
		inv2DutStates[s.GetId().GetValue()] = s
	}

	ufsHosts := make(map[string]*ufspb.MachineLSE)
	ufsDutStates := make(map[string]*chromeosLab.DutState)
	for _, r := range newHosts.Passed() {
		m := r.Data.(*ufspb.MachineLSE)
		if m.GetChromeBrowserMachineLse() != nil {
			logs = append(logs, fmt.Sprintf("\nWrong browser host in OS namespace: %s", m.GetName()))
		}
		ufsHosts[m.GetName()] = m
	}
	for _, r := range newDutStates.Passed() {
		m := r.Data.(*chromeosLab.DutState)
		ufsDutStates[m.GetId().GetValue()] = m
	}

	logs = append(logs, "Resources in inventory V2 but not in UFS:")
	var diffs []string
	for k, v := range inv2Hosts {
		if v2, ok := ufsHosts[k]; !ok {
			logs = append(logs, v.GetName())
		} else if v != v2 {
			if len(diffs) < onlyShowNum {
				diffs = append(diffs, logOSLseDiff(v, v2))
			}
		}
	}
	logs = append(logs, "Resources in UFS but not in inventory V2:")
	for k, v := range ufsHosts {
		if _, ok := inv2Hosts[k]; !ok {
			logs = append(logs, v.GetName())
		}
	}
	logs = append(logs, "Resources in both UFS and inventory V2 but has difference (UFS ++):")
	logs = append(logs, diffs...)

	logs = append(logs, "DutStates in inventory V2 but not in UFS:")
	var stateDiffs []string
	for k, v := range inv2DutStates {
		if v2, ok := ufsDutStates[k]; !ok {
			logs = append(logs, v.GetHostname())
		} else if v != v2 {
			if len(stateDiffs) < onlyShowNum {
				stateDiffs = append(stateDiffs, logOSDutStateDiff(v, v2))
			}
		}
	}
	logs = append(logs, "DutStates in UFS but not in inventory V2:")
	for k, v := range ufsHosts {
		if _, ok := inv2Hosts[k]; !ok {
			logs = append(logs, v.GetName())
		}
	}
	logs = append(logs, "DutStates in both UFS and inventory V2 but has difference (UFS ++):")
	logs = append(logs, stateDiffs...)
	if _, err := fmt.Fprintf(writer, strings.Join(logs, "\n")); err != nil {
		return err
	}
	return nil
}

// Compare does protobuf comparison between both MachineLSE
func logOSLseDiff(inv2Lse, ufsLSE *ufspb.MachineLSE) string {
	// Ignoring fields not required for comparison
	opts1 := protocmp.IgnoreFields(inv2Lse, protoreflect.Name("update_time"))
	// See: https://developers.google.com/protocol-buffers/docs/reference/go/faq#deepequal
	opts2 := protocmp.Transform()
	return cmp.Diff(inv2Lse, ufsLSE, opts1, opts2)
}

//
func logOSDutStateDiff(inv2DS, ufsDS *chromeosLab.DutState) string {
	// Ignoring fields not required for comparison
	opts1 := protocmp.IgnoreFields(inv2DS, protoreflect.Name("update_time"))
	// See: https://developers.google.com/protocol-buffers/docs/reference/go/faq#deepequal
	opts2 := protocmp.Transform()
	return cmp.Diff(inv2DS, ufsDS, opts1, opts2)
}

func compareCrimson(ctx context.Context, machineDBHost string) error {
	filename := fmt.Sprintf("crimson_ufs_diff.%s.log", time.Now().UTC().Format("2006-01-02T03:04:05"))
	writer, err := getCloudStorageWriter(ctx, filename)
	if err != nil {
		return err
	}
	defer func() {
		if writer != nil {
			if err := writer.Close(); err != nil {
				logging.Warningf(ctx, "failed to close cloud storage writer: %s", err)
			}
		}
	}()

	t, err := auth.GetRPCTransport(ctx, auth.AsSelf)
	if err != nil {
		return err
	}
	crimsonClient := crimson.NewCrimsonPRPCClient(&prpc.Client{
		C:    &http.Client{Transport: t},
		Host: machineDBHost,
	})
	stateRes, err := state.GetAllStates(ctx)
	if err != nil {
		return err
	}
	if stateRes == nil {
		return errors.New("state entity corrupted")
	}
	stateMap := make(map[string]ufspb.State)
	for _, sr := range stateRes.Passed() {
		s := sr.Data.(*ufspb.StateRecord)
		stateMap[s.GetResourceName()] = s.GetState()
	}

	dhcpRes, err := configuration.GetAllDHCPs(ctx)
	if err != nil {
		return err
	}
	if dhcpRes == nil {
		return errors.New("dhcp entity corrupted")
	}
	dhcpMap := make(map[string]*ufspb.DHCPConfig)
	dhcpHostMap := make(map[string]*ufspb.DHCPConfig)
	for _, dhcp := range dhcpRes.Passed() {
		d := dhcp.Data.(*ufspb.DHCPConfig)
		dhcpMap[d.GetMacAddress()] = d
		dhcpHostMap[d.GetHostname()] = d
	}

	rackRes, err := registration.GetAllRacks(ctx)
	if err != nil {
		return err
	}
	if rackRes == nil {
		return errors.New("rack entity corrupted")
	}

	vlanResp, err := crimsonClient.ListVLANs(ctx, &crimson.ListVLANsRequest{})
	if err != nil {
		return err
	}
	vlanRes, err := configuration.GetAllVlans(ctx)
	if vlanRes == nil {
		return errors.New("vlan entity corrupted")
	}
	if err := compareVlans(ctx, writer, vlanResp.Vlans, vlanRes, stateMap); err != nil {
		return err
	}

	rackResp, err := crimsonClient.ListRacks(ctx, &crimson.ListRacksRequest{})
	if err != nil {
		return err
	}

	if err := compareRacks(ctx, writer, rackResp.Racks, rackRes, stateMap); err != nil {
		return err
	}

	machineResp, err := crimsonClient.ListMachines(ctx, &crimson.ListMachinesRequest{})
	if err != nil {
		return err
	}
	machineRes, err := registration.GetAllMachines(ctx)
	if err != nil {
		return err
	}
	if machineRes == nil {
		return errors.New("machine entity corrupted")
	}
	if err := compareMachines(ctx, writer, machineResp.Machines, machineRes, stateMap); err != nil {
		return err
	}

	kvmResp, err := crimsonClient.ListKVMs(ctx, &crimson.ListKVMsRequest{})
	if err != nil {
		return err
	}
	kvmRes, err := registration.GetAllKVMs(ctx)
	if err := compareKVMs(ctx, writer, kvmResp.Kvms, kvmRes, stateMap); err != nil {
		return err
	}
	if kvmRes == nil {
		return errors.New("kvm entity corrupted")
	}

	switchResp, err := crimsonClient.ListSwitches(ctx, &crimson.ListSwitchesRequest{})
	if err != nil {
		return err
	}
	switchRes, err := registration.GetAllSwitches(ctx)
	if err := compareSwitches(ctx, writer, switchResp.Switches, switchRes, stateMap); err != nil {
		return err
	}
	if switchRes == nil {
		return errors.New("switch entity corrupted")
	}

	nicResp, err := crimsonClient.ListNICs(ctx, &crimson.ListNICsRequest{})
	if err != nil {
		return err
	}
	nicRes, err := registration.GetAllNics(ctx)
	if err := compareNics(ctx, writer, nicResp.Nics, nicRes); err != nil {
		return err
	}
	if nicRes == nil {
		return errors.New("nic entity corrupted")
	}

	dracResp, err := crimsonClient.ListDRACs(ctx, &crimson.ListDRACsRequest{})
	if err != nil {
		return err
	}
	dracRes, err := registration.GetAllDracs(ctx)
	if err := compareDracs(ctx, writer, dracResp.Dracs, dracRes, dhcpMap); err != nil {
		return err
	}
	if dracRes == nil {
		return errors.New("drac entity corrupted")
	}

	hostResp, err := crimsonClient.ListPhysicalHosts(ctx, &crimson.ListPhysicalHostsRequest{})
	if err != nil {
		return err
	}
	hostRes, err := inventory.GetAllMachineLSEs(ctx)
	if hostRes == nil {
		return errors.New("host entity corrupted")
	}
	if err := compareHosts(ctx, writer, hostResp.Hosts, hostRes, stateMap, dhcpHostMap); err != nil {
		return err
	}

	vmResp, err := crimsonClient.ListVMs(ctx, &crimson.ListVMsRequest{})
	if err != nil {
		return err
	}
	vmRes, err := inventory.GetAllVMs(ctx)
	if vmRes == nil {
		return errors.New("vm entity corrupted")
	}
	if err := compareVMs(ctx, writer, vmResp.Vms, vmRes, stateMap, dhcpHostMap); err != nil {
		return err
	}
	return nil
}

func compareVMs(ctx context.Context, writer *storage.Writer, vms []*crimson.VM, vmRes *fleetds.OpResults, stateMap map[string]ufspb.State, dhcpHostMap map[string]*ufspb.DHCPConfig) error {
	logs := []string{"\n\n######## get-vm diff ############"}
	crimsonVMs := make(map[string]string)
	for _, r := range vms {
		name := r.GetName()
		crimsonVMs[name] = formatVM(name, r.GetIpv4(), r.GetHost(), r.GetOs(), util.ToState(r.GetState()))
	}
	ufsVMs := make(map[string]string)
	for _, r := range vmRes.Passed() {
		v := r.Data.(*ufspb.VM)
		vmName := v.GetName()
		resourceName := util.AddPrefix(util.VMCollection, vmName)
		ufsVMs[vmName] = formatVM(vmName, dhcpHostMap[vmName].GetIp(), v.GetMachineLseId(), v.GetOsVersion().GetValue(), stateMap[resourceName])
	}
	return logDiff(crimsonVMs, ufsVMs, writer, logs)
}

func formatVM(name, ip, machinelse, os string, state ufspb.State) string {
	return fmt.Sprintf("%s\t%s\t%s\t%s\t%s", util.FormatDHCPHostname(name), ip, util.FormatDHCPHostname(machinelse), os, strings.ToLower(state.String()))
}

func compareHosts(ctx context.Context, writer *storage.Writer, hosts []*crimson.PhysicalHost, hostRes *fleetds.OpResults, stateMap map[string]ufspb.State, dhcpHostMap map[string]*ufspb.DHCPConfig) error {
	logs := []string{"\n\n######## get-host diff ############"}
	crimsonHosts := make(map[string]string)
	for _, r := range hosts {
		name := r.GetName()
		crimsonHosts[name] = formatHost(name, r.GetIpv4(), r.GetMachine(), r.GetOs(), r.GetVmSlots(), util.ToState(r.GetState()))
	}
	ufsHosts := make(map[string]string)
	for _, r := range hostRes.Passed() {
		m := r.Data.(*ufspb.MachineLSE)
		if m.GetChromeBrowserMachineLse() != nil {
			bm := m.GetChromeBrowserMachineLse()
			name := m.GetName()
			resourceName := util.AddPrefix(util.HostCollection, name)
			ufsHosts[name] = formatHost(name, dhcpHostMap[name].GetIp(), m.GetMachines()[0], bm.GetOsVersion().GetValue(), bm.GetVmCapacity(), stateMap[resourceName])
		}
	}
	return logDiff(crimsonHosts, ufsHosts, writer, logs)
}

func formatHost(name, ip, machine, os string, vmSlots int32, state ufspb.State) string {
	return fmt.Sprintf("%s\t%s\t%s\t%s\t%d\t%s", util.FormatDHCPHostname(name), ip, machine, os, vmSlots, strings.ToLower(state.String()))
}

func compareDracs(ctx context.Context, writer *storage.Writer, dracs []*crimson.DRAC, dracRes *fleetds.OpResults, dhcpMap map[string]*ufspb.DHCPConfig) error {
	logs := []string{"\n\n######## get-drac diff ############"}
	crimsonDracs := make(map[string]string)
	for _, r := range dracs {
		crimsonDracs[r.GetName()] = formatDrac(r.GetName(), r.GetMacAddress(), r.GetSwitch(), util.Int32ToStr(r.GetSwitchport()), r.GetIpv4())
	}
	ufsDracs := make(map[string]string)
	for _, r := range dracRes.Passed() {
		m := r.Data.(*ufspb.Drac)
		name := m.GetName()
		si := m.GetSwitchInterface()
		ufsDracs[name] = formatDrac(name, m.GetMacAddress(), si.GetSwitch(), si.GetPortName(), dhcpMap[m.GetMacAddress()].GetIp())
	}
	return logDiff(crimsonDracs, ufsDracs, writer, logs)
}

func formatDrac(name, macAddr, sw, port string, ip string) string {
	return fmt.Sprintf("%s\t%s\t%s\t%s\t%s", util.FormatDHCPHostname(name), macAddr, sw, port, ip)
}

func compareNics(ctx context.Context, writer *storage.Writer, nics []*crimson.NIC, nicRes *fleetds.OpResults) error {
	logs := []string{"\n\n######## get-nic diff ############"}
	crimsonNics := make(map[string]string)
	for _, r := range nics {
		if strings.Contains(r.GetName(), "drac") {
			continue
		}
		name := util.GetNicName(r.GetName(), r.GetMachine())
		crimsonNics[name] = formatNic(name, r.GetMacAddress(), r.GetSwitch(), util.Int32ToStr(r.GetSwitchport()))
	}
	ufsNics := make(map[string]string)
	for _, r := range nicRes.Passed() {
		m := r.Data.(*ufspb.Nic)
		name := m.GetName()
		si := m.GetSwitchInterface()
		ufsNics[name] = formatNic(name, m.GetMacAddress(), si.GetSwitch(), si.GetPortName())
	}
	return logDiff(crimsonNics, ufsNics, writer, logs)
}

func formatNic(name, macAddr, sw, port string) string {
	return fmt.Sprintf("%s\t%s\t%s\t%s", name, macAddr, sw, port)
}

func compareVlans(ctx context.Context, writer *storage.Writer, vlans []*crimson.VLAN, vlanRes *fleetds.OpResults, stateMap map[string]ufspb.State) error {
	logs := []string{"\n\n######## get-vlan diff ############"}
	crimsonVlans := make(map[string]string)
	for _, r := range vlans {
		name := util.GetBrowserLabName(util.Int64ToStr(r.GetId()))
		crimsonVlans[name] = formatVlan(name, r.GetCidrBlock(), r.GetAlias(), util.ToState(r.GetState()))
	}
	ufsVlans := make(map[string]string)
	for _, r := range vlanRes.Passed() {
		m := r.Data.(*ufspb.Vlan)
		name := m.GetName()
		if util.IsInBrowserZone(name) {
			ufsVlans[name] = formatVlan(name, m.GetVlanAddress(), m.GetDescription(), stateMap[name])
		}
	}
	return logDiff(crimsonVlans, ufsVlans, writer, logs)
}

func formatVlan(id, cidr, alias string, state ufspb.State) string {
	return fmt.Sprintf("%s\t%s\t%s\t%s", id, cidr, alias, strings.ToLower(state.String()))
}

func compareSwitches(ctx context.Context, writer *storage.Writer, switches []*crimson.Switch, switchRes *fleetds.OpResults, stateMap map[string]ufspb.State) error {
	logs := []string{"\n\n######## get-switch diff ############"}
	crimsonSwitches := make(map[string]string)
	for _, r := range switches {
		crimsonSwitches[r.GetName()] = formatSwitch(r.GetName(), r.GetRack(), r.GetDescription(), util.ToState(r.GetState()), r.GetPorts())
	}
	ufsSwitches := make(map[string]string)
	for _, r := range switchRes.Passed() {
		m := r.Data.(*ufspb.Switch)
		name := m.GetName()
		cState := stateMap[util.AddPrefix(util.SwitchCollection, m.GetName())]
		ufsSwitches[name] = formatSwitch(name, m.GetRack(), m.GetDescription(), cState, m.GetCapacityPort())
	}
	return logDiff(crimsonSwitches, ufsSwitches, writer, logs)
}

func formatSwitch(name, rack, description string, state ufspb.State, port int32) string {
	return fmt.Sprintf("%s\t%s\t%s\t%s\t%d", name, rack, description, strings.ToLower(state.String()), port)
}

func compareKVMs(ctx context.Context, writer *storage.Writer, kvms []*crimson.KVM, kvmRes *fleetds.OpResults, stateMap map[string]ufspb.State) error {
	logs := []string{"\n\n######## get-kvm diff ############"}
	crimsonKVMs := make(map[string]string)
	for _, r := range kvms {
		crimsonKVMs[r.GetName()] = formatKVM(r.GetName(), r.GetPlatform(), r.GetMacAddress(), util.ToState(r.GetState()))
	}
	ufsKVMs := make(map[string]string)
	for _, r := range kvmRes.Passed() {
		m := r.Data.(*ufspb.KVM)
		name := m.GetName()
		cState := stateMap[util.AddPrefix(util.KVMCollection, m.GetName())]
		ufsKVMs[name] = formatKVM(name, m.GetChromePlatform(), m.GetMacAddress(), cState)
	}
	return logDiff(crimsonKVMs, ufsKVMs, writer, logs)
}

func formatKVM(name, platform, macAddr string, state ufspb.State) string {
	return fmt.Sprintf("%s\t%s\t%s\t%s", util.FormatDHCPHostname(name), util.FormatResourceName(platform), macAddr, strings.ToLower(state.String()))
}

func compareMachines(ctx context.Context, writer *storage.Writer, machines []*crimson.Machine, machineRes *fleetds.OpResults, stateMap map[string]ufspb.State) error {
	logs := []string{"\n\n######## get-machine diff ############"}
	crimsonMachines := make(map[string]string)
	for _, r := range machines {
		crimsonMachines[r.GetName()] = formatMachine(r.GetName(), r.GetRack(), util.ToZone(r.GetDatacenter()), util.ToState(r.GetState()))
	}
	ufsMachines := make(map[string]string)
	for _, r := range machineRes.Passed() {
		m := r.Data.(*ufspb.Machine)
		if m.GetChromeBrowserMachine() != nil {
			rack := m.GetLocation().GetRack()
			resourceName := util.AddPrefix(util.MachineCollection, m.GetName())
			ufsMachines[m.GetName()] = formatMachine(m.GetName(), rack, m.GetLocation().GetZone(), stateMap[resourceName])
		}
	}
	return logDiff(crimsonMachines, ufsMachines, writer, logs)
}

func formatMachine(name, rack string, zone ufspb.Zone, state ufspb.State) string {
	return fmt.Sprintf("%s\t%s\t%s\t%s", name, rack, zone.String(), strings.ToLower(state.String()))
}

func compareRacks(ctx context.Context, writer *storage.Writer, racks []*crimson.Rack, rackRes *fleetds.OpResults, stateMap map[string]ufspb.State) error {
	logs := []string{"\n\n######## get-rack diff ############"}
	crimsonRacks := make(map[string]string)
	for _, r := range racks {
		crimsonRacks[r.GetName()] = formatRack(r.GetName(), util.ToZone(r.GetDatacenter()), util.ToState(r.GetState()), r.GetKvm())
	}
	ufsRacks := make(map[string]string)
	for _, r := range rackRes.Passed() {
		rack := r.Data.(*ufspb.Rack)
		if rack.GetChromeBrowserRack() != nil {
			resourceName := util.AddPrefix(util.RackCollection, rack.GetName())
			kvms, _ := registration.QueryKVMByPropertyName(ctx, "rack", rack.GetName(), true)
			kvm := ""
			if len(kvms) > 0 {
				kvm = kvms[0].GetName()
			}
			ufsRacks[rack.GetName()] = formatRack(rack.GetName(), rack.GetLocation().GetZone(), stateMap[resourceName], kvm)
		}
	}
	return logDiff(crimsonRacks, ufsRacks, writer, logs)
}

func formatRack(rackName string, zone ufspb.Zone, state ufspb.State, kvm string) string {
	return fmt.Sprintf("%s\t%s\t%s\t%s", rackName, zone.String(), strings.ToLower(state.String()), kvm)
}

func logDiff(crimsonData, ufsData map[string]string, writer *storage.Writer, logs []string) error {
	logs = append(logs, "Resources in crimson but not in UFS:")
	var diffs []string
	for k, v := range crimsonData {
		if v2, ok := ufsData[k]; !ok {
			logs = append(logs, v)
		} else if v != v2 {
			diffs = append(diffs, v, v2)
		}
	}
	logs = append(logs, "Resources in UFS but not in crimson:")
	for k, v := range ufsData {
		if _, ok := crimsonData[k]; !ok {
			logs = append(logs, v)
		}
	}
	logs = append(logs, "Resources in both UFS and crimson but has difference (crimson first):")
	logs = append(logs, diffs...)
	if _, err := fmt.Fprintf(writer, strings.Join(logs, "\n")); err != nil {
		return err
	}
	return nil
}

func getCurTimeFilename() string {
	// Use PST timezone to generate time-related filename
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err == nil {
		return time.Now().UTC().In(loc).Format("2006-01-02T03:04:05")
	}
	return time.Now().UTC().Format("2006-01-02T03:04:05")
}

func getCloudStorageWriter(ctx context.Context, filename string) (*storage.Writer, error) {
	bucketName := config.Get(ctx).SelfStorageBucket
	if bucketName == "" {
		bucketName = "unified-fleet-system.appspot.com"
	}
	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		logging.Warningf(ctx, "failed to create cloud storage client")
		return nil, err
	}
	bucket := storageClient.Bucket(bucketName)
	logging.Infof(ctx, "All diff will be saved to https://storage.cloud.google.com/%s/%s", bucketName, filename)
	return bucket.Object(filename).NewWriter(ctx), nil
}
