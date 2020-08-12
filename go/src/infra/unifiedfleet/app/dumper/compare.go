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
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/grpc/prpc"
	crimson "go.chromium.org/luci/machine-db/api/crimson/v1"
	"go.chromium.org/luci/server/auth"

	ufspb "infra/unifiedfleet/api/v1/proto"
	"infra/unifiedfleet/app/config"
	"infra/unifiedfleet/app/model/configuration"
	fleetds "infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/model/inventory"
	"infra/unifiedfleet/app/model/registration"
	"infra/unifiedfleet/app/model/state"
	"infra/unifiedfleet/app/util"
)

func compareCrimson(ctx context.Context, machineDBHost string) error {
	writer, err := getCloudStorageWriter(ctx)
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
	stateMap := make(map[string]ufspb.State)
	for _, sr := range stateRes.Passed() {
		s := sr.Data.(*ufspb.StateRecord)
		stateMap[s.GetResourceName()] = s.GetState()
	}
	dhcpRes, err := configuration.GetAllDHCPs(ctx)
	if err != nil {
		return err
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
	vlanResp, err := crimsonClient.ListVLANs(ctx, &crimson.ListVLANsRequest{})
	if err != nil {
		return err
	}
	vlanRes, err := configuration.GetAllVlans(ctx)
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

	switchResp, err := crimsonClient.ListSwitches(ctx, &crimson.ListSwitchesRequest{})
	if err != nil {
		return err
	}
	switchRes, err := registration.GetAllSwitches(ctx)
	if err := compareSwitches(ctx, writer, switchResp.Switches, switchRes, stateMap); err != nil {
		return err
	}

	nicResp, err := crimsonClient.ListNICs(ctx, &crimson.ListNICsRequest{})
	if err != nil {
		return err
	}
	nicRes, err := registration.GetAllNics(ctx)
	if err := compareNics(ctx, writer, nicResp.Nics, nicRes); err != nil {
		return err
	}

	dracResp, err := crimsonClient.ListDRACs(ctx, &crimson.ListDRACsRequest{})
	if err != nil {
		return err
	}
	dracRes, err := registration.GetAllDracs(ctx)
	if err := compareDracs(ctx, writer, dracResp.Dracs, dracRes, dhcpMap); err != nil {
		return err
	}

	hostResp, err := crimsonClient.ListPhysicalHosts(ctx, &crimson.ListPhysicalHostsRequest{})
	if err != nil {
		return err
	}
	hostRes, err := inventory.GetAllMachineLSEs(ctx)
	if err := compareHosts(ctx, writer, hostResp.Hosts, hostRes, stateMap, dhcpHostMap); err != nil {
		return err
	}

	vmResp, err := crimsonClient.ListVMs(ctx, &crimson.ListVMsRequest{})
	if err != nil {
		return err
	}
	if err := compareVMs(ctx, writer, vmResp.Vms, hostRes, stateMap, dhcpHostMap); err != nil {
		return err
	}

	return nil
}

func compareVMs(ctx context.Context, writer *storage.Writer, vms []*crimson.VM, hostRes *fleetds.OpResults, stateMap map[string]ufspb.State, dhcpHostMap map[string]*ufspb.DHCPConfig) error {
	logs := []string{"\n\n######## get-vm diff ############"}
	crimsonVMs := make(map[string]string)
	for _, r := range vms {
		name := r.GetName()
		crimsonVMs[name] = formatVM(name, r.GetIpv4(), r.GetHost(), r.GetOs(), util.ToState(r.GetState()))
	}
	ufsVMs := make(map[string]string)
	for _, r := range hostRes.Passed() {
		m := r.Data.(*ufspb.MachineLSE)
		if m.GetChromeBrowserMachineLse() != nil {
			for _, v := range m.GetChromeBrowserMachineLse().GetVms() {
				vmName := v.GetName()
				resourceName := util.AddPrefix(util.VMCollection, vmName)
				ufsVMs[vmName] = formatVM(vmName, dhcpHostMap[vmName].GetIp(), m.GetName(), v.GetOsVersion().GetValue(), stateMap[resourceName])
			}
		}
	}
	return logDiff(crimsonVMs, ufsVMs, writer, logs)
}

func formatVM(name, ip, machine, os string, state ufspb.State) string {
	return fmt.Sprintf("%s\t%s\t%s\t%s\t%s", name, ip, machine, os, strings.ToLower(state.String()))
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
	return fmt.Sprintf("%s\t%s\t%s\t%s\t%d\t%s", name, ip, machine, os, vmSlots, strings.ToLower(state.String()))
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
	return fmt.Sprintf("%s\t%s\t%s\t%s\t%s", name, macAddr, sw, port, ip)
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
		if util.IsInBrowserLab(name) {
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
	return fmt.Sprintf("%s\t%s\t%s\t%s", name, util.FormatResourceName(platform), macAddr, strings.ToLower(state.String()))
}

func compareMachines(ctx context.Context, writer *storage.Writer, machines []*crimson.Machine, machineRes *fleetds.OpResults, stateMap map[string]ufspb.State) error {
	logs := []string{"\n\n######## get-machine diff ############"}
	crimsonMachines := make(map[string]string)
	for _, r := range machines {
		crimsonMachines[r.GetName()] = formatMachine(r.GetName(), r.GetRack(), util.ToLab(r.GetDatacenter()), util.ToState(r.GetState()))
	}
	ufsMachines := make(map[string]string)
	for _, r := range machineRes.Passed() {
		m := r.Data.(*ufspb.Machine)
		if m.GetChromeBrowserMachine() != nil {
			rack := m.GetLocation().GetRack()
			resourceName := util.AddPrefix(util.MachineCollection, m.GetName())
			ufsMachines[m.GetName()] = formatMachine(m.GetName(), rack, m.GetLocation().GetLab(), stateMap[resourceName])
		}
	}
	return logDiff(crimsonMachines, ufsMachines, writer, logs)
}

func formatMachine(name, rack string, lab ufspb.Lab, state ufspb.State) string {
	return fmt.Sprintf("%s\t%s\t%s\t%s", name, rack, lab.String(), strings.ToLower(state.String()))
}

func compareRacks(ctx context.Context, writer *storage.Writer, racks []*crimson.Rack, rackRes *fleetds.OpResults, stateMap map[string]ufspb.State) error {
	logs := []string{"\n\n######## get-rack diff ############"}
	crimsonRacks := make(map[string]string)
	for _, r := range racks {
		crimsonRacks[r.GetName()] = formatRack(r.GetName(), util.ToLab(r.GetDatacenter()), util.ToState(r.GetState()), r.GetKvm())
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
			ufsRacks[rack.GetName()] = formatRack(rack.GetName(), rack.GetLocation().GetLab(), stateMap[resourceName], kvm)
		}
	}
	return logDiff(crimsonRacks, ufsRacks, writer, logs)
}

func formatRack(rackName string, lab ufspb.Lab, state ufspb.State, kvm string) string {
	return fmt.Sprintf("%s\t%s\t%s\t%s", rackName, lab.String(), strings.ToLower(state.String()), kvm)
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
	logs = append(logs, "Resources in both UFS and crimson but has difference:")
	logs = append(logs, diffs...)
	if _, err := fmt.Fprintf(writer, strings.Join(logs, "\n")); err != nil {
		return err
	}
	return nil
}

func getCloudStorageWriter(ctx context.Context) (*storage.Writer, error) {
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
	filename := fmt.Sprintf("crimson_ufs_diff.%s.log", time.Now().UTC().Format("2006-01-02T03:04:05"))
	logging.Infof(ctx, "All diff will be saved to https://storage.cloud.google.com/%s/%s", bucketName, filename)
	return bucket.Object(filename).NewWriter(ctx), nil
}
