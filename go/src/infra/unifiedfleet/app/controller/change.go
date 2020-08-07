// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"context"
	"fmt"

	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/server/auth"

	ufspb "infra/unifiedfleet/api/v1/proto"
	"infra/unifiedfleet/app/model/history"
	"infra/unifiedfleet/app/util"
)

// SaveChangeEvents saves change events to database
func SaveChangeEvents(ctx context.Context, changes []*ufspb.ChangeEvent) error {
	logging.Debugf(ctx, "Logging %d changes", len(changes))
	user := auth.CurrentUser(ctx)
	for _, c := range changes {
		c.UserEmail = user.Email
	}
	_, err := history.CreateBatchChangeEvents(ctx, changes)
	if err != nil {
		logging.Debugf(ctx, "fail to log changes: %s", err.Error())
		return err
	}
	logging.Debugf(ctx, "Finish logging changes successfully")
	return nil
}

const (
	// LifeCycleRegistration indicates the registration of a resource.
	LifeCycleRegistration = "REGISTRATION"

	// LifeCycleDeployment indicates the deployment of a device.
	LifeCycleDeployment = "DEPLOYMENT"

	// LifeCycleDecomm indicates the decommission of a device.
	LifeCycleDecomm = "DECOMMISSION"

	// LifeCycleRetire indicates the retirement of a device.
	LifeCycleRetire = "RETIREMENT"
)

// logLifeCycle logs the life cycle event of a ChromeOSDevice.
func logLifeCycle(resourceName, resourceType, lifeCycleType string) (changes []*ufspb.ChangeEvent) {
	return []*ufspb.ChangeEvent{
		{
			Name:       resourceName,
			EventLabel: resourceType,
			OldValue:   lifeCycleType,
			NewValue:   lifeCycleType,
		},
	}
}

// LogMachineChanges logs the change of the given machine.
func LogMachineChanges(oldData *ufspb.Machine, newData *ufspb.Machine) []*ufspb.ChangeEvent {
	changes := make([]*ufspb.ChangeEvent, 0)
	if oldData == nil && newData == nil {
		return changes
	}
	if oldData == nil {
		return append(changes, logLifeCycle(util.AddPrefix(util.MachineCollection, newData.GetName()), "machine", LifeCycleRegistration)...)
	}
	if newData == nil {
		return append(changes, logLifeCycle(util.AddPrefix(util.MachineCollection, oldData.GetName()), "machine", LifeCycleRetire)...)
	}
	changes = append(changes, logCommon("", "machine.serial_number", oldData.GetSerialNumber(), newData.GetSerialNumber())...)
	changes = append(changes, logCommon("", "machine.location", oldData.GetLocation(), newData.GetLocation())...)
	changes = append(changes, logCommon("", "machine.realm", oldData.GetRealm(), newData.GetRealm())...)
	if newData.GetChromeBrowserMachine() != nil {
		changes = append(changes, logChromeBrowserMachine(oldData.GetChromeBrowserMachine(), newData.GetChromeBrowserMachine())...)
	}

	// Set resource name for all changes.
	for i := range changes {
		changes[i].Name = util.AddPrefix(util.MachineCollection, oldData.GetName())
	}
	return changes
}

// LogMachineLSEChanges logs the change of the given machine lse.
func LogMachineLSEChanges(oldData *ufspb.MachineLSE, newData *ufspb.MachineLSE) []*ufspb.ChangeEvent {
	changes := make([]*ufspb.ChangeEvent, 0)
	if oldData == nil && newData == nil {
		return changes
	}
	if oldData == nil {
		return append(changes, logLifeCycle(util.AddPrefix(util.HostCollection, newData.GetName()), "machine_lse", LifeCycleRegistration)...)
	}
	if newData == nil {
		return append(changes, logLifeCycle(util.AddPrefix(util.HostCollection, oldData.GetName()), "machine_lse", LifeCycleRetire)...)
	}
	changes = append(changes, logCommon("", "machine_lse.machine_lse_prototype", oldData.GetMachineLsePrototype(), newData.GetMachineLsePrototype())...)
	changes = append(changes, logCommon("", "machine_lse.hostname", oldData.GetHostname(), newData.GetHostname())...)
	changes = append(changes, logCommon("", "machine_lse.machines", oldData.GetMachines(), newData.GetMachines())...)
	changes = append(changes, logCommon("", "machine_lse.nic", oldData.GetNic(), newData.GetNic())...)
	changes = append(changes, logCommon("", "machine_lse.lab", approxLab(oldData.GetLab()), approxLab(newData.GetLab()))...)
	changes = append(changes, logCommon("", "machine_lse.rack", oldData.GetRack(), newData.GetRack())...)
	changes = append(changes, logCommon("", "machine_lse.manufacturer", oldData.GetManufacturer(), newData.GetManufacturer())...)
	if newData.GetChromeBrowserMachineLse() != nil {
		changes = append(changes, logChromeBrowserMachineLse(oldData.GetChromeBrowserMachineLse(), newData.GetChromeBrowserMachineLse())...)
	}

	// Set resource name for all changes.
	for i := range changes {
		changes[i].Name = util.AddPrefix(util.HostCollection, oldData.GetName())
	}

	return changes
}

