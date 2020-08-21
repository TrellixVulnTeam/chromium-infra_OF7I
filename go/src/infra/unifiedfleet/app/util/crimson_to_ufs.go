// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package util

import (
	"fmt"
	"strings"

	crimsoncommon "go.chromium.org/luci/machine-db/api/common/v1"
	crimsonconfig "go.chromium.org/luci/machine-db/api/config/v1"
	"go.chromium.org/luci/machine-db/api/crimson/v1"

	ufspb "infra/unifiedfleet/api/v1/proto"
)

// ToChromeMachines converts crimson machines to UFS format.
func ToChromeMachines(old []*crimson.Machine, machineToNics map[string][]string, machineToDracs map[string]string) []*ufspb.Machine {
	newObjects := make([]*ufspb.Machine, len(old))
	for i, o := range old {
		newObjects[i] = &ufspb.Machine{
			// Temporarily use existing display name as browser machine's name instead of serial number/assettag
			Name:         o.Name,
			Location:     toLocation(o.Rack, o.Datacenter),
			SerialNumber: o.GetServiceTag(),
			Device: &ufspb.Machine_ChromeBrowserMachine{
				ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{
					// RpmInterface is not available for browser machine.
					// KvmInterface is currently attached to rack.
					// NetworkDeviceInterface is attached to the nics.
					DisplayName:      o.Name,
					ChromePlatform:   FormatResourceName(o.Platform),
					Description:      o.GetDescription(),
					DeploymentTicket: o.DeploymentTicket,
				},
			},
			Realm: BrowserLabAdminRealm,
			State: ToState(o.GetState()).String(),
		}
	}
	return newObjects
}

func toLocation(rack, datacenter string) *ufspb.Location {
	return &ufspb.Location{
		Rack: rack,
		Zone: ToZone(strings.ToLower(datacenter)),
	}
}

// ToChromePlatforms converts platforms in static file to UFS format.
func ToChromePlatforms(oldP *crimsonconfig.Platforms) []*ufspb.ChromePlatform {
	ps := oldP.GetPlatform()
	newP := make([]*ufspb.ChromePlatform, len(ps))
	for i, p := range ps {
		newP[i] = &ufspb.ChromePlatform{
			Name:         FormatResourceName(p.GetName()),
			Manufacturer: p.GetManufacturer(),
			Description:  p.GetDescription(),
		}
	}
	return newP
}

// ToOses converts the os versions to UFS format.
func ToOses(old []*crimson.OS) []*ufspb.OSVersion {
	newOSes := make([]*ufspb.OSVersion, len(old))
	for i, p := range old {
		newOSes[i] = &ufspb.OSVersion{
			Value:       FormatResourceName(p.GetName()),
			Description: p.GetDescription(),
		}
	}
	return newOSes
}

