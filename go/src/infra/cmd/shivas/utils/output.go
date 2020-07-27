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
	SwitchTitle = []string{"Switch Name", "CapacityPort", "UpdateTime"}
	KvmTitle    = []string{"KVM Name", "MAC Address", "ChromePlatform",
		"CapacityPort", "UpdateTime"}
	RpmTitle = []string{"RPM Name", "MAC Address", "CapacityPort",
		"UpdateTime"}
	DracTitle = []string{"Drac Name", "Display name", "MAC Address", "Switch",
		"Switch Port", "Password", "UpdateTime"}
	NicTitle = []string{"Nic Name", "MAC Address", "Switch", "Switch Port",
		"UpdateTime"}
	MachineTitle = []string{"Machine Name", "Lab", "Rack", "Aisle", "Row",
		"Rack Number", "Shelf", "Position", "DisplayName", "ChromePlatform",
		"Nics", "KVM", "KVM Port", "RPM", "RPM Port", "Switch", "Switch Port",
		"Drac", "DeploymentTicket", "Description", "Realm", "UpdateTime"}
	MachinelseprototypeTitle = []string{"Machine Prototype Name",
		"Occupied Capacity", "PeripheralTypes", "VirtualTypes",
		"UpdateTime"}
	RacklseprototypeTitle = []string{"Rack Prototype Name", "PeripheralTypes",
		"UpdateTime"}
	ChromePlatformTitle = []string{"Chrome Platform Name", "Manufacturer",
		"Description", "UpdateTime"}
	vmTitle = []string{"VM Name", "OS Version", "OS Desc", "MAC Address",
		"VM Hostname"}
	RackTitle = []string{"Rack Name", "Lab", "Switches", "KVMs", "RPMs",
		"Capacity", "Realm", "UpdateTime"}
)

// TimeFormat for all timestamps handled by shivas
var timeFormat = "2006-01-02-15:04:05"

// The tab writer which defines the write format
var tw = tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)

// The io writer for json output
var bw = bufio.NewWriter(os.Stdout)

type printAll func(context.Context, ufsAPI.FleetClient, bool, int32, string, string) (string, error)

