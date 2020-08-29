// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package utils

import (
	"encoding/csv"
	"os"

	"github.com/golang/protobuf/proto"

	ufspb "infra/unifiedfleet/api/v1/proto"
)

// CSVWriter refers to a customized csv writer
type CSVWriter struct {
	*csv.Writer
}

// NewCSVWriter creates a new csv writer
func NewCSVWriter() *CSVWriter {
	w := &CSVWriter{csv.NewWriter(os.Stdout)}
	w.Comma = '\t'
	return w
}

// PrintTSVDracFull prints the full related msg for drac in tsv mode
func PrintTSVDracFull(drac *ufspb.Drac, dhcp *ufspb.DHCPConfig) {
	csw := NewCSVWriter()
	defer csw.Flush()
	csw.Write(dracFullOutputStrs(drac, dhcp))
}

// PrintTSVHostFull prints the full related msg for host in tsv mode
func PrintTSVHostFull(lse *ufspb.MachineLSE, dhcp *ufspb.DHCPConfig) {
	csw := NewCSVWriter()
	defer csw.Flush()
	csw.Write(machineLSEFullOutputStrs(lse, dhcp))
}

// PrintTSVKVMFull prints the full related msg for kvm in tsv mode
func PrintTSVKVMFull(kvm *ufspb.KVM, dhcp *ufspb.DHCPConfig) {
	csw := NewCSVWriter()
	defer csw.Flush()
	csw.Write(kvmFullOutputStrs(kvm, dhcp))
}

// PrintTSVVmFull prints the full related msg for vm in tsv mode
func PrintTSVVmFull(vm *ufspb.VM, dhcp *ufspb.DHCPConfig) {
	csw := NewCSVWriter()
	defer csw.Flush()
	csw.Write(vmFullOutputStrs(vm, dhcp))
}

// printTSVs prints tsv format of entities
func printTSVs(res []proto.Message, keysOnly bool, outputFunc func(proto.Message) []string) {
	csw := NewCSVWriter()
	defer csw.Flush()
	for _, m := range res {
		outputs := outputFunc(m)
		if keysOnly {
			csw.Write([]string{outputs[0]})
			continue
		}
		csw.Write(outputs)
	}
}

// PrintTSVMachines prints the tsv format of machines
func PrintTSVMachines(msgs []proto.Message, keysOnly bool) {
	printTSVs(msgs, keysOnly, machineOutputStrs)
}

// PrintTSVDracs prints the tsv format of dracs
func PrintTSVDracs(dracs []*ufspb.Drac, keysOnly bool) {
	msgs := make([]proto.Message, len(dracs))
	for i, d := range dracs {
		msgs[i] = d
	}
	printTSVs(msgs, keysOnly, dracOutputStrs)
}

// PrintTSVNics prints the tsv format of nics
func PrintTSVNics(msgs []proto.Message, keysOnly bool) {
	printTSVs(msgs, keysOnly, nicOutputStrs)
}

// PrintTSVRacks prints the tsv format of racks
func PrintTSVRacks(msgs []proto.Message, keysOnly bool) {
	printTSVs(msgs, keysOnly, rackOutputStrs)
}

// PrintTSVSwitches prints the tsv format of switches
func PrintTSVSwitches(msgs []proto.Message, keysOnly bool) {
	printTSVs(msgs, keysOnly, switchOutputStrs)
}

// PrintTSVKVMs prints the tsv format of kvms
func PrintTSVKVMs(res []proto.Message, keysOnly bool) {
	printTSVs(res, keysOnly, kvmOutputStrs)
}

// PrintTSVRPMs prints the tsv format of rpms
func PrintTSVRPMs(msgs []proto.Message, keysOnly bool) {
	printTSVs(msgs, keysOnly, rpmOutputStrs)
}

// PrintTSVMachineLSEs prints the tsv format of machine lses
func PrintTSVMachineLSEs(msgs []proto.Message, keysOnly bool) {
	printTSVs(msgs, keysOnly, machineLSEOutputStrs)
}

// PrintTSVVMs prints the tsv format of vms
func PrintTSVVMs(res []*ufspb.VM, keysOnly bool) {
	msgs := make([]proto.Message, len(res))
	for i, d := range res {
		msgs[i] = d
	}
	printTSVs(msgs, keysOnly, vmOutputStrs)
}

// PrintTSVVlans prints the tsv format of vlans
func PrintTSVVlans(msgs []proto.Message, keysOnly bool) {
	printTSVs(msgs, keysOnly, vlanOutputStrs)
}

// PrintTSVRackLSEPrototypes prints the tsv format of rack lse prototypes
func PrintTSVRackLSEPrototypes(msgs []proto.Message, keysOnly bool) {
	printTSVs(msgs, keysOnly, rackLSEPrototypeOutputStrs)
}

// PrintTSVMachineLSEPrototypes prints the tsv format of machine lse prototypes
func PrintTSVMachineLSEPrototypes(msgs []proto.Message, keysOnly bool) {
	printTSVs(msgs, keysOnly, machineLSEPrototypeOutputStrs)
}

// PrintTSVPlatforms prints the tsv format of chrome platforms
func PrintTSVPlatforms(msgs []proto.Message, keysOnly bool) {
	printTSVs(msgs, keysOnly, platformOutputStrs)
}
