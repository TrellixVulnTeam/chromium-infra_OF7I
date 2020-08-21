// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package utils

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"google.golang.org/protobuf/encoding/protojson"

	ufspb "infra/unifiedfleet/api/v1/proto"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	ufsUtil "infra/unifiedfleet/app/util"
)

// Titles for printing table format list
var (
	SwitchTitle     = []string{"Switch Name", "CapacityPort", "Zone", "Rack", "State", "UpdateTime"}
	SwitchFullTitle = []string{"Switch Name", "CapacityPort", "Zone", "Rack", "State", "nics", "dracs", "UpdateTime"}
	KvmTitle        = []string{"KVM Name", "MAC Address", "ChromePlatform", "CapacityPort", "Zone", "Rack", "State", "UpdateTime"}
	KvmFullTitle    = []string{"KVM Name", "MAC Address", "ChromePlatform", "CapacityPort", "IP", "Vlan", "State", "Zone", "Rack", "UpdateTime"}
	RpmTitle        = []string{"RPM Name", "MAC Address", "CapacityPort",
		"UpdateTime"}
	DracTitle = []string{"Drac Name", "Display name", "MAC Address", "Switch",
		"Switch Port", "Password", "Zone", "Rack", "Machine", "UpdateTime"}
	DracFullTitle = []string{"Drac Name", "MAC Address", "Switch", "Switch Port", "Attached Host", "IP", "Vlan", "Zone", "Rack", "Machine", "UpdateTime"}
	NicTitle      = []string{"Nic Name", "MAC Address", "Switch", "Switch Port", "Zone", "Rack", "Machine", "UpdateTime"}
	NicFullTitle  = []string{"Nic Name", "MAC Address", "Switch", "Switch Port", "Attached Host", "IP", "Vlan", "Zone", "Rack", "Machine", "UpdateTime"}
	MachineTitle  = []string{"Machine Name", "Serial Number", "Zone", "Rack", "ChromePlatform",
		"Nics", "Drac", "DeploymentTicket", "Description", "State", "Realm", "UpdateTime"}
	BrowserMachineFullTitle = []string{
		"Machine Name", "Serial Number", "Host", "Zone", "Rack", "ChromePlatform",
		"Nics", "Drac", "kvms", "switches", "DeploymentTicket", "Description", "State", "Realm", "UpdateTime",
	}
	OSMachineFullTitle = []string{
		"Machine Name", "Zone", "Rack", "Barcode", "UpdateTime",
	}
	MachinelseprototypeTitle = []string{"Machine Prototype Name",
		"Occupied Capacity", "PeripheralTypes", "VirtualTypes",
		"Tags", "UpdateTime"}
	RacklseprototypeTitle = []string{"Rack Prototype Name", "PeripheralTypes",
		"Tags", "UpdateTime"}
	ChromePlatformTitle = []string{"Platform Name", "Manufacturer", "Description", "UpdateTime"}
	VlanTitle           = []string{"Vlan Name", "CIDR Block", "IP Capacity", "Description", "State", "UpdateTime"}
	VMTitle             = []string{"VM Name", "OS Version", "MAC Address", "Zone", "Host", "Vlan", "State", "Description", "UpdateTime"}
	VMFullTitle         = []string{"VM Name", "OS Version", "MAC Address", "Zone", "Host", "Vlan", "IP", "State", "Description", "UpdateTime"}
	RackTitle           = []string{"Rack Name", "Zone", "KVMs", "Switches", "RPMs", "Capacity", "State", "Realm", "UpdateTime"}
	MachineLSETitle     = []string{"Host", "OS Version", "Zone", "Virtual Datacenter", "Rack", "Machine(s)", "Nic", "State", "VM capacity", "VMs", "UpdateTime"}
	MachineLSETFullitle = []string{"Host", "OS Version", "Manufacturer", "Machine", "Zone", "Virtual Datacenter", "Rack", "Nic", "IP", "Vlan", "State", "VM capacity", "VMs", "UpdateTime"}
)

// TimeFormat for all timestamps handled by shivas
var timeFormat = "2006-01-02-15:04:05"

// The tab writer which defines the write format
var tw = tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)

// The io writer for json output
var bw = bufio.NewWriter(os.Stdout)

type printAll func(context.Context, ufsAPI.FleetClient, bool, int32, string, string, bool, bool, bool) (string, error)

// PrintListJSONFormat prints the list output in JSON format
func PrintListJSONFormat(ctx context.Context, ic ufsAPI.FleetClient, f printAll, json bool, pageSize int32, filter string, keysOnly, emit bool) error {
	var pageToken string
	fmt.Print("[")
	if pageSize == 0 {
		for {
			token, err := f(ctx, ic, json, ufsUtil.MaxPageSize, pageToken, filter, keysOnly, false, emit)
			if err != nil {
				return err
			}
			if token == "" {
				break
			}
			fmt.Print(",")
			pageToken = token
		}
	} else {
		for i := int32(0); i < pageSize; i = i + ufsUtil.MaxPageSize {
			var size int32
			if pageSize-i < ufsUtil.MaxPageSize {
				size = pageSize % ufsUtil.MaxPageSize
			} else {
				size = ufsUtil.MaxPageSize
			}
			token, err := f(ctx, ic, json, size, pageToken, filter, keysOnly, false, emit)
			if err != nil {
				return err
			}
			if token == "" {
				break
			} else if i+ufsUtil.MaxPageSize < pageSize {
				fmt.Print(",")
			}
			pageToken = token
		}
	}
	fmt.Println("]")
	return nil
}