func approxLab(lab string) string {
	if lab == "" {
		return ufspb.Lab_LAB_UNSPECIFIED.String()
	}
	return lab
}

func approxState(s string) string {
	if s == "" {
		return ufspb.State_STATE_UNSPECIFIED.String()
	}
	return s
}

// LogVMChanges logs the change of the given vms.
func LogVMChanges(oldData *ufspb.VM, newData *ufspb.VM) []*ufspb.ChangeEvent {
	changes := make([]*ufspb.ChangeEvent, 0)
	if oldData == nil && newData == nil {
		return changes
	}
	if oldData == nil {
		return append(changes, logLifeCycle(util.AddPrefix(util.VMCollection, newData.GetName()), "vm", LifeCycleRegistration)...)
	}
	if newData == nil {
		return append(changes, logLifeCycle(util.AddPrefix(util.VMCollection, oldData.GetName()), "vm", LifeCycleRetire)...)
	}
	if oldData.GetName() != newData.GetName() {
		return nil
	}
	resourceName := util.AddPrefix(util.VMCollection, oldData.GetName())
	changes = append(changes, logCommon(resourceName, "vm.mac_address", oldData.GetMacAddress(), newData.GetMacAddress())...)
	changes = append(changes, logCommon(resourceName, "vm.os_version", oldData.GetOsVersion(), newData.GetOsVersion())...)
	changes = append(changes, logCommon(resourceName, "vm.vlan", oldData.GetVlan(), newData.GetVlan())...)
	changes = append(changes, logCommon(resourceName, "vm.lab", approxLab(oldData.GetLab()), approxLab(newData.GetLab()))...)
	changes = append(changes, logCommon(resourceName, "vm.machine_lse_id", oldData.GetMachineLseId(), newData.GetMachineLseId())...)
	changes = append(changes, logCommon(resourceName, "vm.state", approxState(oldData.GetState()), approxState(newData.GetState()))...)
	return changes
}

// LogRackChanges logs the change of the given rack.
func LogRackChanges(oldData *ufspb.Rack, newData *ufspb.Rack) []*ufspb.ChangeEvent {
	changes := make([]*ufspb.ChangeEvent, 0)
	if oldData == nil && newData == nil {
		return changes
	}
	if oldData == nil {
		return append(changes, logLifeCycle(util.AddPrefix(util.RackCollection, newData.GetName()), "rack", LifeCycleRegistration)...)
	}
	if newData == nil {
		return append(changes, logLifeCycle(util.AddPrefix(util.RackCollection, oldData.GetName()), "rack", LifeCycleRetire)...)
	}
	changes = append(changes, logCommon("", "rack.location", oldData.GetLocation(), newData.GetLocation())...)
	changes = append(changes, logCommon("", "rack.capacity_ru", oldData.GetCapacityRu(), newData.GetCapacityRu())...)
	changes = append(changes, logCommon("", "rack.realm", oldData.GetRealm(), newData.GetRealm())...)
	if newData.GetChromeBrowserRack() != nil {
		changes = append(changes, logChromeBrowserRack(oldData.GetChromeBrowserRack(), newData.GetChromeBrowserRack())...)
	}

	// Set resource name for all changes.
	for i := range changes {
		changes[i].Name = util.AddPrefix(util.RackCollection, oldData.GetName())
	}
	return changes
}

