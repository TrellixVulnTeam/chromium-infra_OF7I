// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package util

import (
	"fmt"
	"strings"

	fleet "infra/unifiedfleet/api/v1/proto"

	crimsonconfig "go.chromium.org/luci/machine-db/api/config/v1"
	"go.chromium.org/luci/machine-db/api/crimson/v1"
)

// ToChromeMachines converts crimson machines to UFS format.
func ToChromeMachines(old []*crimson.Machine, machineToNics map[string]string, machineToDracs map[string]string, machineToSwitch map[string]*fleet.SwitchInterface) []*fleet.Machine {
	newObjects := make([]*fleet.Machine, len(old))
	for i, o := range old {
		newObjects[i] = &fleet.Machine{
			// Temporarily use existing display name as browser machine's name instead of serial number/assettag
			Name:     o.Name,
			Location: toLocation(o.Rack, o.Datacenter),
			Device: &fleet.Machine_ChromeBrowserMachine{
				ChromeBrowserMachine: &fleet.ChromeBrowserMachine{
					// RpmInterface is not available for browser machine.
					// KvmInterface is currently attached to rack.
					DisplayName:            o.Name,
					ChromePlatform:         o.Platform,
					Nic:                    machineToNics[o.Name],
					NetworkDeviceInterface: machineToSwitch[o.Name],
					Drac:                   machineToDracs[o.Name],
					DeploymentTicket:       o.DeploymentTicket,
				},
			},
		}
	}
	return newObjects
}

func toLocation(rack, datacenter string) *fleet.Location {
	l := &fleet.Location{
		Rack: rack,
	}
	switch strings.ToLower(datacenter) {
	case "atl97":
		l.Lab = fleet.Lab_LAB_DATACENTER_ATL97
	case "iad97":
		l.Lab = fleet.Lab_LAB_DATACENTER_IAD97
	case "mtv96":
		l.Lab = fleet.Lab_LAB_DATACENTER_MTV96
	case "mtv97":
		l.Lab = fleet.Lab_LAB_DATACENTER_MTV97
	case "lab01":
		l.Lab = fleet.Lab_LAB_DATACENTER_FUCHSIA
	default:
		l.Lab = fleet.Lab_LAB_UNSPECIFIED
	}
	return l
}

// ToChromePlatforms converts platforms in static file to UFS format.
func ToChromePlatforms(oldP *crimsonconfig.Platforms) []*fleet.ChromePlatform {
	ps := oldP.GetPlatform()
	newP := make([]*fleet.ChromePlatform, len(ps))
	for i, p := range ps {
		newP[i] = &fleet.ChromePlatform{
			Name:         p.GetName(),
			Manufacturer: p.GetManufacturer(),
			Description:  p.GetDescription(),
		}
	}
	return newP
}

// ProcessDatacenters converts datacenters to several UFS objects
func ProcessDatacenters(dc *crimsonconfig.Datacenter) ([]*fleet.Rack, []*fleet.KVM, []*fleet.Switch, []*fleet.DHCPConfig) {
	dcName := dc.GetName()
	switches := make([]*fleet.Switch, 0)
	racks := make([]*fleet.Rack, 0)
	rackToKvms := make(map[string][]string, 0)
	kvms := make([]*fleet.KVM, 0)
	dhcps := make([]*fleet.DHCPConfig, 0)
	for _, oldKVM := range dc.GetKvm() {
		name := oldKVM.GetName()
		k := &fleet.KVM{
			Name:           name,
			MacAddress:     oldKVM.GetMacAddress(),
			ChromePlatform: oldKVM.GetPlatform(),
		}
		kvms = append(kvms, k)
		rackName := oldKVM.GetRack()
		rackToKvms[rackName] = append(rackToKvms[rackName], name)
		dhcps = append(dhcps, &fleet.DHCPConfig{
			MacAddress: oldKVM.GetMacAddress(),
			Hostname:   name,
			Ip:         oldKVM.GetIpv4(),
		})
	}
	for _, old := range dc.GetRack() {
		rackName := old.GetName()
		switchNames := make([]string, 0)
		for _, crimsonSwitch := range old.GetSwitch() {
			s := &fleet.Switch{
				Name:         crimsonSwitch.GetName(),
				CapacityPort: crimsonSwitch.GetPorts(),
			}
			switches = append(switches, s)
			switchNames = append(switchNames, s.GetName())
		}
		r := &fleet.Rack{
			Name:     rackName,
			Location: toLocation(rackName, dcName),
			Rack: &fleet.Rack_ChromeBrowserRack{
				ChromeBrowserRack: &fleet.ChromeBrowserRack{
					Switches: switchNames,
					Kvms:     rackToKvms[rackName],
				},
			},
		}
		racks = append(racks, r)
	}
	return racks, kvms, switches, dhcps
}