// PrintListTableFormat prints list output in Table format
func PrintListTableFormat(ctx context.Context, ic ufsAPI.FleetClient, f printAll, json bool, pageSize int32, filter string, keysOnly bool, title []string, tsv bool) error {
	if !tsv {
		if keysOnly {
			PrintTitle(title[0:1])
		} else {
			PrintTitle(title)
		}
	}
	var pageToken string
	if pageSize == 0 {
		for {
			token, err := f(ctx, ic, json, ufsUtil.MaxPageSize, pageToken, filter, keysOnly, tsv, false)
			if err != nil {
				return err
			}
			if token == "" {
				break
			}
			pageToken = token
		}
	} else {
		for i := int32(0); i < pageSize; i = i + ufsUtil.MaxPageSize {
			var size int32
			if pageSize-i < ufsUtil.MaxPageSize {
				size = pageSize % ufsUtil.MaxPageSize
			} else {
				size = ufsUtil.MaxPageSize
			}
			token, err := f(ctx, ic, json, size, pageToken, filter, keysOnly, tsv, false)
			if err != nil {
				return err
			}
			if token == "" {
				break
			}
			pageToken = token
		}
	}
	return nil
}

// PrintProtoJSON prints the output as json
func PrintProtoJSON(pm proto.Message, emit bool) {
	defer bw.Flush()
	m := protojson.MarshalOptions{
		EmitUnpopulated: emit,
		Indent:          "\t",
	}
	json, err := m.Marshal(proto.MessageV2(pm))
	if err != nil {
		fmt.Println("Failed to marshal JSON")
	} else {
		bw.Write(json)
		fmt.Fprintf(bw, "\n")
	}
}

// PrintTitle prints the title fields in table form.
func PrintTitle(title []string) {
	for _, s := range title {
		fmt.Fprint(tw, fmt.Sprintf("%s\t", s))
	}
	fmt.Fprintln(tw)
}

// PrintSwitches prints the all switches in table form.
func PrintSwitches(switches []*ufspb.Switch, keysOnly bool) {
	defer tw.Flush()
	for _, s := range switches {
		printSwitch(s, keysOnly)
	}
}

func switchFullOutputStrs(sw *ufspb.Switch, nics []*ufspb.Nic, dracs []*ufspb.Drac) []string {
	var ts string
	if t, err := ptypes.Timestamp(sw.GetUpdateTime()); err == nil {
		ts = t.Format(timeFormat)
	}
	return []string{
		ufsUtil.RemovePrefix(sw.GetName()),
		fmt.Sprintf("%d", sw.GetCapacityPort()),
		sw.GetZone(),
		sw.GetRack(),
		sw.GetState(),
		strSlicesToStr(ufsAPI.ParseResources(nics, "Name")),
		strSlicesToStr(ufsAPI.ParseResources(dracs, "Name")),
		ts,
	}
}

// PrintSwitchFull prints the full related msg for a switch
func PrintSwitchFull(sw *ufspb.Switch, nics []*ufspb.Nic, dracs []*ufspb.Drac) {
	defer tw.Flush()
	var out string
	for _, s := range switchFullOutputStrs(sw, nics, dracs) {
		out += fmt.Sprintf("%s\t", s)
	}
	fmt.Fprintln(tw, out)
}

func switchOutputStrs(pm proto.Message) []string {
	m := pm.(*ufspb.Switch)
	var ts string
	if t, err := ptypes.Timestamp(m.GetUpdateTime()); err == nil {
		ts = t.Format(timeFormat)
	}
	return []string{
		ufsUtil.RemovePrefix(m.GetName()),
		fmt.Sprintf("%d", m.GetCapacityPort()),
		m.GetZone(),
		m.GetRack(),
		m.GetState(),
		ts,
	}
}

func printSwitch(sw *ufspb.Switch, keysOnly bool) {
	if keysOnly {
		fmt.Fprintln(tw, ufsUtil.RemovePrefix(sw.Name))
		return
	}
	var out string
	for _, s := range switchOutputStrs(sw) {
		out += fmt.Sprintf("%s\t", s)
	}
	fmt.Fprintln(tw, out)
}

// PrintSwitchesJSON prints the switch details in json format.
func PrintSwitchesJSON(switches []*ufspb.Switch, emit bool) {
	len := len(switches) - 1
	for i, s := range switches {
		s.Name = ufsUtil.RemovePrefix(s.Name)
		PrintProtoJSON(s, emit)
		if i < len {
			fmt.Print(",")
			fmt.Println()
		}
	}
}