// ProcessDatacenters converts datacenters to several UFS objects
func ProcessDatacenters(dc *crimsonconfig.Datacenter) ([]*ufspb.Rack, []*ufspb.RackLSE, []*ufspb.KVM, []*ufspb.Switch, []*ufspb.DHCPConfig) {
	dcName := dc.GetName()
	switches := make([]*ufspb.Switch, 0)
	racks := make([]*ufspb.Rack, 0)
	rackLSEs := make([]*ufspb.RackLSE, 0)
	rackToKvms := make(map[string][]string, 0)
	kvms := make([]*ufspb.KVM, 0)
	dhcps := make([]*ufspb.DHCPConfig, 0)
	for _, oldKVM := range dc.GetKvm() {
		name := oldKVM.GetName()
		k := &ufspb.KVM{
			Name:           name,
			MacAddress:     oldKVM.GetMacAddress(),
			ChromePlatform: FormatResourceName(oldKVM.GetPlatform()),
			Rack:           oldKVM.GetRack(),
			Zone:           ToZone(strings.ToLower(dcName)).String(),
			State:          ToState(oldKVM.GetState()).String(),
			Description:    oldKVM.GetDescription(),
		}
		kvms = append(kvms, k)
		rackName := oldKVM.GetRack()
		rackToKvms[rackName] = append(rackToKvms[rackName], name)
		dhcps = append(dhcps, &ufspb.DHCPConfig{
			MacAddress: oldKVM.GetMacAddress(),
			Hostname:   name,
			Ip:         oldKVM.GetIpv4(),
		})
	}
	for _, old := range dc.GetRack() {
		rackName := old.GetName()
		switchNames := make([]string, 0)
		for _, crimsonSwitch := range old.GetSwitch() {
			s := &ufspb.Switch{
				Name:         crimsonSwitch.GetName(),
				CapacityPort: crimsonSwitch.GetPorts(),
				Description:  crimsonSwitch.GetDescription(),
				Rack:         rackName,
				Zone:         ToZone(strings.ToLower(dcName)).String(),
				State:        ToState(crimsonSwitch.GetState()).String(),
			}
			switches = append(switches, s)
			switchNames = append(switchNames, s.GetName())
		}
		// Also add the kvms which is attached to the rack in rack definitation
		found := false
		for _, rack := range rackToKvms[rackName] {
			if rack == old.GetKvm() {
				found = true
				break
			}
		}
		if !found {
			rackToKvms[rackName] = append(rackToKvms[rackName], old.GetKvm())
		}
		r := &ufspb.Rack{
			Name:     rackName,
			Location: toLocation(rackName, dcName),
			Rack: &ufspb.Rack_ChromeBrowserRack{
				ChromeBrowserRack: &ufspb.ChromeBrowserRack{},
			},
			State:       ToState(old.GetState()).String(),
			Description: old.GetDescription(),
		}
		rlse := &ufspb.RackLSE{
			Name:             GetRackHostname(rackName),
			RackLsePrototype: "browser-lab:normal",
			Lse: &ufspb.RackLSE_ChromeBrowserRackLse{
				ChromeBrowserRackLse: &ufspb.ChromeBrowserRackLSE{
					// Still keep them as they are potential hostnames
					Kvms:     rackToKvms[rackName],
					Switches: switchNames,
				},
			},
			Racks: []string{rackName},
		}
		racks = append(racks, r)
		rackLSEs = append(rackLSEs, rlse)
	}
	return racks, rackLSEs, kvms, switches, dhcps
}

// ProcessNetworkInterfaces converts nics and dracs to several UFS formats for further importing
func ProcessNetworkInterfaces(nics []*crimson.NIC, dracs []*crimson.DRAC, machines []*crimson.Machine) ([]*ufspb.Nic, []*ufspb.Drac, []*ufspb.DHCPConfig, map[string][]string, map[string]string) {
	machineToNics := make(map[string][]string, 0)
	machineToDracs := make(map[string]string, 0)
	machineMap := make(map[string]*crimson.Machine, len(machines))
	newNics := make([]*ufspb.Nic, 0)
	newDracs := make([]*ufspb.Drac, 0)
	dhcps := make([]*ufspb.DHCPConfig, 0)
	for _, machine := range machines {
		machineMap[machine.GetName()] = machine
	}
	for _, nic := range nics {
		name := GetNicName(nic.GetName(), nic.GetMachine())
		switch nic.GetName() {
		case "drac":
			// Use ListDrac() as the source of truth for drac
			continue
		default:
			// zone and rack are for indexing nic table
			var rack string
			var zone string
			machine, ok := machineMap[nic.GetMachine()]
			if ok {
				rack = machine.GetRack()
				zone = ToZone(strings.ToLower(machine.GetDatacenter())).String()
			}
			// Multiple nic names, e.g. eth0, eth1, bmc
			newNic := &ufspb.Nic{
				Name:       name,
				MacAddress: nic.GetMacAddress(),
				SwitchInterface: &ufspb.SwitchInterface{
					Switch:   nic.GetSwitch(),
					PortName: Int32ToStr(nic.GetSwitchport()),
				},
				Rack:    rack,
				Zone:    zone,
				Machine: nic.GetMachine(),
			}
			newNics = append(newNics, newNic)
			machineToNics[nic.GetMachine()] = append(machineToNics[nic.GetMachine()], name)
		}
	}
	for _, drac := range dracs {
		// zone and rack are for indexing drac table
		var rack string
		var zone string
		machine, ok := machineMap[drac.GetMachine()]
		if ok {
			rack = machine.GetRack()
			zone = ToZone(strings.ToLower(machine.GetDatacenter())).String()
		}
		hostname := FormatResourceName(drac.GetName())
		d := &ufspb.Drac{
			Name: hostname,
			// Inject machine name to display name
			DisplayName: GetNicName("drac", drac.GetMachine()),
			MacAddress:  drac.GetMacAddress(),
			SwitchInterface: &ufspb.SwitchInterface{
				Switch:   drac.GetSwitch(),
				PortName: Int32ToStr(drac.GetSwitchport()),
			},
			Rack:    rack,
			Zone:    zone,
			Machine: drac.GetMachine(),
		}
		newDracs = append(newDracs, d)
		machineToDracs[drac.GetMachine()] = hostname
		if ip := drac.GetIpv4(); ip != "" {
			dhcps = append(dhcps, &ufspb.DHCPConfig{
				MacAddress: drac.GetMacAddress(),
				Hostname:   hostname,
				Ip:         drac.GetIpv4(),
				Vlan:       GetBrowserLabName(Int64ToStr(drac.GetVlan())),
			})
		}
	}
	return newNics, newDracs, dhcps, machineToNics, machineToDracs
}