// PrintListJSONFormat prints the list output in JSON format
func PrintListJSONFormat(ctx context.Context, ic ufsAPI.FleetClient, f printAll, json bool, pageSize int32, filter string) error {
	var pageToken string
	fmt.Print("[")
	if pageSize == 0 {
		for {
			token, err := f(ctx, ic, json, ufsUtil.MaxPageSize, pageToken, filter)
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
			token, err := f(ctx, ic, json, size, pageToken, filter)
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
func PrintListTableFormat(ctx context.Context, ic ufsAPI.FleetClient, f printAll, json bool, pageSize int32, filter string, title []string) error {
	printTitle(title)
	var pageToken string
	if pageSize == 0 {
		for {
			token, err := f(ctx, ic, json, ufsUtil.MaxPageSize, pageToken, filter)
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
			token, err := f(ctx, ic, json, size, pageToken, filter)
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

// printTitle prints the title fields in table form.
func printTitle(title []string) {
	for _, s := range title {
		fmt.Fprint(tw, fmt.Sprintf("%s\t", s))
	}
	fmt.Fprintln(tw)
}

// PrintSwitches prints the all switches in table form.
func PrintSwitches(switches []*ufspb.Switch) {
	defer tw.Flush()
	for _, s := range switches {
		printSwitch(s)
	}
}

func printSwitch(s *ufspb.Switch) {
	var ts string
	if t, err := ptypes.Timestamp(s.GetUpdateTime()); err == nil {
		ts = t.Format(timeFormat)
	}
	s.Name = ufsUtil.RemovePrefix(s.Name)
	out := fmt.Sprintf("%s\t%d\t%s\t", s.GetName(), s.GetCapacityPort(), ts)
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

// PrintKVMs prints the all kvms in table form.
func PrintKVMs(kvms []*ufspb.KVM) {
	defer tw.Flush()
	for _, kvm := range kvms {
		printKVM(kvm)
	}
}

func printKVM(kvm *ufspb.KVM) {
	var ts string
	if t, err := ptypes.Timestamp(kvm.GetUpdateTime()); err == nil {
		ts = t.Format(timeFormat)
	}
	out := fmt.Sprintf("%s\t", ufsUtil.RemovePrefix(kvm.Name))
	out += fmt.Sprintf("%s\t", kvm.GetMacAddress())
	out += fmt.Sprintf("%s\t", kvm.GetChromePlatform())
	out += fmt.Sprintf("%d\t", kvm.GetCapacityPort())
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
func PrintRPMs(rpms []*ufspb.RPM) {
	defer tw.Flush()
	for _, rpm := range rpms {
		printRPM(rpm)
	}
}

func printRPM(rpm *ufspb.RPM) {
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

// PrintDracs prints the all dracs in table form.
func PrintDracs(dracs []*ufspb.Drac) {
	defer tw.Flush()
	for _, drac := range dracs {
		printDrac(drac)
	}
}

func printDrac(drac *ufspb.Drac) {
	var ts string
	if t, err := ptypes.Timestamp(drac.GetUpdateTime()); err == nil {
		ts = t.Format(timeFormat)
	}
	out := fmt.Sprintf("%s\t", ufsUtil.RemovePrefix(drac.Name))
	out += fmt.Sprintf("%s\t", drac.GetDisplayName())
	out += fmt.Sprintf("%s\t", drac.GetMacAddress())
	out += fmt.Sprintf("%s\t", drac.GetSwitchInterface().GetSwitch())
	out += fmt.Sprintf("%d\t", drac.GetSwitchInterface().GetPort())
	out += fmt.Sprintf("%s\t", drac.GetPassword())
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

// PrintNics prints the all nics in table form.
func PrintNics(nics []*ufspb.Nic) {
	defer tw.Flush()
	for _, nic := range nics {
		printNic(nic)
	}
}

func printNic(nic *ufspb.Nic) {
	var ts string
	if t, err := ptypes.Timestamp(nic.GetUpdateTime()); err == nil {
		ts = t.Format(timeFormat)
	}
	out := fmt.Sprintf("%s\t", ufsUtil.RemovePrefix(nic.Name))
	out += fmt.Sprintf("%s\t", nic.GetMacAddress())
	out += fmt.Sprintf("%s\t", nic.GetSwitchInterface().GetSwitch())
	out += fmt.Sprintf("%d\t", nic.GetSwitchInterface().GetPort())
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

// PrintMachines prints the all machines in table form.
func PrintMachines(machines []*ufspb.Machine) {
	defer tw.Flush()
	for _, m := range machines {
		printMachine(m)
	}
}

func printMachine(m *ufspb.Machine) {
	var ts string
	if t, err := ptypes.Timestamp(m.GetUpdateTime()); err == nil {
		ts = t.Format(timeFormat)
	}
	m.Name = ufsUtil.RemovePrefix(m.Name)
	out := fmt.Sprintf("%s\t", m.GetName())
	out += fmt.Sprintf("%s\t", m.GetLocation().GetLab())
	out += fmt.Sprintf("%s\t", m.GetLocation().GetRack())
	out += fmt.Sprintf("%s\t", m.GetLocation().GetAisle())
	out += fmt.Sprintf("%s\t", m.GetLocation().GetRow())
	out += fmt.Sprintf("%s\t", m.GetLocation().GetRackNumber())
	out += fmt.Sprintf("%s\t", m.GetLocation().GetShelf())
	out += fmt.Sprintf("%s\t", m.GetLocation().GetPosition())
	out += fmt.Sprintf("%s\t", m.GetChromeBrowserMachine().GetDisplayName())
	out += fmt.Sprintf("%s\t", m.GetChromeBrowserMachine().GetChromePlatform())
	out += fmt.Sprintf("%s\t", m.GetChromeBrowserMachine().GetNics())
	out += fmt.Sprintf("%s\t", m.GetChromeBrowserMachine().GetKvmInterface().GetKvm())
	out += fmt.Sprintf("%d\t", m.GetChromeBrowserMachine().GetKvmInterface().GetPort())
	out += fmt.Sprintf("%s\t", m.GetChromeBrowserMachine().GetRpmInterface().GetRpm())
	out += fmt.Sprintf("%d\t", m.GetChromeBrowserMachine().GetRpmInterface().GetPort())
	out += fmt.Sprintf("%s\t", m.GetChromeBrowserMachine().GetNetworkDeviceInterface().GetSwitch())
	out += fmt.Sprintf("%d\t", m.GetChromeBrowserMachine().GetNetworkDeviceInterface().GetPort())
	out += fmt.Sprintf("%s\t", m.GetChromeBrowserMachine().GetDrac())
	out += fmt.Sprintf("%s\t", m.GetChromeBrowserMachine().GetDeploymentTicket())
	out += fmt.Sprintf("%s\t", m.GetChromeBrowserMachine().GetDescription())
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
func PrintMachineLSEPrototypes(msleps []*ufspb.MachineLSEPrototype) {
	defer tw.Flush()
	for _, m := range msleps {
		printMachineLSEPrototype(m)
	}
}

func printMachineLSEPrototype(m *ufspb.MachineLSEPrototype) {
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
func PrintRackLSEPrototypes(msleps []*ufspb.RackLSEPrototype) {
	defer tw.Flush()
	for _, m := range msleps {
		printRackLSEPrototype(m)
	}
}

func printRackLSEPrototype(m *ufspb.RackLSEPrototype) {
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

// PrintChromePlatforms prints the all msleps in table form.
func PrintChromePlatforms(msleps []*ufspb.ChromePlatform) {
	defer tw.Flush()
	for _, m := range msleps {
		printChromePlatform(m)
	}
}

func printChromePlatform(m *ufspb.ChromePlatform) {
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

// PrintVMs prints the all vms in table form.
func PrintVMs(vms []*ufspb.VM) {
	defer tw.Flush()
	printTitle(vmTitle)
	for _, vm := range vms {
		printVM(vm)
	}
}

func printVM(vm *ufspb.VM) {
	out := fmt.Sprintf("%s\t", vm.Name)
	out += fmt.Sprintf("%s\t", vm.GetOsVersion().GetValue())
	out += fmt.Sprintf("%s\t", vm.GetOsVersion().GetDescription())
	out += fmt.Sprintf("%s\t", vm.GetMacAddress())
	out += fmt.Sprintf("%s\t", vm.GetHostname())
	fmt.Fprintln(tw, out)
}

// PrintVMsJSON prints the vm details in json format.
func PrintVMsJSON(vms []*ufspb.VM) {
	fmt.Print("[")
	len := len(vms) - 1
	for i, s := range vms {
		s.Name = ufsUtil.RemovePrefix(s.Name)
		PrintProtoJSON(s)
		if i < len {
			fmt.Print(",")
			fmt.Println()
		}
	}
	fmt.Print("]")
	fmt.Println()
}

// PrintRacks prints the all racks in table form.
func PrintRacks(racks []*ufspb.Rack) {
	defer tw.Flush()
	for _, m := range racks {
		printRack(m)
	}
}

func printRack(m *ufspb.Rack) {
	var ts string
	if t, err := ptypes.Timestamp(m.GetUpdateTime()); err == nil {
		ts = t.Format(timeFormat)
	}
	m.Name = ufsUtil.RemovePrefix(m.Name)
	out := fmt.Sprintf("%s\t", m.GetName())
	out += fmt.Sprintf("%s\t", m.GetLocation().GetLab())
	out += fmt.Sprintf("%s\t", m.GetChromeBrowserRack().GetSwitches())
	out += fmt.Sprintf("%s\t", m.GetChromeBrowserRack().GetKvms())
	out += fmt.Sprintf("%s\t", m.GetChromeBrowserRack().GetRpms())
	out += fmt.Sprintf("%d\t", m.GetCapacityRu())
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