func kvmFullOutputStrs(kvm *ufspb.KVM, dhcp *ufspb.DHCPConfig) []string {
	var ts string
	if t, err := ptypes.Timestamp(kvm.GetUpdateTime()); err == nil {
		ts = t.Format(timeFormat)
	}
	return []string{
		ufsUtil.RemovePrefix(kvm.Name),
		kvm.GetMacAddress(),
		kvm.GetChromePlatform(),
		fmt.Sprintf("%d", kvm.GetCapacityPort()),
		dhcp.GetIp(),
		dhcp.GetVlan(),
		kvm.GetState(),
		kvm.GetZone(),
		kvm.GetRack(),
		ts,
	}
}

// PrintKVMFull prints the full info for kvm
func PrintKVMFull(kvm *ufspb.KVM, dhcp *ufspb.DHCPConfig) {
	defer tw.Flush()
	var out string
	for _, s := range kvmFullOutputStrs(kvm, dhcp) {
		out += fmt.Sprintf("%s\t", s)
	}
	fmt.Fprintln(tw, out)
}

// PrintKVMs prints the all kvms in table form.
func PrintKVMs(kvms []*ufspb.KVM, keysOnly bool) {
	defer tw.Flush()
	for _, kvm := range kvms {
		printKVM(kvm, keysOnly)
	}
}

func kvmOutputStrs(pm proto.Message) []string {
	m := pm.(*ufspb.KVM)
	var ts string
	if t, err := ptypes.Timestamp(m.GetUpdateTime()); err == nil {
		ts = t.Format(timeFormat)
	}
	return []string{
		ufsUtil.RemovePrefix(m.GetName()),
		m.GetMacAddress(),
		m.GetChromePlatform(),
		fmt.Sprintf("%d", m.GetCapacityPort()),
		m.GetZone(),
		m.GetRack(),
		m.GetState(),
		ts,
	}
}

func printKVM(kvm *ufspb.KVM, keysOnly bool) {
	if keysOnly {
		fmt.Fprintln(tw, ufsUtil.RemovePrefix(kvm.Name))
		return
	}
	var out string
	for _, s := range kvmOutputStrs(kvm) {
		out += fmt.Sprintf("%s\t", s)
	}
	fmt.Fprintln(tw, out)
}

// PrintKVMsJSON prints the kvm details in json format.
func PrintKVMsJSON(kvms []*ufspb.KVM, emit bool) {
	len := len(kvms) - 1
	for i, s := range kvms {
		s.Name = ufsUtil.RemovePrefix(s.Name)
		PrintProtoJSON(s, emit)
		if i < len {
			fmt.Print(",")
			fmt.Println()
		}
	}
}

// PrintRPMs prints the all rpms in table form.
func PrintRPMs(rpms []*ufspb.RPM, keysOnly bool) {
	defer tw.Flush()
	for _, rpm := range rpms {
		printRPM(rpm, keysOnly)
	}
}

func rpmOutputStrs(pm proto.Message) []string {
	m := pm.(*ufspb.RPM)
	var ts string
	if t, err := ptypes.Timestamp(m.GetUpdateTime()); err == nil {
		ts = t.Format(timeFormat)
	}
	return []string{
		ufsUtil.RemovePrefix(m.GetName()),
		m.GetMacAddress(),
		fmt.Sprintf("%d", m.GetCapacityPort()),
		m.GetZone(),
		m.GetRack(),
		m.GetState(),
		ts,
	}
}

func printRPM(rpm *ufspb.RPM, keysOnly bool) {
	if keysOnly {
		fmt.Fprintln(tw, ufsUtil.RemovePrefix(rpm.Name))
		return
	}
	var out string
	for _, s := range rpmOutputStrs(rpm) {
		out += fmt.Sprintf("%s\t", s)
	}
	fmt.Fprintln(tw, out)
}

// PrintRPMsJSON prints the rpm details in json format.
func PrintRPMsJSON(rpms []*ufspb.RPM, emit bool) {
	len := len(rpms) - 1
	for i, s := range rpms {
		s.Name = ufsUtil.RemovePrefix(s.Name)
		PrintProtoJSON(s, emit)
		if i < len {
			fmt.Print(",")
			fmt.Println()
		}
	}
}

func dracFullOutputStrs(m *ufspb.Drac, dhcp *ufspb.DHCPConfig) []string {
	var ts string
	if t, err := ptypes.Timestamp(m.GetUpdateTime()); err == nil {
		ts = t.Format(timeFormat)
	}
	return []string{
		ufsUtil.RemovePrefix(m.Name),
		m.GetMacAddress(),
		m.GetSwitchInterface().GetSwitch(),
		m.GetSwitchInterface().GetPortName(),
		dhcp.GetHostname(),
		dhcp.GetIp(),
		dhcp.GetVlan(),
		m.GetZone(),
		m.GetRack(),
		m.GetMachine(),
		ts,
	}
}

