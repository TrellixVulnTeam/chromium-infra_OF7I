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

var (
	switchTitle  = []string{"Switch Name", "CapacityPort", "UpdateTime"}
	machineTitle = []string{"Machine Name", "Lab", "Rack", "Aisle", "Row",
		"Rack Number", "Shelf", "Position", "DisplayName", "ChromePlatform",
		"Nics", "KVM", "KVM Port", "RPM", "RPM Port", "Switch", "Switch Port",
		"Drac", "DeploymentTicket", "Description", "Realm", "UpdateTime"}
	machinelseprototypeTitle = []string{"Machine Prototype Name",
		"Occupied Capacity", "PeripheralTypes", "VirtualTypes",
		"UpdateTime"}
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
			pageToken, err := f(ctx, ic, json, ufsUtil.MaxPageSize, pageToken, filter)
			if err != nil {
				return err
			}
			if pageToken == "" {
				break
			}
			fmt.Print(",")
		}
	} else {
		for i := int32(0); i < pageSize; i = i + ufsUtil.MaxPageSize {
			var size int32
			if pageSize-i < ufsUtil.MaxPageSize {
				size = pageSize % ufsUtil.MaxPageSize
			} else {
				size = ufsUtil.MaxPageSize
			}
			pageToken, err := f(ctx, ic, json, size, pageToken, filter)
			if err != nil {
				return err
			}
			if pageToken == "" {
				break
			} else if i+ufsUtil.MaxPageSize < pageSize {
				fmt.Print(",")
			}
		}
	}
	fmt.Println("]")
	return nil
}

// PrintListTableFormat prints list output in Table format
func PrintListTableFormat(ctx context.Context, ic ufsAPI.FleetClient, f printAll, json bool, pageSize int32, filter string) error {
	var pageToken string
	if pageSize == 0 {
		for {
			pageToken, err := f(ctx, ic, json, ufsUtil.MaxPageSize, pageToken, filter)
			if err != nil {
				return err
			}
			if pageToken == "" {
				break
			}
		}
	} else {
		for i := int32(0); i < pageSize; i = i + ufsUtil.MaxPageSize {
			var size int32
			if pageSize-i < ufsUtil.MaxPageSize {
				size = pageSize % ufsUtil.MaxPageSize
			} else {
				size = ufsUtil.MaxPageSize
			}
			pageToken, err := f(ctx, ic, json, size, pageToken, filter)
			if err != nil {
				return err
			}
			if pageToken == "" {
				break
			}
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
	printTitle(switchTitle)
	for _, s := range switches {
		printSwitch(s)
	}
}

func printSwitch(s *ufspb.Switch) {
	var ts string
	if t, err := ptypes.Timestamp(s.GetUpdateTime()); err == nil {
		ts = t.Format(timeFormat)
	}
	//s.Name = ufsUtil.RemovePrefix(s.Name)
	out := fmt.Sprintf("%s\t%d\t%s\t", s.GetName(), s.GetCapacityPort(), ts)
	fmt.Fprintln(tw, out)
}

// PrintSwitchesJSON prints the switch details in json format.
func PrintSwitchesJSON(switches []*ufspb.Switch) {
	for _, s := range switches {
		//s.Name = ufsUtil.RemovePrefix(s.Name)
		PrintProtoJSON(s)
	}
}

// PrintMachines prints the all machines in table form.
func PrintMachines(machines []*ufspb.Machine) {
	defer tw.Flush()
	printTitle(machineTitle)
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
	for _, m := range machines {
		m.Name = ufsUtil.RemovePrefix(m.Name)
		PrintProtoJSON(m)
	}
}

// PrintMachineLSEPrototypes prints the all msleps in table form.
func PrintMachineLSEPrototypes(msleps []*ufspb.MachineLSEPrototype) {
	defer tw.Flush()
	printTitle(machinelseprototypeTitle)
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