// LogNicChanges logs the change of the given nic.
func LogNicChanges(oldData, newData *ufspb.Nic) []*ufspb.ChangeEvent {
	changes := make([]*ufspb.ChangeEvent, 0)
	if oldData == nil && newData == nil {
		return changes
	}
	if oldData == nil {
		return append(changes, logLifeCycle(util.AddPrefix(util.NicCollection, newData.GetName()), "nic", LifeCycleRegistration)...)
	}
	if newData == nil {
		return append(changes, logLifeCycle(util.AddPrefix(util.NicCollection, oldData.GetName()), "nic", LifeCycleRetire)...)
	}
	changes = append(changes, logCommon("", "nic.mac_address", oldData.GetMacAddress(), newData.GetMacAddress())...)
	changes = append(changes, logCommon("", "nic.machine", oldData.GetMachine(), newData.GetMachine())...)
	changes = append(changes, logCommon("", "nic.lab", approxLab(oldData.GetLab()), approxLab(newData.GetLab()))...)
	changes = append(changes, logCommon("", "nic.rack", oldData.GetRack(), newData.GetRack())...)
	changes = append(changes, logSwitchInterface(oldData.GetSwitchInterface(), newData.GetSwitchInterface())...)

	// Set resource name for all changes.
	for i := range changes {
		changes[i].Name = util.AddPrefix(util.NicCollection, oldData.GetName())
	}
	return changes
}

// LogDracChanges logs the change of the given drac.
func LogDracChanges(oldData, newData *ufspb.Drac) []*ufspb.ChangeEvent {
	changes := make([]*ufspb.ChangeEvent, 0)
	if oldData == nil && newData == nil {
		return changes
	}
	if oldData == nil {
		return append(changes, logLifeCycle(util.AddPrefix(util.DracCollection, newData.GetName()), "drac", LifeCycleRegistration)...)
	}
	if newData == nil {
		return append(changes, logLifeCycle(util.AddPrefix(util.DracCollection, oldData.GetName()), "drac", LifeCycleRetire)...)
	}
	changes = append(changes, logCommon("", "drac.display_name", oldData.GetDisplayName(), newData.GetDisplayName())...)
	changes = append(changes, logCommon("", "drac.mac_address", oldData.GetMacAddress(), newData.GetMacAddress())...)
	changes = append(changes, logCommon("", "drac.machine", oldData.GetMachine(), newData.GetMachine())...)
	changes = append(changes, logCommon("", "drac.lab", approxLab(oldData.GetLab()), approxLab(newData.GetLab()))...)
	changes = append(changes, logCommon("", "drac.rack", oldData.GetRack(), newData.GetRack())...)
	changes = append(changes, logSwitchInterface(oldData.GetSwitchInterface(), newData.GetSwitchInterface())...)

	// Set resource name for all changes.
	for i := range changes {
		changes[i].Name = util.AddPrefix(util.DracCollection, oldData.GetName())
	}
	return changes
}

// LogKVMChanges logs the change of the given kvm.
func LogKVMChanges(oldData, newData *ufspb.KVM) []*ufspb.ChangeEvent {
	changes := make([]*ufspb.ChangeEvent, 0)
	if oldData == nil && newData == nil {
		return changes
	}
	if oldData == nil {
		return append(changes, logLifeCycle(util.AddPrefix(util.KVMCollection, newData.GetName()), "kvm", LifeCycleRegistration)...)
	}
	if newData == nil {
		return append(changes, logLifeCycle(util.AddPrefix(util.KVMCollection, oldData.GetName()), "kvm", LifeCycleRetire)...)
	}
	changes = append(changes, logCommon("", "kvm.mac_address", oldData.GetMacAddress(), newData.GetMacAddress())...)
	changes = append(changes, logCommon("", "kvm.chrome_platform", oldData.GetChromePlatform(), newData.GetChromePlatform())...)
	changes = append(changes, logCommon("", "kvm.capacity_port", oldData.GetCapacityPort(), newData.GetCapacityPort())...)
	changes = append(changes, logCommon("", "kvm.lab", approxLab(oldData.GetLab()), approxLab(newData.GetLab()))...)
	changes = append(changes, logCommon("", "kvm.rack", oldData.GetRack(), newData.GetRack())...)

	// Set resource name for all changes.
	for i := range changes {
		changes[i].Name = util.AddPrefix(util.KVMCollection, oldData.GetName())
	}
	return changes
}