// PrintDracFull prints the full related msg for drac
func PrintDracFull(drac *ufspb.Drac, dhcp *ufspb.DHCPConfig) {
	defer tw.Flush()
	var out string
	for _, s := range dracFullOutputStrs(drac, dhcp) {
		out += fmt.Sprintf("%s\t", s)
	}
	fmt.Fprintln(tw, out)
}

// PrintDracs prints the all dracs in table form.
func PrintDracs(dracs []*ufspb.Drac, keysOnly bool) {
	defer tw.Flush()
	for _, drac := range dracs {
		printDrac(drac, keysOnly)
	}
}

func dracOutputStrs(pm proto.Message) []string {
	m := pm.(*ufspb.Drac)
	var ts string
	if t, err := ptypes.Timestamp(m.GetUpdateTime()); err == nil {
		ts = t.Format(timeFormat)
	}
	return []string{
		ufsUtil.RemovePrefix(m.Name),
		m.GetDisplayName(),
		m.GetMacAddress(),
		m.GetSwitchInterface().GetSwitch(),
		m.GetSwitchInterface().GetPortName(),
		m.GetPassword(),
		m.GetZone(),
		m.GetRack(),
		m.GetMachine(),
		ts,
	}
}

func printDrac(drac *ufspb.Drac, keysOnly bool) {
	if keysOnly {
		fmt.Fprintln(tw, ufsUtil.RemovePrefix(drac.Name))
		return
	}
	var out string
	for _, s := range dracOutputStrs(drac) {
		out += fmt.Sprintf("%s\t", s)
	}
	fmt.Fprintln(tw, out)
}

// PrintDracsJSON prints the drac details in json format.
func PrintDracsJSON(dracs []*ufspb.Drac, emit bool) {
	len := len(dracs) - 1
	for i, s := range dracs {
		s.Name = ufsUtil.RemovePrefix(s.Name)
		PrintProtoJSON(s, emit)
		if i < len {
			fmt.Print(",")
			fmt.Println()
		}
	}
}

func nicFullOutputStrs(nic *ufspb.Nic, dhcp *ufspb.DHCPConfig) []string {
	var ts string
	if t, err := ptypes.Timestamp(nic.GetUpdateTime()); err == nil {
		ts = t.Format(timeFormat)
	}
	return []string{
		ufsUtil.RemovePrefix(nic.Name),
		nic.GetMacAddress(),
		nic.GetSwitchInterface().GetSwitch(),
		nic.GetSwitchInterface().GetPortName(),
		dhcp.GetHostname(),
		dhcp.GetIp(),
		dhcp.GetVlan(),
		nic.GetZone(),
		nic.GetRack(),
		nic.GetMachine(),
		ts,
	}
}

// PrintNicFull prints the full related msg for nic
func PrintNicFull(nic *ufspb.Nic, dhcp *ufspb.DHCPConfig) {
	defer tw.Flush()
	var out string
	for _, s := range nicFullOutputStrs(nic, dhcp) {
		out += fmt.Sprintf("%s\t", s)
	}
	fmt.Fprintln(tw, out)
}

// PrintNics prints the all nics in table form.
func PrintNics(nics []*ufspb.Nic, keysOnly bool) {
	defer tw.Flush()
	for _, nic := range nics {
		printNic(nic, keysOnly)
	}
}

func nicOutputStrs(pm proto.Message) []string {
	m := pm.(*ufspb.Nic)
	var ts string
	if t, err := ptypes.Timestamp(m.GetUpdateTime()); err == nil {
		ts = t.Format(timeFormat)
	}
	return []string{
		ufsUtil.RemovePrefix(m.GetName()),
		m.GetMacAddress(),
		m.GetSwitchInterface().GetSwitch(),
		m.GetSwitchInterface().GetPortName(),
		m.GetZone(),
		m.GetRack(),
		m.GetMachine(),
		ts,
	}
}

func printNic(nic *ufspb.Nic, keysOnly bool) {
	if keysOnly {
		fmt.Fprintln(tw, ufsUtil.RemovePrefix(nic.Name))
		return
	}
	var out string
	for _, s := range nicOutputStrs(nic) {
		out += fmt.Sprintf("%s\t", s)
	}
	fmt.Fprintln(tw, out)
}

// PrintNicsJSON prints the nic details in json format.
func PrintNicsJSON(nics []*ufspb.Nic, emit bool) {
	len := len(nics) - 1
	for i, s := range nics {
		s.Name = ufsUtil.RemovePrefix(s.Name)
		PrintProtoJSON(s, emit)
		if i < len {
			fmt.Print(",")
			fmt.Println()
		}
	}
}

// PrintMachineFull prints the full machine info.
func PrintMachineFull(m *ufspb.Machine, lse *ufspb.MachineLSE, rack *ufspb.Rack) {
	defer tw.Flush()
	var out string
	for _, s := range machineFullOutputStrs(m, lse, rack) {
		out += fmt.Sprintf("%s\t", s)
	}
	fmt.Fprintln(tw, out)
}