// ProcessNics converts nics to several UFS formats for further importing
func ProcessNics(nics []*crimson.NIC) ([]*fleet.Nic, []*fleet.Drac, []*fleet.DHCPConfig, map[string]string, map[string]string, map[string]*fleet.SwitchInterface) {
	machineToSwitch := make(map[string]*fleet.SwitchInterface, 0)
	machineToNics := make(map[string]string, 0)
	machineToDracs := make(map[string]string, 0)
	newNics := make([]*fleet.Nic, 0)
	newDracs := make([]*fleet.Drac, 0)
	dhcps := make([]*fleet.DHCPConfig, len(nics))
	for i, nic := range nics {
		name := getNicName(nic)
		switch nic.GetName() {
		case "eth0":
			newNic := &fleet.Nic{
				Name:       name,
				MacAddress: nic.GetMacAddress(),
			}
			newNics = append(newNics, newNic)
			machineToSwitch[nic.GetMachine()] = &fleet.SwitchInterface{
				Switch: nic.GetSwitch(),
				Port:   nic.GetSwitchport(),
			}
			machineToNics[nic.GetMachine()] = name
		case "drac":
			d := &fleet.Drac{
				Name:        name,
				DisplayName: name,
				MacAddress:  nic.GetMacAddress(),
				SwitchInterface: &fleet.SwitchInterface{
					Switch: nic.GetSwitch(),
					Port:   nic.GetSwitchport(),
				},
			}
			newDracs = append(newDracs, d)
			machineToDracs[nic.GetMachine()] = name
		}
		if ip := nic.GetIpv4(); ip != "" {
			dhcps[i] = &fleet.DHCPConfig{
				MacAddress: nic.GetMacAddress(),
				Hostname:   name,
				Ip:         nic.GetIpv4(),
			}
		}
	}
	return newNics, newDracs, dhcps, machineToNics, machineToDracs, machineToSwitch
}

// ToMachineLSEs converts crimson data to UFS LSEs.
func ToMachineLSEs(hosts []*crimson.PhysicalHost, vms []*crimson.VM) ([]*fleet.MachineLSE, []*fleet.IP, []*fleet.DHCPConfig) {
	hostToVMs := make(map[string][]*fleet.VM, 0)
	ips := make([]*fleet.IP, 0)
	dhcps := make([]*fleet.DHCPConfig, 0)
	for _, vm := range vms {
		name := vm.GetName()
		v := &fleet.VM{
			Name: name,
			OsVersion: &fleet.OSVersion{
				Value: vm.GetOs(),
			},
			Hostname: getVMHostname(vm.GetHost(), vm.GetName()),
		}
		hostToVMs[vm.GetHost()] = append(hostToVMs[vm.GetHost()], v)
		ip := FormatIP(vm.GetVlan(), vm.GetIpv4(), true)
		if ip != nil {
			ips = append(ips, ip)
		}
		dhcps = append(dhcps, &fleet.DHCPConfig{
			Hostname: v.GetHostname(),
			Ip:       vm.GetIpv4(),
			// No mac address found
		})
	}
	lses := make([]*fleet.MachineLSE, 0)
	var lsePrototype string
	for _, h := range hosts {
		name := h.GetName()
		vms := hostToVMs[name]
		if len(vms) > 0 {
			lsePrototype = "browser-lab:vm"
		} else {
			lsePrototype = "browser-lab:no-vm"
		}
		lse := &fleet.MachineLSE{
			Name:                name,
			MachineLsePrototype: lsePrototype,
			Hostname:            name,
			Machines:            []string{h.GetMachine()},
			Lse: &fleet.MachineLSE_ChromeBrowserMachineLse{
				ChromeBrowserMachineLse: &fleet.ChromeBrowserMachineLSE{
					Vms: vms,
				},
			},
		}
		lses = append(lses, lse)
		ip := FormatIP(h.GetVlan(), h.GetIpv4(), true)
		if ip != nil {
			ips = append(ips, ip)
		}
		dhcps = append(dhcps, &fleet.DHCPConfig{
			Hostname:   h.GetName(),
			Ip:         h.GetIpv4(),
			MacAddress: h.GetMacAddress(),
		})
	}
	return lses, ips, dhcps
}

func getNicName(nic *crimson.NIC) string {
	return fmt.Sprintf("%s-%s", nic.GetMachine(), nic.GetName())
}

// GetBrowserLabVlanName return a browser lab vlan ID
func GetBrowserLabVlanName(id int64) string {
	return fmt.Sprintf("browser-lab:%d", id)
}

// getVMHostname return a vm hostname
func getVMHostname(phisicalHost, vmName string) string {
	return fmt.Sprintf("%s:%s", phisicalHost, vmName)
}
