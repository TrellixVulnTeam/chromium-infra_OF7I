// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package utils

import (
	"bufio"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	fleet "infra/unifiedfleet/api/v1/proto"
	UfleetUtil "infra/unifiedfleet/app/util"
)

var (
	switchTitle  = []string{"Switch Name", "CapacityPort", "UpdateTime"}
	machineTitle = []string{"Machine Name", "Lab", "Rack", "Aisle", "Row",
		"Rack Number", "Shelf", "Position", "DisplayName", "ChromePlatform",
		"Nics", "KVM", "KVM Port", "RPM", "RPM Port", "Switch", "Switch Port",
		"Drac", "DeploymentTicket", "Description", "Realm", "UpdateTime"}
)

// TimeFormat for all timestamps handled by shivas
var timeFormat = "2006-01-02-15:04:05"

// The tab writer which defines the write format
var tw = tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)

// The io writer for json output
var bw = bufio.NewWriter(os.Stdout)

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
	fmt.Println()
}

// printTitle prints the title fields in table form.
func printTitle(title []string) {
	for _, s := range title {
		fmt.Fprint(tw, fmt.Sprintf("%s\t", s))
	}
	fmt.Fprintln(tw)
}

// PrintSwitches prints the all switches in table form.
func PrintSwitches(switches []*fleet.Switch) {
	defer tw.Flush()
	printTitle(switchTitle)
	for _, s := range switches {
		printSwitch(s)
	}
}

func printSwitch(s *fleet.Switch) {
	var ts string
	if t, err := ptypes.Timestamp(s.GetUpdateTime()); err == nil {
		ts = t.Format(timeFormat)
	}
	//s.Name = UfleetUtil.RemovePrefix(s.Name)
	out := fmt.Sprintf("%s\t%d\t%s\t", s.GetName(), s.GetCapacityPort(), ts)
	fmt.Fprintln(tw, out)
}

// PrintSwitchesJSON prints the switch details in json format.
func PrintSwitchesJSON(switches []*fleet.Switch) {
	for _, s := range switches {
		//s.Name = UfleetUtil.RemovePrefix(s.Name)
		PrintProtoJSON(s)
	}
}

// PrintMachines prints the all machines in table form.
func PrintMachines(machines []*fleet.Machine) {
	defer tw.Flush()
	printTitle(machineTitle)
	for _, m := range machines {
		printMachine(m)
	}
}

func printMachine(m *fleet.Machine) {
	var ts string
	if t, err := ptypes.Timestamp(m.GetUpdateTime()); err == nil {
		ts = t.Format(timeFormat)
	}
	m.Name = UfleetUtil.RemovePrefix(m.Name)
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
func PrintMachinesJSON(machines []*fleet.Machine) {
	for _, m := range machines {
		m.Name = UfleetUtil.RemovePrefix(m.Name)
		PrintProtoJSON(m)
	}
}