func machineFullOutputStrs(m *ufspb.Machine, lse *ufspb.MachineLSE, rack *ufspb.Rack) []string {
	var ts string
	if t, err := ptypes.Timestamp(m.GetUpdateTime()); err == nil {
		ts = t.Format(timeFormat)
	}
	if m.GetChromeBrowserMachine() != nil {
		return []string{
			ufsUtil.RemovePrefix(m.GetName()),
			m.GetSerialNumber(),
			lse.GetName(),
			m.GetLocation().GetZone().String(),
			m.GetLocation().GetRack(),
			m.GetChromeBrowserMachine().GetChromePlatform(),
			strSlicesToStr(ufsAPI.ParseResources(m.GetChromeBrowserMachine().GetNicObjects(), "Name")),
			m.GetChromeBrowserMachine().GetDracObject().GetName(),
			strSlicesToStr(ufsAPI.ParseResources(rack.GetChromeBrowserRack().GetKvmObjects(), "Name")),
			strSlicesToStr(ufsAPI.ParseResources(rack.GetChromeBrowserRack().GetSwitchObjects(), "Name")),
			m.GetChromeBrowserMachine().GetDeploymentTicket(),
			m.GetChromeBrowserMachine().GetDescription(),
			m.GetState(),
			m.GetRealm(),
			ts,
		}
	}
	return []string{
		ufsUtil.RemovePrefix(m.GetName()),
		m.GetLocation().GetZone().String(),
		m.GetLocation().GetRack(),
		m.GetLocation().GetBarcodeName(),
		ts,
	}
}

// PrintMachines prints the all machines in table form.
func PrintMachines(machines []*ufspb.Machine, keysOnly bool) {
	defer tw.Flush()
	for _, m := range machines {
		printMachine(m, keysOnly)
	}
}

func machineOutputStrs(pm proto.Message) []string {
	m := pm.(*ufspb.Machine)
	var ts string
	if t, err := ptypes.Timestamp(m.GetUpdateTime()); err == nil {
		ts = t.Format(timeFormat)
	}
	return []string{
		ufsUtil.RemovePrefix(m.GetName()),
		m.GetSerialNumber(),
		m.GetLocation().GetZone().String(),
		m.GetLocation().GetRack(),
		m.GetChromeBrowserMachine().GetChromePlatform(),
		strSlicesToStr(ufsAPI.ParseResources(m.GetChromeBrowserMachine().GetNicObjects(), "Name")),
		m.GetChromeBrowserMachine().GetDracObject().GetName(),
		m.GetChromeBrowserMachine().GetDeploymentTicket(),
		m.GetChromeBrowserMachine().GetDescription(),
		m.GetState(),
		m.GetRealm(),
		ts,
	}
}

func printMachine(m *ufspb.Machine, keysOnly bool) {
	if keysOnly {
		fmt.Fprintln(tw, ufsUtil.RemovePrefix(m.Name))
		return
	}
	var out string
	for _, s := range machineOutputStrs(m) {
		out += fmt.Sprintf("%s\t", s)
	}
	fmt.Fprintln(tw, out)
}

// PrintMachinesJSON prints the machine details in json format.
func PrintMachinesJSON(machines []*ufspb.Machine, emit bool) {
	len := len(machines) - 1
	for i, m := range machines {
		m.Name = ufsUtil.RemovePrefix(m.Name)
		PrintProtoJSON(m, emit)
		if i < len {
			fmt.Print(",")
			fmt.Println()
		}
	}
}

// PrintMachineLSEPrototypes prints the all msleps in table form.
func PrintMachineLSEPrototypes(msleps []*ufspb.MachineLSEPrototype, keysOnly bool) {
	defer tw.Flush()
	for _, m := range msleps {
		printMachineLSEPrototype(m, keysOnly)
	}
}

func machineLSEPrototypeOutputStrs(pm proto.Message) []string {
	m := pm.(*ufspb.MachineLSEPrototype)
	var ts string
	if t, err := ptypes.Timestamp(m.GetUpdateTime()); err == nil {
		ts = t.Format(timeFormat)
	}
	res := []string{
		ufsUtil.RemovePrefix(m.GetName()),
		fmt.Sprintf("%d", m.GetOccupiedCapacityRu()),
	}
	prs := m.GetPeripheralRequirements()
	var peripheralTypes string
	for _, pr := range prs {
		peripheralTypes += fmt.Sprintf("%s,", pr.GetPeripheralType())
	}
	res = append(res, strings.TrimSuffix(peripheralTypes, ","))
	var virtualTypes string
	for _, vm := range m.GetVirtualRequirements() {
		virtualTypes += fmt.Sprintf("%s,", vm.GetVirtualType())
	}
	res = append(res, strings.TrimSuffix(virtualTypes, ","))
	res = append(res, fmt.Sprintf("%s", m.GetTags()))
	res = append(res, ts)
	return res
}

func printMachineLSEPrototype(m *ufspb.MachineLSEPrototype, keysOnly bool) {
	if keysOnly {
		fmt.Fprintln(tw, ufsUtil.RemovePrefix(m.Name))
		return
	}
	var out string
	for _, s := range machineLSEPrototypeOutputStrs(m) {
		out += fmt.Sprintf("%s\t", s)
	}
	fmt.Fprintln(tw, out)
}