// LogSwitchChanges logs the change of the given switch.
func LogSwitchChanges(oldData, newData *ufspb.Switch) []*ufspb.ChangeEvent {
	changes := make([]*ufspb.ChangeEvent, 0)
	if oldData == nil && newData == nil {
		return changes
	}
	if oldData == nil {
		return append(changes, logLifeCycle(util.AddPrefix(util.SwitchCollection, newData.GetName()), "switch", LifeCycleRegistration)...)
	}
	if newData == nil {
		return append(changes, logLifeCycle(util.AddPrefix(util.SwitchCollection, oldData.GetName()), "switch", LifeCycleRetire)...)
	}
	changes = append(changes, logCommon("", "switch.description", oldData.GetDescription(), newData.GetDescription())...)
	changes = append(changes, logCommon("", "switch.capacity_port", oldData.GetCapacityPort(), newData.GetCapacityPort())...)
	changes = append(changes, logCommon("", "switch.lab", approxLab(oldData.GetLab()), approxLab(newData.GetLab()))...)
	changes = append(changes, logCommon("", "switch.rack", oldData.GetRack(), newData.GetRack())...)

	// Set resource name for all changes.
	for i := range changes {
		changes[i].Name = util.AddPrefix(util.SwitchCollection, oldData.GetName())
	}
	return changes
}

// LogRPMChanges logs the change of the given rpms.
func LogRPMChanges(oldData, newData *ufspb.RPM) []*ufspb.ChangeEvent {
	changes := make([]*ufspb.ChangeEvent, 0)
	if oldData == nil && newData == nil {
		return changes
	}
	if oldData == nil {
		return append(changes, logLifeCycle(util.AddPrefix(util.RPMCollection, newData.GetName()), "rpm", LifeCycleRegistration)...)
	}
	if newData == nil {
		return append(changes, logLifeCycle(util.AddPrefix(util.RPMCollection, oldData.GetName()), "rpm", LifeCycleRetire)...)
	}
	changes = append(changes, logCommon("", "rpm.mac_address", oldData.GetMacAddress(), newData.GetMacAddress())...)
	changes = append(changes, logCommon("", "rpm.capacity_port", oldData.GetCapacityPort(), newData.GetCapacityPort())...)
	changes = append(changes, logCommon("", "rpm.lab", approxLab(oldData.GetLab()), approxLab(newData.GetLab()))...)
	changes = append(changes, logCommon("", "rpm.rack", oldData.GetRack(), newData.GetRack())...)

	// Set resource name for all changes.
	for i := range changes {
		changes[i].Name = util.AddPrefix(util.RPMCollection, oldData.GetName())
	}
	return changes
}

// LogDHCPChanges logs the change of the given dhcp.
func LogDHCPChanges(oldData, newData *ufspb.DHCPConfig) []*ufspb.ChangeEvent {
	changes := make([]*ufspb.ChangeEvent, 0)
	if oldData == nil && newData == nil {
		return changes
	}
	if oldData == nil {
		return append(changes, logCommon(util.AddPrefix(util.DHCPCollection, newData.GetHostname()), "dhcp_config.ip", "", newData.GetIp())...)
	}
	if newData == nil {
		return append(changes, logCommon(util.AddPrefix(util.DHCPCollection, oldData.GetHostname()), "dhcp_config.ip", oldData.GetIp(), "")...)
	}
	resourceName := util.AddPrefix(util.DHCPCollection, oldData.GetHostname())
	changes = append(changes, logCommon(resourceName, "dhcp_config.mac_address", oldData.GetMacAddress(), newData.GetMacAddress())...)
	changes = append(changes, logCommon(resourceName, "dhcp_config.ip", oldData.GetIp(), newData.GetIp())...)
	changes = append(changes, logCommon(resourceName, "dhcp_config.vlan", oldData.GetVlan(), newData.GetVlan())...)

	return changes
}

// LogIPChanges logs the change of the given ip.
func LogIPChanges(oldData, newData *ufspb.IP) []*ufspb.ChangeEvent {
	changes := make([]*ufspb.ChangeEvent, 0)
	if oldData == nil && newData == nil {
		return changes
	}
	var resourceName string
	if oldData != nil {
		resourceName = util.AddPrefix(util.IPCollection, oldData.GetId())
	} else {
		resourceName = util.AddPrefix(util.IPCollection, newData.GetId())
	}
	changes = append(changes, logCommon(resourceName, "ip.occupied", oldData.GetOccupied(), newData.GetOccupied())...)

	return changes
}

// LogStateChanges logs the change of the given state record.
func LogStateChanges(oldData, newData *ufspb.StateRecord) []*ufspb.ChangeEvent {
	changes := make([]*ufspb.ChangeEvent, 0)
	if oldData == nil && newData == nil {
		return changes
	}
	var resourceName string
	if oldData != nil {
		resourceName = util.AddPrefix(util.StateCollection, oldData.GetResourceName())
	} else {
		resourceName = util.AddPrefix(util.StateCollection, newData.GetResourceName())
	}
	changes = append(changes, logCommon(resourceName, "state_record.state", oldData.GetState().String(), newData.GetState().String())...)

	return changes
}