// ToMachineLSEs converts crimson data to UFS LSEs.
func ToMachineLSEs(hosts []*crimson.PhysicalHost, vms []*crimson.VM, machines []*crimson.Machine, platforms []*crimson.Platform) ([]*ufspb.MachineLSE, []*ufspb.VM, []*ufspb.IP, []*ufspb.DHCPConfig) {
	hostToVMs := make(map[string][]*ufspb.VM, 0)
	ufsVMs := make([]*ufspb.VM, 0)
	ips := make([]*ufspb.IP, 0)
	dhcps := make([]*ufspb.DHCPConfig, 0)
	machineMap := make(map[string]*crimson.Machine, len(machines))
	platformMap := make(map[string]*crimson.Platform, len(platforms))
	hostToMachine := make(map[string]*crimson.Machine, len(hosts))
	for _, machine := range machines {
		machineMap[machine.GetName()] = machine
	}
	for _, p := range platforms {
		platformMap[p.GetName()] = p
	}
	for _, h := range hosts {
		hostToMachine[h.GetName()] = machineMap[h.GetMachine()]
	}
	for _, vm := range vms {
		name := vm.GetName()
		var zone string
		if machine, ok := hostToMachine[vm.GetHost()]; ok {
			zone = ToZone(strings.ToLower(machine.GetDatacenter())).String()
		}
		v := &ufspb.VM{
			Name: name,
			OsVersion: &ufspb.OSVersion{
				Value: FormatResourceName(vm.GetOs()),
			},
			Hostname:     name,
			Vlan:         GetBrowserLabName(Int64ToStr(vm.GetVlan())),
			Zone:         zone,
			MachineLseId: vm.GetHost(),
			State:        ToState(vm.GetState()).String(),
			Description:  vm.GetDescription(),
		}
		hostToVMs[vm.GetHost()] = append(hostToVMs[vm.GetHost()], v)
		ufsVMs = append(ufsVMs, v)
		ip := FormatIP(GetBrowserLabName(Int64ToStr(vm.GetVlan())), vm.GetIpv4(), true)
		if ip != nil {
			ips = append(ips, ip)
		}
		dhcps = append(dhcps, &ufspb.DHCPConfig{
			Hostname: v.GetHostname(),
			Ip:       vm.GetIpv4(),
			Vlan:     GetBrowserLabName(Int64ToStr(vm.GetVlan())),
			// No mac address found
		})
	}
	lses := make([]*ufspb.MachineLSE, 0)
	var lsePrototype string
	for _, h := range hosts {
		var rack string
		var zone string
		machine, ok := machineMap[h.GetMachine()]
		if ok {
			rack = machine.GetRack()
			zone = ToZone(strings.ToLower(machine.GetDatacenter())).String()
		}
		name := h.GetName()
		vms := hostToVMs[name]
		if len(vms) > 0 {
			lsePrototype = "browser-lab:vm"
		} else {
			lsePrototype = "browser-lab:no-vm"
		}
		var manufacturer string
		if machine.GetPlatform() != "" {
			manufacturer = platformMap[machine.GetPlatform()].GetManufacturer()
		}
		lse := &ufspb.MachineLSE{
			Name:                name,
			MachineLsePrototype: lsePrototype,
			Hostname:            name,
			Machines:            []string{h.GetMachine()},
			Lse: &ufspb.MachineLSE_ChromeBrowserMachineLse{
				ChromeBrowserMachineLse: &ufspb.ChromeBrowserMachineLSE{
					Vms:        vms,
					VmCapacity: h.GetVmSlots(),
					OsVersion: &ufspb.OSVersion{
						Value: FormatResourceName(h.GetOs()),
					},
					VirtualDatacenter: h.GetVirtualDatacenter(),
				},
			},
			Rack:             rack,
			Zone:             zone,
			Nic:              GetNicName(h.GetNic(), h.GetMachine()),
			State:            ToState(h.GetState()).String(),
			Manufacturer:     manufacturer,
			Description:      h.GetDescription(),
			DeploymentTicket: h.DeploymentTicket,
		}
		lses = append(lses, lse)
		ip := FormatIP(GetBrowserLabName(Int64ToStr(h.GetVlan())), h.GetIpv4(), true)
		if ip != nil {
			ips = append(ips, ip)
		}
		dhcps = append(dhcps, &ufspb.DHCPConfig{
			Hostname:   h.GetName(),
			Ip:         h.GetIpv4(),
			MacAddress: h.GetMacAddress(),
			Vlan:       GetBrowserLabName(Int64ToStr(h.GetVlan())),
		})
	}
	return lses, ufsVMs, ips, dhcps
}