// PrintMachineLSEPrototypesJSON prints the mslep details in json format.
func PrintMachineLSEPrototypesJSON(msleps []*ufspb.MachineLSEPrototype, emit bool) {
	len := len(msleps) - 1
	for i, m := range msleps {
		m.Name = ufsUtil.RemovePrefix(m.Name)
		PrintProtoJSON(m, emit)
		if i < len {
			fmt.Print(",")
			fmt.Println()
		}
	}
}

// PrintRackLSEPrototypes prints the all msleps in table form.
func PrintRackLSEPrototypes(msleps []*ufspb.RackLSEPrototype, keysOnly bool) {
	defer tw.Flush()
	for _, m := range msleps {
		printRackLSEPrototype(m, keysOnly)
	}
}

func rackLSEPrototypeOutputStrs(pm proto.Message) []string {
	m := pm.(*ufspb.RackLSEPrototype)
	var ts string
	if t, err := ptypes.Timestamp(m.GetUpdateTime()); err == nil {
		ts = t.Format(timeFormat)
	}
	res := []string{ufsUtil.RemovePrefix(m.GetName())}
	var peripheralTypes string
	for _, pr := range m.GetPeripheralRequirements() {
		peripheralTypes += fmt.Sprintf("%s,", pr.GetPeripheralType())
	}
	res = append(res, strings.TrimSuffix(peripheralTypes, ","))
	res = append(res, fmt.Sprintf("%s", m.GetTags()))
	res = append(res, ts)
	return res
}

func printRackLSEPrototype(m *ufspb.RackLSEPrototype, keysOnly bool) {
	if keysOnly {
		fmt.Fprintln(tw, ufsUtil.RemovePrefix(m.Name))
		return
	}
	var out string
	for _, s := range rackLSEPrototypeOutputStrs(m) {
		out += fmt.Sprintf("%s\t", s)
	}
	fmt.Fprintln(tw, out)
}

// PrintRackLSEPrototypesJSON prints the mslep details in json format.
func PrintRackLSEPrototypesJSON(rlseps []*ufspb.RackLSEPrototype, emit bool) {
	len := len(rlseps) - 1
	for i, m := range rlseps {
		m.Name = ufsUtil.RemovePrefix(m.Name)
		PrintProtoJSON(m, emit)
		if i < len {
			fmt.Print(",")
			fmt.Println()
		}
	}
}

// PrintVlansJSON prints the vlan details in json format.
func PrintVlansJSON(vlans []*ufspb.Vlan, emit bool) {
	len := len(vlans) - 1
	for i, m := range vlans {
		m.Name = ufsUtil.RemovePrefix(m.Name)
		PrintProtoJSON(m, emit)
		if i < len {
			fmt.Print(",")
			fmt.Println()
		}
	}
}

// PrintVlans prints the all vlans in table form.
func PrintVlans(vs []*ufspb.Vlan, keysOnly bool) {
	defer tw.Flush()
	for _, v := range vs {
		printVlan(v, keysOnly)
	}
}

func vlanOutputStrs(pm proto.Message) []string {
	m := pm.(*ufspb.Vlan)
	var ts string
	if t, err := ptypes.Timestamp(m.GetUpdateTime()); err == nil {
		ts = t.Format(timeFormat)
	}
	return []string{
		ufsUtil.RemovePrefix(m.GetName()),
		m.GetVlanAddress(),
		fmt.Sprintf("%d", m.GetCapacityIp()),
		m.GetDescription(),
		m.GetState(),
		ts,
	}
}

func printVlan(m *ufspb.Vlan, keysOnly bool) {
	if keysOnly {
		fmt.Fprintln(tw, ufsUtil.RemovePrefix(m.Name))
		return
	}
	var out string
	for _, s := range vlanOutputStrs(m) {
		out += fmt.Sprintf("%s\t", s)
	}
	fmt.Fprintln(tw, out)
}

// PrintChromePlatforms prints the all msleps in table form.
func PrintChromePlatforms(msleps []*ufspb.ChromePlatform, keysOnly bool) {
	defer tw.Flush()
	for _, m := range msleps {
		printChromePlatform(m, keysOnly)
	}
}

func platformOutputStrs(pm proto.Message) []string {
	m := pm.(*ufspb.ChromePlatform)
	var ts string
	if t, err := ptypes.Timestamp(m.GetUpdateTime()); err == nil {
		ts = t.Format(timeFormat)
	}
	return []string{
		ufsUtil.RemovePrefix(m.GetName()),
		m.GetManufacturer(),
		m.GetDescription(),
		ts,
	}
}

func printChromePlatform(m *ufspb.ChromePlatform, keysOnly bool) {
	if keysOnly {
		fmt.Fprintln(tw, ufsUtil.RemovePrefix(m.Name))
		return
	}
	var out string
	for _, s := range platformOutputStrs(m) {
		out += fmt.Sprintf("%s\t", s)
	}
	fmt.Fprintln(tw, out)
}

