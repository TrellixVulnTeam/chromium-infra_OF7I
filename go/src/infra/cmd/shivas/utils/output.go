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

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"

	ufspb "infra/unifiedfleet/api/v1/proto"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	ufsUtil "infra/unifiedfleet/app/util"
)

// Titles for printing table format list
var (
	SwitchTitle     = []string{"Switch Name", "CapacityPort", "Lab", "Rack", "State", "UpdateTime"}
	SwitchFullTitle = []string{"Switch Name", "CapacityPort", "Lab", "Rack", "State", "nics", "dracs", "UpdateTime"}
	KvmTitle        = []string{"KVM Name", "MAC Address", "ChromePlatform", "CapacityPort", "Lab", "Rack", "State", "UpdateTime"}
	KvmFullTitle    = []string{"KVM Name", "MAC Address", "ChromePlatform", "CapacityPort", "IP", "Vlan", "State", "Lab", "Rack", "UpdateTime"}
	RpmTitle        = []string{"RPM Name", "MAC Address", "CapacityPort",
		"UpdateTime"}
	DracTitle = []string{"Drac Name", "Display name", "MAC Address", "Switch",
		"Switch Port", "Password", "Lab", "Rack", "Machine", "UpdateTime"}
	DracFullTitle = []string{"Drac Name", "MAC Address", "Switch", "Switch Port", "Attached Host", "IP", "Vlan", "Lab", "Rack", "Machine", "UpdateTime"}
	NicTitle      = []string{"Nic Name", "MAC Address", "Switch", "Switch Port", "Lab", "Rack", "Machine", "UpdateTime"}
	NicFullTitle  = []string{"Nic Name", "MAC Address", "Switch", "Switch Port", "Attached Host", "IP", "Vlan", "Lab", "Rack", "Machine", "UpdateTime"}
	MachineTitle  = []string{"Machine Name", "Serial Number", "Lab", "Rack", "ChromePlatform",
		"Nics", "Drac", "DeploymentTicket", "Description", "State", "Realm", "UpdateTime"}
	BrowserMachineFullTitle = []string{
		"Machine Name", "Serial Number", "Host", "Lab", "Rack", "ChromePlatform",
		"Nics", "Drac", "kvms", "switches", "DeploymentTicket", "Description", "State", "Realm", "UpdateTime",
	}
	OSMachineFullTitle = []string{
		"Machine Name", "Lab", "Rack", "Barcode", "UpdateTime",
	}
	MachinelseprototypeTitle = []string{"Machine Prototype Name",
		"Occupied Capacity", "PeripheralTypes", "VirtualTypes",
		"Tags", "UpdateTime"}
	RacklseprototypeTitle = []string{"Rack Prototype Name", "PeripheralTypes",
		"Tags", "UpdateTime"}
	ChromePlatformTitle = []string{"Platform Name", "Manufacturer", "Description", "UpdateTime"}
	VlanTitle           = []string{"Vlan Name", "CIDR Block", "IP Capacity", "Description", "State", "UpdateTime"}
	VMTitle             = []string{"VM Name", "OS Version", "OS Desc", "MAC Address", "Lab", "Host", "Vlan", "State", "UpdateTime"}
	VMFullTitle         = []string{"VM Name", "OS Version", "OS Desc", "MAC Address", "Lab", "Host", "Vlan", "IP", "State", "UpdateTime"}
	RackTitle           = []string{"Rack Name", "Lab", "Switches", "KVMs", "RPMs", "Capacity", "State", "Realm", "UpdateTime"}
	MachineLSETitle     = []string{"Host", "OS Version", "Lab", "Rack", "Nic", "State", "VM capacity", "VMs", "UpdateTime"}
	MachineLSETFullitle = []string{"Host", "OS Version", "Machine", "Lab", "Rack", "Nic", "IP", "Vlan", "State", "VM capacity", "VMs", "UpdateTime"}
)

// TimeFormat for all timestamps handled by shivas
var timeFormat = "2006-01-02-15:04:05"

// The tab writer which defines the write format
var tw = tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)

// The io writer for json output
var bw = bufio.NewWriter(os.Stdout)

type printAll func(context.Context, ufsAPI.FleetClient, bool, int32, string, string, bool) (string, error)