// ToState converts crimson state to UFS state.
func ToState(state crimsoncommon.State) ufspb.State {
	switch state {
	case crimsoncommon.State_SERVING:
		return ufspb.State_STATE_SERVING
	case crimsoncommon.State_DECOMMISSIONED:
		return ufspb.State_STATE_DECOMMISSIONED
	case crimsoncommon.State_REPAIR:
		return ufspb.State_STATE_NEEDS_REPAIR
	case crimsoncommon.State_TEST:
		return ufspb.State_STATE_DEPLOYED_TESTING
	case crimsoncommon.State_PRERELEASE:
		return ufspb.State_STATE_DEPLOYED_PRE_SERVING
	case crimsoncommon.State_FREE:
		return ufspb.State_STATE_REGISTERED
	}
	return ufspb.State_STATE_UNSPECIFIED
}

// ToLab converts the crimson lab string to UFS lab.
func ToLab(datacenter string) ufspb.Lab {
	switch strings.ToLower(datacenter) {
	case "atl97":
		return ufspb.Lab_LAB_DATACENTER_ATL97
	case "iad97":
		return ufspb.Lab_LAB_DATACENTER_IAD97
	case "mtv96":
		return ufspb.Lab_LAB_DATACENTER_MTV96
	case "mtv97":
		return ufspb.Lab_LAB_DATACENTER_MTV97
	case "lab01":
		return ufspb.Lab_LAB_DATACENTER_FUCHSIA
	default:
		return ufspb.Lab_LAB_UNSPECIFIED
	}
}

// ToZone converts the crimson lab string to UFS zone.
func ToZone(datacenter string) ufspb.Zone {
	switch strings.ToLower(datacenter) {
	case "atl97":
		return ufspb.Zone_ZONE_ATL97
	case "iad97":
		return ufspb.Zone_ZONE_IAD97
	case "mtv96":
		return ufspb.Zone_ZONE_MTV96
	case "mtv97":
		return ufspb.Zone_ZONE_MTV97
	case "lab01":
		return ufspb.Zone_ZONE_FUCHSIA
	case "atl":
		return ufspb.Zone_ZONE_ATLANTA
	default:
		return ufspb.Zone_ZONE_UNSPECIFIED
	}
}

// GetNicName formats a nic name with its attached machine
func GetNicName(nicName, machineName string) string {
	return fmt.Sprintf("%s:%s", machineName, nicName)
}