// PrintChromePlatformsJSON prints the mslep details in json format.
func PrintChromePlatformsJSON(msleps []*ufspb.ChromePlatform, emit bool) {
	len := len(msleps) - 1
	for i, m := range msleps {
		m.Name = ufsUtil.RemovePrefix(m.Name)
		PrintProtoJSON(m, emit)
		if i < len {
			fmt.Print(",")
			fmt.Println()
		}
	}
}

// PrintMachineLSEsJSON prints the machinelse details in json format.
func PrintMachineLSEsJSON(machinelses []*ufspb.MachineLSE, emit bool) {
	len := len(machinelses) - 1
	for i, m := range machinelses {
		m.Name = ufsUtil.RemovePrefix(m.Name)
		PrintProtoJSON(m, emit)
		if i < len {
			fmt.Print(",")
			fmt.Println()
		}
	}
}

func machineLSEFullOutputStrs(lse *ufspb.MachineLSE, machine *ufspb.Machine, dhcp *ufspb.DHCPConfig) []string {
	var ts string
	if t, err := ptypes.Timestamp(lse.GetUpdateTime()); err == nil {
		ts = t.Format(timeFormat)
	}
	return []string{
		ufsUtil.RemovePrefix(lse.GetName()),
		lse.GetChromeBrowserMachineLse().GetOsVersion().GetValue(),
		lse.GetManufacturer(),
		machine.GetName(),
		lse.GetZone(),
		lse.GetChromeBrowserMachineLse().GetVirtualDatacenter(),
		lse.GetRack(),
		lse.GetNic(),
		dhcp.GetIp(),
		dhcp.GetVlan(),
		lse.GetState(),
		fmt.Sprintf("%d", lse.GetChromeBrowserMachineLse().GetVmCapacity()),
		strSlicesToStr(ufsAPI.ParseResources(lse.GetChromeBrowserMachineLse().GetVms(), "Name")),
		ts,
	}
}

// PrintMachineLSEFull prints the full info for a host
func PrintMachineLSEFull(lse *ufspb.MachineLSE, machine *ufspb.Machine, dhcp *ufspb.DHCPConfig) {
	defer tw.Flush()
	var out string
	for _, s := range machineLSEFullOutputStrs(lse, machine, dhcp) {
		out += fmt.Sprintf("%s\t", s)
	}
	fmt.Fprintln(tw, out)
}

// PrintMachineLSEs prints the all machinelses in table form.
func PrintMachineLSEs(machinelses []*ufspb.MachineLSE, keysOnly bool) {
	defer tw.Flush()
	for _, m := range machinelses {
		m.Name = ufsUtil.RemovePrefix(m.Name)
		printMachineLSE(m, keysOnly)
	}
}

func machineLSEOutputStrs(pm proto.Message) []string {
	m := pm.(*ufspb.MachineLSE)
	var ts string
	if t, err := ptypes.Timestamp(m.GetUpdateTime()); err == nil {
		ts = t.Format(timeFormat)
	}
	machine := ""
	if len(m.GetMachines()) == 1 {
		machine = m.GetMachines()[0]
	}
	if len(m.GetMachines()) > 1 {
		machine = strSlicesToStr(m.GetMachines())
	}
	return []string{
		ufsUtil.RemovePrefix(m.GetName()),
		m.GetChromeBrowserMachineLse().GetOsVersion().GetValue(),
		m.GetZone(),
		m.GetChromeBrowserMachineLse().GetVirtualDatacenter(),
		m.GetRack(),
		machine,
		m.GetNic(),
		m.GetState(),
		fmt.Sprintf("%d", m.GetChromeBrowserMachineLse().GetVmCapacity()),
		strSlicesToStr(ufsAPI.ParseResources(m.GetChromeBrowserMachineLse().GetVms(), "Name")),
		ts,
	}
}

func printMachineLSE(m *ufspb.MachineLSE, keysOnly bool) {
	if keysOnly {
		fmt.Fprintln(tw, ufsUtil.RemovePrefix(m.Name))
		return
	}
	var out string
	for _, s := range machineLSEOutputStrs(m) {
		out += fmt.Sprintf("%s\t", s)
	}
	fmt.Fprintln(tw, out)
}

// PrintFreeVMs prints the all free slots in table form.
func PrintFreeVMs(ctx context.Context, ic ufsAPI.FleetClient, hosts []*ufspb.MachineLSE) {
	defer tw.Flush()
	PrintTitle([]string{"Host", "Os Version", "Manufacturer", "Vlan", "Zone", "Free slots", "State"})
	for _, h := range hosts {
		h.Name = ufsUtil.RemovePrefix(h.Name)
		printFreeVM(ctx, ic, h)
	}
}