// PrintListJSONFormat prints the list output in JSON format
func PrintListJSONFormat(ctx context.Context, ic ufsAPI.FleetClient, f printAll, json bool, pageSize int32, filter string, keysOnly bool) error {
	var pageToken string
	fmt.Print("[")
	if pageSize == 0 {
		for {
			token, err := f(ctx, ic, json, ufsUtil.MaxPageSize, pageToken, filter, keysOnly)
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
			token, err := f(ctx, ic, json, size, pageToken, filter, keysOnly)
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
			token, err := f(ctx, ic, json, ufsUtil.MaxPageSize, pageToken, filter, keysOnly)
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
			token, err := f(ctx, ic, json, size, pageToken, filter, keysOnly)
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
func PrintProtoJSON(pm proto.Message) {
	defer bw.Flush()
	m := jsonpb.Marshaler{
		EnumsAsInts: false,
		Indent:      "\t",
	}
	if err := m.Marshal(bw, pm); err != nil {
		fmt.Println("Failed to marshal JSON")
	} else {
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

// PrintSwitchFull prints the full related msg for a switch
func PrintSwitchFull(sw *ufspb.Switch, nics []*ufspb.Nic, dracs []*ufspb.Drac) {
	defer tw.Flush()
	var ts string
	if t, err := ptypes.Timestamp(sw.GetUpdateTime()); err == nil {
		ts = t.Format(timeFormat)
	}
	out := fmt.Sprintf("%s\t", ufsUtil.RemovePrefix(sw.GetName()))
	out += fmt.Sprintf("%d\t", sw.GetCapacityPort())
	out += fmt.Sprintf("%s\t", sw.GetLab())
	out += fmt.Sprintf("%s\t", sw.GetRack())
	out += fmt.Sprintf("%s\t", sw.GetState())
	out += fmt.Sprintf("%s\t", ufsAPI.ParseResources(nics, "Name"))
	out += fmt.Sprintf("%s\t", ufsAPI.ParseResources(dracs, "Name"))
	out += fmt.Sprintf("%s\t", ts)
	fmt.Fprintln(tw, out)
}

func printSwitch(s *ufspb.Switch, keysOnly bool) {
	if keysOnly {
		fmt.Fprintln(tw, ufsUtil.RemovePrefix(s.Name))
		return
	}
	var ts string
	if t, err := ptypes.Timestamp(s.GetUpdateTime()); err == nil {
		ts = t.Format(timeFormat)
	}
	s.Name = ufsUtil.RemovePrefix(s.Name)
	out := fmt.Sprintf("%s\t", s.GetName())
	out += fmt.Sprintf("%d\t", s.GetCapacityPort())
	out += fmt.Sprintf("%s\t", s.GetLab())
	out += fmt.Sprintf("%s\t", s.GetRack())
	out += fmt.Sprintf("%s\t", s.GetState())
	out += fmt.Sprintf("%s\t", ts)
	fmt.Fprintln(tw, out)
}

// PrintSwitchesJSON prints the switch details in json format.
func PrintSwitchesJSON(switches []*ufspb.Switch) {
	len := len(switches) - 1
	for i, s := range switches {
		s.Name = ufsUtil.RemovePrefix(s.Name)
		PrintProtoJSON(s)
		if i < len {
			fmt.Print(",")
			fmt.Println()
		}
	}
}

// PrintKVMFull prints the full info for kvm
func PrintKVMFull(kvm *ufspb.KVM, dhcp *ufspb.DHCPConfig, s *ufspb.StateRecord) {
	defer tw.Flush()
	var ts string
	if t, err := ptypes.Timestamp(kvm.GetUpdateTime()); err == nil {
		ts = t.Format(timeFormat)
	}
	out := fmt.Sprintf("%s\t", ufsUtil.RemovePrefix(kvm.Name))
	out += fmt.Sprintf("%s\t", kvm.GetMacAddress())
	out += fmt.Sprintf("%s\t", kvm.GetChromePlatform())
	out += fmt.Sprintf("%d\t", kvm.GetCapacityPort())
	out += fmt.Sprintf("%s\t", dhcp.GetIp())
	out += fmt.Sprintf("%s\t", dhcp.GetVlan())
	out += fmt.Sprintf("%s\t", s.GetState())
	out += fmt.Sprintf("%s\t", kvm.GetLab())
	out += fmt.Sprintf("%s\t", kvm.GetRack())
	out += fmt.Sprintf("%s\t", ts)
	fmt.Fprintln(tw, out)
}

// PrintKVMs prints the all kvms in table form.
func PrintKVMs(kvms []*ufspb.KVM, keysOnly bool) {
	defer tw.Flush()
	for _, kvm := range kvms {
		printKVM(kvm, keysOnly)
	}
}

func printKVM(kvm *ufspb.KVM, keysOnly bool) {
	if keysOnly {
		fmt.Fprintln(tw, ufsUtil.RemovePrefix(kvm.Name))
		return
	}
	var ts string
	if t, err := ptypes.Timestamp(kvm.GetUpdateTime()); err == nil {
		ts = t.Format(timeFormat)
	}
	out := fmt.Sprintf("%s\t", ufsUtil.RemovePrefix(kvm.Name))
	out += fmt.Sprintf("%s\t", kvm.GetMacAddress())
	out += fmt.Sprintf("%s\t", kvm.GetChromePlatform())
	out += fmt.Sprintf("%d\t", kvm.GetCapacityPort())
	out += fmt.Sprintf("%s\t", kvm.GetLab())
	out += fmt.Sprintf("%s\t", kvm.GetRack())
	out += fmt.Sprintf("%s\t", kvm.GetState())
	out += fmt.Sprintf("%s\t", ts)
	fmt.Fprintln(tw, out)
}

// PrintKVMsJSON prints the kvm details in json format.
func PrintKVMsJSON(kvms []*ufspb.KVM) {
	len := len(kvms) - 1
	for i, s := range kvms {
		s.Name = ufsUtil.RemovePrefix(s.Name)
		PrintProtoJSON(s)
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

func printRPM(rpm *ufspb.RPM, keysOnly bool) {
	if keysOnly {
		fmt.Fprintln(tw, ufsUtil.RemovePrefix(rpm.Name))
		return
	}
	var ts string
	if t, err := ptypes.Timestamp(rpm.GetUpdateTime()); err == nil {
		ts = t.Format(timeFormat)
	}
	out := fmt.Sprintf("%s\t", ufsUtil.RemovePrefix(rpm.Name))
	out += fmt.Sprintf("%s\t", rpm.GetMacAddress())
	out += fmt.Sprintf("%d\t", rpm.GetCapacityPort())
	out += fmt.Sprintf("%s\t", ts)
	fmt.Fprintln(tw, out)
}

// PrintRPMsJSON prints the rpm details in json format.
func PrintRPMsJSON(rpms []*ufspb.RPM) {
	len := len(rpms) - 1
	for i, s := range rpms {
		s.Name = ufsUtil.RemovePrefix(s.Name)
		PrintProtoJSON(s)
		if i < len {
			fmt.Print(",")
			fmt.Println()
		}
	}
}

// PrintDracFull prints the full related msg for drac
func PrintDracFull(drac *ufspb.Drac, machine *ufspb.Machine, dhcp *ufspb.DHCPConfig) {
	defer tw.Flush()
	var ts string
	if t, err := ptypes.Timestamp(drac.GetUpdateTime()); err == nil {
		ts = t.Format(timeFormat)
	}
	out := fmt.Sprintf("%s\t", ufsUtil.RemovePrefix(drac.Name))
	out += fmt.Sprintf("%s\t", drac.GetMacAddress())
	out += fmt.Sprintf("%s\t", drac.GetSwitchInterface().GetSwitch())
	out += fmt.Sprintf("%s\t", drac.GetSwitchInterface().GetPortName())
	out += fmt.Sprintf("%s\t", dhcp.GetHostname())
	out += fmt.Sprintf("%s\t", dhcp.GetIp())
	out += fmt.Sprintf("%s\t", dhcp.GetVlan())
	out += fmt.Sprintf("%s\t", drac.GetLab())
	out += fmt.Sprintf("%s\t", drac.GetRack())
	out += fmt.Sprintf("%s\t", drac.GetMachine())
	out += fmt.Sprintf("%s\t", ts)
	fmt.Fprintln(tw, out)
}

// PrintDracs prints the all dracs in table form.
func PrintDracs(dracs []*ufspb.Drac, keysOnly bool) {
	defer tw.Flush()
	for _, drac := range dracs {
		printDrac(drac, keysOnly)
	}
}

func printDrac(drac *ufspb.Drac, keysOnly bool) {
	if keysOnly {
		fmt.Fprintln(tw, ufsUtil.RemovePrefix(drac.Name))
		return
	}
	var ts string
	if t, err := ptypes.Timestamp(drac.GetUpdateTime()); err == nil {
		ts = t.Format(timeFormat)
	}
	out := fmt.Sprintf("%s\t", ufsUtil.RemovePrefix(drac.Name))
	out += fmt.Sprintf("%s\t", drac.GetDisplayName())
	out += fmt.Sprintf("%s\t", drac.GetMacAddress())
	out += fmt.Sprintf("%s\t", drac.GetSwitchInterface().GetSwitch())
	out += fmt.Sprintf("%s\t", drac.GetSwitchInterface().GetPortName())
	out += fmt.Sprintf("%s\t", drac.GetPassword())
	out += fmt.Sprintf("%s\t", drac.GetLab())
	out += fmt.Sprintf("%s\t", drac.GetRack())
	out += fmt.Sprintf("%s\t", drac.GetMachine())
	out += fmt.Sprintf("%s\t", ts)
	fmt.Fprintln(tw, out)
}

// PrintDracsJSON prints the drac details in json format.
func PrintDracsJSON(dracs []*ufspb.Drac) {
	len := len(dracs) - 1
	for i, s := range dracs {
		s.Name = ufsUtil.RemovePrefix(s.Name)
		PrintProtoJSON(s)
		if i < len {
			fmt.Print(",")
			fmt.Println()
		}
	}
}

// PrintNicFull prints the full related msg for nic
func PrintNicFull(nic *ufspb.Nic, machine *ufspb.Machine, dhcp *ufspb.DHCPConfig) {
	defer tw.Flush()
	var ts string
	if t, err := ptypes.Timestamp(nic.GetUpdateTime()); err == nil {
		ts = t.Format(timeFormat)
	}
	out := fmt.Sprintf("%s\t", ufsUtil.RemovePrefix(nic.Name))
	out += fmt.Sprintf("%s\t", nic.GetMacAddress())
	out += fmt.Sprintf("%s\t", nic.GetSwitchInterface().GetSwitch())
	out += fmt.Sprintf("%s\t", nic.GetSwitchInterface().GetPortName())
	out += fmt.Sprintf("%s\t", dhcp.GetHostname())
	out += fmt.Sprintf("%s\t", dhcp.GetIp())
	out += fmt.Sprintf("%s\t", dhcp.GetVlan())
	out += fmt.Sprintf("%s\t", nic.GetLab())
	out += fmt.Sprintf("%s\t", nic.GetRack())
	out += fmt.Sprintf("%s\t", nic.GetMachine())
	out += fmt.Sprintf("%s\t", ts)
	fmt.Fprintln(tw, out)
}

// PrintNics prints the all nics in table form.
func PrintNics(nics []*ufspb.Nic, keysOnly bool) {
	defer tw.Flush()
	for _, nic := range nics {
		printNic(nic, keysOnly)
	}
}

func printNic(nic *ufspb.Nic, keysOnly bool) {
	if keysOnly {
		fmt.Fprintln(tw, ufsUtil.RemovePrefix(nic.Name))
		return
	}
	var ts string
	if t, err := ptypes.Timestamp(nic.GetUpdateTime()); err == nil {
		ts = t.Format(timeFormat)
	}
	out := fmt.Sprintf("%s\t", ufsUtil.RemovePrefix(nic.Name))
	out += fmt.Sprintf("%s\t", nic.GetMacAddress())
	out += fmt.Sprintf("%s\t", nic.GetSwitchInterface().GetSwitch())
	out += fmt.Sprintf("%s\t", nic.GetSwitchInterface().GetPortName())
	out += fmt.Sprintf("%s\t", nic.GetLab())
	out += fmt.Sprintf("%s\t", nic.GetRack())
	out += fmt.Sprintf("%s\t", nic.GetMachine())
	out += fmt.Sprintf("%s\t", ts)
	fmt.Fprintln(tw, out)
}

// PrintNicsJSON prints the nic details in json format.
func PrintNicsJSON(nics []*ufspb.Nic) {
	len := len(nics) - 1
	for i, s := range nics {
		s.Name = ufsUtil.RemovePrefix(s.Name)
		PrintProtoJSON(s)
		if i < len {
			fmt.Print(",")
			fmt.Println()
		}
	}
}

// PrintMachineFull prints the full machine info.
func PrintMachineFull(m *ufspb.Machine, lse *ufspb.MachineLSE, rack *ufspb.Rack) {
	defer tw.Flush()
	if m.GetChromeBrowserMachine() != nil {
		printBrowserMachineFull(m, lse, rack)
	}
	if m.GetChromeosMachine() != nil {
		printOSMachineFull(m)
	}
}

func printBrowserMachineFull(m *ufspb.Machine, lse *ufspb.MachineLSE, rack *ufspb.Rack) {
	var ts string
	if t, err := ptypes.Timestamp(m.GetUpdateTime()); err == nil {
		ts = t.Format(timeFormat)
	}
	out := fmt.Sprintf("%s\t", m.GetName())
	out += fmt.Sprintf("%s\t", m.GetSerialNumber())
	out += fmt.Sprintf("%s\t", lse.GetName())
	out += fmt.Sprintf("%s\t", m.GetLocation().GetLab())
	out += fmt.Sprintf("%s\t", m.GetLocation().GetRack())
	out += fmt.Sprintf("%s\t", m.GetChromeBrowserMachine().GetChromePlatform())
	out += fmt.Sprintf("%s\t", strSlicesToStr(ufsAPI.ParseResources(m.GetChromeBrowserMachine().GetNicObjects(), "Name")))
	out += fmt.Sprintf("%s\t", m.GetChromeBrowserMachine().GetDracObject().GetName())
	out += fmt.Sprintf("%s\t", strSlicesToStr(ufsAPI.ParseResources(rack.GetChromeBrowserRack().GetKvmObjects(), "Name")))
	out += fmt.Sprintf("%s\t", strSlicesToStr(ufsAPI.ParseResources(rack.GetChromeBrowserRack().GetSwitchObjects(), "Name")))
	out += fmt.Sprintf("%s\t", m.GetChromeBrowserMachine().GetDeploymentTicket())
	out += fmt.Sprintf("%s\t", m.GetChromeBrowserMachine().GetDescription())
	out += fmt.Sprintf("%s\t", m.GetState())
	out += fmt.Sprintf("%s\t", m.GetRealm())
	out += fmt.Sprintf("%s\t", ts)
	fmt.Fprintln(tw, out)
}

func printOSMachineFull(m *ufspb.Machine) {
	var ts string
	if t, err := ptypes.Timestamp(m.GetUpdateTime()); err == nil {
		ts = t.Format(timeFormat)
	}
	out := fmt.Sprintf("%s\t", m.GetName())
	out += fmt.Sprintf("%s\t", m.GetLocation().GetLab())
	out += fmt.Sprintf("%s\t", m.GetLocation().GetRack())
	out += fmt.Sprintf("%s\t", m.GetLocation().GetBarcodeName())
	out += fmt.Sprintf("%s\t", ts)
	fmt.Fprintln(tw, out)
}

// PrintMachines prints the all machines in table form.
func PrintMachines(machines []*ufspb.Machine, keysOnly bool) {
	defer tw.Flush()
	for _, m := range machines {
		printMachine(m, keysOnly)
	}
}

func printMachine(m *ufspb.Machine, keysOnly bool) {
	m.Name = ufsUtil.RemovePrefix(m.Name)
	if keysOnly {
		fmt.Fprintln(tw, m.GetName())
		return
	}
	var ts string
	if t, err := ptypes.Timestamp(m.GetUpdateTime()); err == nil {
		ts = t.Format(timeFormat)
	}
	out := fmt.Sprintf("%s\t", m.GetName())
	out += fmt.Sprintf("%s\t", m.GetSerialNumber())
	out += fmt.Sprintf("%s\t", m.GetLocation().GetLab())
	out += fmt.Sprintf("%s\t", m.GetLocation().GetRack())
	out += fmt.Sprintf("%s\t", m.GetChromeBrowserMachine().GetChromePlatform())
	out += fmt.Sprintf("%s\t", strSlicesToStr(ufsAPI.ParseResources(m.GetChromeBrowserMachine().GetNicObjects(), "Name")))
	out += fmt.Sprintf("%s\t", m.GetChromeBrowserMachine().GetDracObject().GetName())
	out += fmt.Sprintf("%s\t", m.GetChromeBrowserMachine().GetDeploymentTicket())
	out += fmt.Sprintf("%s\t", m.GetChromeBrowserMachine().GetDescription())
	out += fmt.Sprintf("%s\t", m.GetState())
	out += fmt.Sprintf("%s\t", m.GetRealm())
	out += fmt.Sprintf("%s\t", ts)
	fmt.Fprintln(tw, out)
}

// PrintMachinesJSON prints the machine details in json format.
func PrintMachinesJSON(machines []*ufspb.Machine) {
	len := len(machines) - 1
	for i, m := range machines {
		m.Name = ufsUtil.RemovePrefix(m.Name)
		PrintProtoJSON(m)
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

func printMachineLSEPrototype(m *ufspb.MachineLSEPrototype, keysOnly bool) {
	if keysOnly {
		fmt.Fprintln(tw, ufsUtil.RemovePrefix(m.Name))
		return
	}
	var ts string
	if t, err := ptypes.Timestamp(m.GetUpdateTime()); err == nil {
		ts = t.Format(timeFormat)
	}
	m.Name = ufsUtil.RemovePrefix(m.Name)
	out := fmt.Sprintf("%s\t", m.GetName())
	out += fmt.Sprintf("%d\t", m.GetOccupiedCapacityRu())
	prs := m.GetPeripheralRequirements()
	var peripheralTypes string
	for _, pr := range prs {
		peripheralTypes += fmt.Sprintf("%s,", pr.GetPeripheralType())
	}
	out += fmt.Sprintf("%s\t", strings.TrimSuffix(peripheralTypes, ","))
	vms := m.GetVirtualRequirements()
	var virtualTypes string
	for _, vm := range vms {
		virtualTypes += fmt.Sprintf("%s,", vm.GetVirtualType())
	}
	out += fmt.Sprintf("%s\t", strings.TrimSuffix(virtualTypes, ","))
	out += fmt.Sprintf("%s\t", m.GetTags())
	out += fmt.Sprintf("%s\t", ts)
	fmt.Fprintln(tw, out)
}

// PrintMachineLSEPrototypesJSON prints the mslep details in json format.
func PrintMachineLSEPrototypesJSON(msleps []*ufspb.MachineLSEPrototype) {
	len := len(msleps) - 1
	for i, m := range msleps {
		m.Name = ufsUtil.RemovePrefix(m.Name)
		PrintProtoJSON(m)
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

func printRackLSEPrototype(m *ufspb.RackLSEPrototype, keysOnly bool) {
	if keysOnly {
		fmt.Fprintln(tw, ufsUtil.RemovePrefix(m.Name))
		return
	}
	var ts string
	if t, err := ptypes.Timestamp(m.GetUpdateTime()); err == nil {
		ts = t.Format(timeFormat)
	}
	m.Name = ufsUtil.RemovePrefix(m.Name)
	out := fmt.Sprintf("%s\t", m.GetName())
	prs := m.GetPeripheralRequirements()
	var peripheralTypes string
	for _, pr := range prs {
		peripheralTypes += fmt.Sprintf("%s,", pr.GetPeripheralType())
	}
	out += fmt.Sprintf("%s\t", strings.TrimSuffix(peripheralTypes, ","))
	out += fmt.Sprintf("%s\t", m.GetTags())
	out += fmt.Sprintf("%s\t", ts)
	fmt.Fprintln(tw, out)
}

// PrintRackLSEPrototypesJSON prints the mslep details in json format.
func PrintRackLSEPrototypesJSON(rlseps []*ufspb.RackLSEPrototype) {
	len := len(rlseps) - 1
	for i, m := range rlseps {
		m.Name = ufsUtil.RemovePrefix(m.Name)
		PrintProtoJSON(m)
		if i < len {
			fmt.Print(",")
			fmt.Println()
		}
	}
}

// PrintVlansJSON prints the vlan details in json format.
func PrintVlansJSON(vlans []*ufspb.Vlan) {
	len := len(vlans) - 1
	for i, m := range vlans {
		m.Name = ufsUtil.RemovePrefix(m.Name)
		PrintProtoJSON(m)
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

func printVlan(m *ufspb.Vlan, keysOnly bool) {
	if keysOnly {
		fmt.Fprintln(tw, ufsUtil.RemovePrefix(m.Name))
		return
	}
	var ts string
	if t, err := ptypes.Timestamp(m.GetUpdateTime()); err == nil {
		ts = t.Format(timeFormat)
	}
	m.Name = ufsUtil.RemovePrefix(m.Name)
	out := fmt.Sprintf("%s\t", m.GetName())
	out += fmt.Sprintf("%s\t", m.GetVlanAddress())
	out += fmt.Sprintf("%d\t", m.GetCapacityIp())
	out += fmt.Sprintf("%s\t", m.GetDescription())
	out += fmt.Sprintf("%s\t", m.GetState())
	out += fmt.Sprintf("%s\t", ts)
	fmt.Fprintln(tw, out)
}

// PrintChromePlatforms prints the all msleps in table form.
func PrintChromePlatforms(msleps []*ufspb.ChromePlatform, keysOnly bool) {
	defer tw.Flush()
	for _, m := range msleps {
		printChromePlatform(m, keysOnly)
	}
}

func printChromePlatform(m *ufspb.ChromePlatform, keysOnly bool) {
	if keysOnly {
		fmt.Fprintln(tw, ufsUtil.RemovePrefix(m.Name))
		return
	}
	var ts string
	if t, err := ptypes.Timestamp(m.GetUpdateTime()); err == nil {
		ts = t.Format(timeFormat)
	}
	m.Name = ufsUtil.RemovePrefix(m.Name)
	out := fmt.Sprintf("%s\t", m.GetName())
	out += fmt.Sprintf("%s\t", m.GetManufacturer())
	out += fmt.Sprintf("%s\t", m.GetDescription())
	out += fmt.Sprintf("%s\t", ts)
	fmt.Fprintln(tw, out)
}

// PrintChromePlatformsJSON prints the mslep details in json format.
func PrintChromePlatformsJSON(msleps []*ufspb.ChromePlatform) {
	len := len(msleps) - 1
	for i, m := range msleps {
		m.Name = ufsUtil.RemovePrefix(m.Name)
		PrintProtoJSON(m)
		if i < len {
			fmt.Print(",")
			fmt.Println()
		}
	}
}

// PrintMachineLSEsJSON prints the machinelse details in json format.
func PrintMachineLSEsJSON(machinelses []*ufspb.MachineLSE) {
	len := len(machinelses) - 1
	for i, m := range machinelses {
		m.Name = ufsUtil.RemovePrefix(m.Name)
		PrintProtoJSON(m)
		if i < len {
			fmt.Print(",")
			fmt.Println()
		}
	}
}

// PrintMachineLSEFull prints the full info for a host
func PrintMachineLSEFull(lse *ufspb.MachineLSE, machine *ufspb.Machine, dhcp *ufspb.DHCPConfig) {
	defer tw.Flush()
	var ts string
	if t, err := ptypes.Timestamp(lse.GetUpdateTime()); err == nil {
		ts = t.Format(timeFormat)
	}
	out := fmt.Sprintf("%s\t", lse.GetName())
	out += fmt.Sprintf("%s\t", lse.GetChromeBrowserMachineLse().GetOsVersion().GetValue())
	out += fmt.Sprintf("%s\t", machine.GetName())
	out += fmt.Sprintf("%s\t", lse.GetLab())
	out += fmt.Sprintf("%s\t", lse.GetRack())
	out += fmt.Sprintf("%s\t", lse.GetNic())
	out += fmt.Sprintf("%s\t", dhcp.GetIp())
	out += fmt.Sprintf("%s\t", dhcp.GetVlan())
	out += fmt.Sprintf("%s\t", lse.GetState())
	out += fmt.Sprintf("%d\t", lse.GetChromeBrowserMachineLse().GetVmCapacity())
	out += fmt.Sprintf("%s\t", strSlicesToStr(ufsAPI.ParseResources(lse.GetChromeBrowserMachineLse().GetVms(), "Name")))
	out += fmt.Sprintf("%s\t", ts)
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

func printMachineLSE(m *ufspb.MachineLSE, keysOnly bool) {
	if keysOnly {
		fmt.Fprintln(tw, ufsUtil.RemovePrefix(m.Name))
		return
	}
	var ts string
	if t, err := ptypes.Timestamp(m.GetUpdateTime()); err == nil {
		ts = t.Format(timeFormat)
	}
	out := fmt.Sprintf("%s\t", m.GetName())
	out += fmt.Sprintf("%s\t", m.GetChromeBrowserMachineLse().GetOsVersion().GetValue())
	out += fmt.Sprintf("%s\t", m.GetLab())
	out += fmt.Sprintf("%s\t", m.GetRack())
	out += fmt.Sprintf("%s\t", m.GetNic())
	out += fmt.Sprintf("%s\t", m.GetState())
	out += fmt.Sprintf("%d\t", m.GetChromeBrowserMachineLse().GetVmCapacity())
	out += fmt.Sprintf("%s\t", strSlicesToStr(ufsAPI.ParseResources(m.GetChromeBrowserMachineLse().GetVms(), "Name")))
	out += fmt.Sprintf("%s\t", ts)
	fmt.Fprintln(tw, out)
}

// PrintFreeVMs prints the all free slots in table form.
func PrintFreeVMs(ctx context.Context, ic ufsAPI.FleetClient, hosts []*ufspb.MachineLSE) {
	defer tw.Flush()
	PrintTitle([]string{"Host", "Os Version", "Vlan", "Lab", "Free slots"})
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
	out += fmt.Sprintf("%s\t", res.GetVlan())
	out += fmt.Sprintf("%s\t", host.GetLab())
	out += fmt.Sprintf("%d\t", host.GetChromeBrowserMachineLse().GetVmCapacity())
	fmt.Fprintln(tw, out)
}

// PrintVMFull prints the full info for vm
func PrintVMFull(vm *ufspb.VM, dhcp *ufspb.DHCPConfig, s *ufspb.StateRecord) {
	defer tw.Flush()
	var ts string
	if t, err := ptypes.Timestamp(vm.GetUpdateTime()); err == nil {
		ts = t.Format(timeFormat)
	}
	out := fmt.Sprintf("%s\t", vm.GetName())
	out += fmt.Sprintf("%s\t", vm.GetOsVersion().GetValue())
	out += fmt.Sprintf("%s\t", vm.GetOsVersion().GetDescription())
	out += fmt.Sprintf("%s\t", vm.GetMacAddress())
	out += fmt.Sprintf("%s\t", vm.GetLab())
	out += fmt.Sprintf("%s\t", vm.GetMachineLseId())
	out += fmt.Sprintf("%s\t", dhcp.GetVlan())
	out += fmt.Sprintf("%s\t", dhcp.GetIp())
	out += fmt.Sprintf("%s\t", s.GetState())
	out += fmt.Sprintf("%s\t", ts)
	fmt.Fprintln(tw, out)
}

// PrintVMs prints the all vms in table form.
func PrintVMs(vms []*ufspb.VM, keysOnly bool) {
	defer tw.Flush()
	for _, vm := range vms {
		printVM(vm, keysOnly)
	}
}

func printVM(vm *ufspb.VM, keysOnly bool) {
	if keysOnly {
		fmt.Fprintln(tw, ufsUtil.RemovePrefix(vm.Name))
		return
	}
	var ts string
	if t, err := ptypes.Timestamp(vm.GetUpdateTime()); err == nil {
		ts = t.Format(timeFormat)
	}
	out := fmt.Sprintf("%s\t", vm.GetName())
	out += fmt.Sprintf("%s\t", vm.GetOsVersion().GetValue())
	out += fmt.Sprintf("%s\t", vm.GetOsVersion().GetDescription())
	out += fmt.Sprintf("%s\t", vm.GetMacAddress())
	out += fmt.Sprintf("%s\t", vm.GetLab())
	out += fmt.Sprintf("%s\t", vm.GetMachineLseId())
	out += fmt.Sprintf("%s\t", vm.GetVlan())
	out += fmt.Sprintf("%s\t", vm.GetState())
	out += fmt.Sprintf("%s\t", ts)
	fmt.Fprintln(tw, out)
}

// PrintVMsJSON prints the vm details in json format.
func PrintVMsJSON(vms []*ufspb.VM) {
	len := len(vms) - 1
	for i, m := range vms {
		m.Name = ufsUtil.RemovePrefix(m.Name)
		PrintProtoJSON(m)
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

func printRack(m *ufspb.Rack, keysOnly bool) {
	m.Name = ufsUtil.RemovePrefix(m.Name)
	if keysOnly {
		fmt.Fprintln(tw, m.GetName())
		return
	}
	var ts string
	if t, err := ptypes.Timestamp(m.GetUpdateTime()); err == nil {
		ts = t.Format(timeFormat)
	}
	out := fmt.Sprintf("%s\t", m.GetName())
	out += fmt.Sprintf("%s\t", m.GetLocation().GetLab())
	out += fmt.Sprintf("%s\t", strSlicesToStr(ufsAPI.ParseResources(m.GetChromeBrowserRack().GetKvmObjects(), "Name")))
	out += fmt.Sprintf("%s\t", strSlicesToStr(ufsAPI.ParseResources(m.GetChromeBrowserRack().GetSwitchObjects(), "Name")))
	out += fmt.Sprintf("%s\t", strSlicesToStr(ufsAPI.ParseResources(m.GetChromeBrowserRack().GetRpmObjects(), "Name")))
	out += fmt.Sprintf("%d\t", m.GetCapacityRu())
	out += fmt.Sprintf("%s\t", m.GetState())
	out += fmt.Sprintf("%s\t", m.GetRealm())
	out += fmt.Sprintf("%s\t", ts)
	fmt.Fprintln(tw, out)
}

// PrintRacksJSON prints the rack details in json format.
func PrintRacksJSON(racks []*ufspb.Rack) {
	len := len(racks) - 1
	for i, m := range racks {
		m.Name = ufsUtil.RemovePrefix(m.Name)
		PrintProtoJSON(m)
		if i < len {
			fmt.Print(",")
			fmt.Println()
		}
	}
}

func strSlicesToStr(slices []string) string {
	return strings.Join(slices, ",")
}