func logChromeBrowserRack(oldData, newData *ufspb.ChromeBrowserRack) []*ufspb.ChangeEvent {
	changes := make([]*ufspb.ChangeEvent, 0)
	changes = append(changes, logCommon("", "rack.chrome_browser_rack.rpms", oldData.GetRpms(), newData.GetRpms())...)
	changes = append(changes, logCommon("", "rack.chrome_browser_rack.kvms", oldData.GetKvms(), newData.GetKvms())...)
	changes = append(changes, logCommon("", "rack.chrome_browser_rack.switches", oldData.GetSwitches(), newData.GetSwitches())...)
	return changes
}

func logChromeBrowserMachineLse(oldData, newData *ufspb.ChromeBrowserMachineLSE) []*ufspb.ChangeEvent {
	changes := make([]*ufspb.ChangeEvent, 0)
	changes = append(changes, logCommon("", "machine_lse.chrome_browser_machine_lse.vm_capacity", oldData.GetVmCapacity(), newData.GetVmCapacity())...)
	changes = append(changes, logCommon("", "machine_lse.chrome_browser_machine_lse.os_version", oldData.GetOsVersion(), newData.GetOsVersion())...)
	return changes
}

func logChromeBrowserMachine(oldData, newData *ufspb.ChromeBrowserMachine) []*ufspb.ChangeEvent {
	changes := make([]*ufspb.ChangeEvent, 0)
	changes = append(changes, logCommon("", "machine.chrome_browser_machine.display_name", oldData.GetDisplayName(), newData.GetDisplayName())...)
	changes = append(changes, logCommon("", "machine.chrome_browser_machine.chrome_platform", oldData.GetChromePlatform(), newData.GetChromePlatform())...)
	changes = append(changes, logCommon("", "machine.chrome_browser_machine.nics", oldData.GetNics(), newData.GetNics())...)
	changes = append(changes, logCommon("", "machine.chrome_browser_machine.drac", oldData.GetDrac(), newData.GetDrac())...)
	changes = append(changes, logCommon("", "machine.chrome_browser_machine.deployment_ticket", oldData.GetDeploymentTicket(), newData.GetDeploymentTicket())...)
	changes = append(changes, logKVMInterface(oldData.GetKvmInterface(), newData.GetKvmInterface())...)
	changes = append(changes, logRPMInterface(oldData.GetRpmInterface(), newData.GetRpmInterface())...)
	return changes
}

func logKVMInterface(oldData, newData *ufspb.KVMInterface) []*ufspb.ChangeEvent {
	changes := make([]*ufspb.ChangeEvent, 0)
	changes = append(changes, logCommon("", "machine.chrome_browser_machine.kvm_interface.kvm", oldData.GetKvm(), newData.GetKvm())...)
	return append(changes, logCommon("", "machine.chrome_browser_machine.kvm_interface.port", oldData.GetPort(), newData.GetPort())...)
}

func logRPMInterface(oldData, newData *ufspb.RPMInterface) []*ufspb.ChangeEvent {
	changes := make([]*ufspb.ChangeEvent, 0)
	changes = append(changes, logCommon("", "machine.chrome_browser_machine.rpm_interface.rpm", oldData.GetRpm(), newData.GetRpm())...)
	return append(changes, logCommon("", "machine.chrome_browser_machine.rpm_interface.port", oldData.GetPort(), newData.GetPort())...)
}

func logSwitchInterface(oldData, newData *ufspb.SwitchInterface) []*ufspb.ChangeEvent {
	changes := make([]*ufspb.ChangeEvent, 0)
	changes = append(changes, logCommon("", "switch_interface.switch", oldData.GetSwitch(), newData.GetSwitch())...)
	return append(changes, logCommon("", "switch_interface.port", oldData.GetPort(), newData.GetPort())...)
}

func logCommon(name, label string, oldValue interface{}, newValue interface{}) []*ufspb.ChangeEvent {
	oldValueStr := fmt.Sprintf("%v", oldValue)
	newValueStr := fmt.Sprintf("%v", newValue)
	if oldValueStr == newValueStr {
		return nil
	}
	return []*ufspb.ChangeEvent{
		{
			Name:       name,
			EventLabel: label,
			OldValue:   oldValueStr,
			NewValue:   newValueStr,
		},
	}
}