func printFreeVM(ctx context.Context, ic ufsAPI.FleetClient, host *ufspb.MachineLSE) {
	res, _ := ic.GetDHCPConfig(ctx, &ufsAPI.GetDHCPConfigRequest{
		Hostname: host.GetName(),
	})
	out := fmt.Sprintf("%s\t", host.GetName())
	out += fmt.Sprintf("%s\t", host.GetChromeBrowserMachineLse().GetOsVersion().GetValue())
	out += fmt.Sprintf("%s\t", host.GetManufacturer())
	out += fmt.Sprintf("%s\t", res.GetVlan())
	out += fmt.Sprintf("%s\t", host.GetZone())
	out += fmt.Sprintf("%d\t", host.GetChromeBrowserMachineLse().GetVmCapacity())
	out += fmt.Sprintf("%s\t", host.GetState())
	fmt.Fprintln(tw, out)
}

func vmFullOutputStrs(vm *ufspb.VM, dhcp *ufspb.DHCPConfig) []string {
	var ts string
	if t, err := ptypes.Timestamp(vm.GetUpdateTime()); err == nil {
		ts = t.Format(timeFormat)
	}
	return []string{
		ufsUtil.RemovePrefix(vm.GetName()),
		vm.GetOsVersion().GetValue(),
		vm.GetMacAddress(),
		vm.GetZone(),
		vm.GetMachineLseId(),
		dhcp.GetVlan(),
		dhcp.GetIp(),
		vm.GetState(),
		vm.GetDescription(),
		ts,
	}
}

// PrintVMFull prints the full info for vm
func PrintVMFull(vm *ufspb.VM, dhcp *ufspb.DHCPConfig) {
	defer tw.Flush()
	var out string
	for _, s := range vmFullOutputStrs(vm, dhcp) {
		out += fmt.Sprintf("%s\t", s)
	}
	fmt.Fprintln(tw, out)
}

// PrintVMs prints the all vms in table form.
func PrintVMs(vms []*ufspb.VM, keysOnly bool) {
	defer tw.Flush()
	for _, vm := range vms {
		printVM(vm, keysOnly)
	}
}

func vmOutputStrs(pm proto.Message) []string {
	m := pm.(*ufspb.VM)
	var ts string
	if t, err := ptypes.Timestamp(m.GetUpdateTime()); err == nil {
		ts = t.Format(timeFormat)
	}
	return []string{
		ufsUtil.RemovePrefix(m.GetName()),
		m.GetOsVersion().GetValue(),
		m.GetMacAddress(),
		m.GetZone(),
		m.GetMachineLseId(),
		m.GetVlan(),
		m.GetState(),
		m.GetDescription(),
		ts,
	}
}

func printVM(vm *ufspb.VM, keysOnly bool) {
	if keysOnly {
		fmt.Fprintln(tw, ufsUtil.RemovePrefix(vm.Name))
		return
	}
	var out string
	for _, s := range vmOutputStrs(vm) {
		out += fmt.Sprintf("%s\t", s)
	}
	fmt.Fprintln(tw, out)
}

// PrintVMsJSON prints the vm details in json format.
func PrintVMsJSON(vms []*ufspb.VM, emit bool) {
	len := len(vms) - 1
	for i, m := range vms {
		m.Name = ufsUtil.RemovePrefix(m.Name)
		PrintProtoJSON(m, emit)
		if i < len {
			fmt.Print(",")
			fmt.Println()
		}
	}
}

// PrintRacks prints the all racks in table form.
func PrintRacks(racks []*ufspb.Rack, keysOnly bool) {
	defer tw.Flush()
	for _, m := range racks {
		printRack(m, keysOnly)
	}
}

func rackOutputStrs(pm proto.Message) []string {
	m := pm.(*ufspb.Rack)
	var ts string
	if t, err := ptypes.Timestamp(m.GetUpdateTime()); err == nil {
		ts = t.Format(timeFormat)
	}
	return []string{
		ufsUtil.RemovePrefix(m.GetName()),
		m.GetLocation().GetZone().String(),
		strSlicesToStr(ufsAPI.ParseResources(m.GetChromeBrowserRack().GetKvmObjects(), "Name")),
		strSlicesToStr(ufsAPI.ParseResources(m.GetChromeBrowserRack().GetSwitchObjects(), "Name")),
		strSlicesToStr(ufsAPI.ParseResources(m.GetChromeBrowserRack().GetRpmObjects(), "Name")),
		fmt.Sprintf("%d", m.GetCapacityRu()),
		m.GetState(),
		m.GetRealm(),
		ts,
	}
}

func printRack(m *ufspb.Rack, keysOnly bool) {
	m.Name = ufsUtil.RemovePrefix(m.Name)
	if keysOnly {
		fmt.Fprintln(tw, m.GetName())
		return
	}
	var out string
	for _, s := range rackOutputStrs(m) {
		out += fmt.Sprintf("%s\t", s)
	}
	fmt.Fprintln(tw, out)
}

// PrintRacksJSON prints the rack details in json format.
func PrintRacksJSON(racks []*ufspb.Rack, emit bool) {
	len := len(racks) - 1
	for i, m := range racks {
		m.Name = ufsUtil.RemovePrefix(m.Name)
		PrintProtoJSON(m, emit)
		if i < len {
			fmt.Print(",")
			fmt.Println()
		}
	}
}

func strSlicesToStr(slices []string) string {
	return strings.Join(slices, ",")
}
