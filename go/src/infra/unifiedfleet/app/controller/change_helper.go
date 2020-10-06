// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	ufspb "infra/unifiedfleet/api/v1/proto"
	"infra/unifiedfleet/app/util"
)

// LogAddRackChanges logs the changes for adding rack
func (hc *HistoryClient) LogAddRackChanges(rack *ufspb.Rack, switches []*ufspb.Switch, kvms []*ufspb.KVM, rpms []*ufspb.RPM) {
	hc.LogRackChanges(nil, rack)
	for _, m := range switches {
		hc.LogSwitchChanges(nil, m)
	}
	for _, m := range kvms {
		hc.LogKVMChanges(nil, m)
	}
	for _, m := range rpms {
		hc.LogRPMChanges(nil, m)
	}
}

// LogAddMachineChanges logs the changes for adding machine
func (hc *HistoryClient) LogAddMachineChanges(machine *ufspb.Machine, nics []*ufspb.Nic, drac *ufspb.Drac) {
	hc.LogMachineChanges(nil, machine)
	for _, m := range nics {
		hc.LogNicChanges(nil, m)
	}
	hc.LogDracChanges(nil, drac)
}

// LogMachineLocationChanges logs the changes for changing locations of a machine
func (hc *HistoryClient) LogMachineLocationChanges(lses []*ufspb.MachineLSE, nics []*ufspb.Nic, dracs []*ufspb.Drac, vms []*ufspb.VM, indexMap, oldIndexMap map[string]string) {
	for k, v := range indexMap {
		switch k {
		case "zone":
			old := oldIndexMap["zone"]
			for _, lse := range lses {
				hc.changes = append(hc.changes, logCommon(lse.GetName(), "machine_lse.zone", approxZone(old), approxZone(v))...)
			}
			for _, nic := range nics {
				hc.changes = append(hc.changes, logCommon(nic.GetName(), "nic.zone", approxZone(old), approxZone(v))...)
			}
			for _, drac := range dracs {
				hc.changes = append(hc.changes, logCommon(drac.GetName(), "drac.zone", approxZone(old), approxZone(v))...)
			}
			for _, vm := range vms {
				hc.changes = append(hc.changes, logCommon(vm.GetName(), "vm.zone", approxZone(old), approxZone(v))...)
			}
		case "rack":
			old := oldIndexMap["rack"]
			for _, lse := range lses {
				hc.changes = append(hc.changes, logCommon(lse.GetName(), "machine_lse.rack", old, v)...)
			}
			for _, nic := range nics {
				hc.changes = append(hc.changes, logCommon(nic.GetName(), "nic.rack", old, v)...)
			}
			for _, drac := range dracs {
				hc.changes = append(hc.changes, logCommon(drac.GetName(), "drac.rack", old, v)...)
			}
			for _, vm := range vms {
				hc.changes = append(hc.changes, logCommon(vm.GetName(), "vm.rack", old, v)...)
			}
		case "machine":
			old := oldIndexMap["machine"]
			for _, lse := range lses {
				hc.changes = append(hc.changes, logCommon(lse.GetName(), "machine_lse.machine", approxZone(old), approxZone(v))...)
			}
			for _, nic := range nics {
				hc.changes = append(hc.changes, logCommon(nic.GetName(), "nic.machine", approxZone(old), approxZone(v))...)
			}
			for _, drac := range dracs {
				hc.changes = append(hc.changes, logCommon(drac.GetName(), "drac.machine", approxZone(old), approxZone(v))...)
			}
		}
	}
	for k, v := range indexMap {
		if v != oldIndexMap[k] {
			for _, lse := range lses {
				hc.logMsgEntity(util.AddPrefix(util.HostCollection, lse.GetName()), false, lse)
			}
			for _, nic := range nics {
				hc.logMsgEntity(util.AddPrefix(util.NicCollection, nic.GetName()), false, nic)
			}
			for _, drac := range dracs {
				hc.logMsgEntity(util.AddPrefix(util.DracCollection, drac.GetName()), false, drac)
			}
			for _, vm := range vms {
				hc.logMsgEntity(util.AddPrefix(util.VMCollection, vm.GetName()), false, vm)
			}
			break
		}
	}
}

// LogRackLocationChanges logs the changes for changing locations of a rack
func (hc *HistoryClient) LogRackLocationChanges(kvms []*ufspb.KVM, switches []*ufspb.Switch, rpms []*ufspb.RPM, indexMap, oldIndexMap map[string]string) {
	for k, v := range indexMap {
		switch k {
		case "zone":
			old := oldIndexMap["zone"]
			for _, kvm := range kvms {
				hc.changes = append(hc.changes, logCommon(kvm.GetName(), "kvm.zone", approxZone(old), approxZone(v))...)
			}
			for _, sw := range switches {
				hc.changes = append(hc.changes, logCommon(sw.GetName(), "switch.zone", approxZone(old), approxZone(v))...)
			}
			for _, rpm := range rpms {
				hc.changes = append(hc.changes, logCommon(rpm.GetName(), "rpm.zone", approxZone(old), approxZone(v))...)
			}
		}
	}
	for k, v := range indexMap {
		if v != oldIndexMap[k] {
			for _, kvm := range kvms {
				hc.logMsgEntity(util.AddPrefix(util.KVMCollection, kvm.GetName()), false, kvm)
			}
			for _, sw := range switches {
				hc.logMsgEntity(util.AddPrefix(util.SwitchCollection, sw.GetName()), false, sw)
			}
			for _, rpm := range rpms {
				hc.logMsgEntity(util.AddPrefix(util.RPMCollection, rpm.GetName()), false, rpm)
			}
			break
		}
	}
}
