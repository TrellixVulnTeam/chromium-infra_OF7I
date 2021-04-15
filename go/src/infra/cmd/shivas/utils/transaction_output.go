// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package utils

import (
	"context"
	"fmt"

	"go.chromium.org/luci/common/errors"

	ufspb "infra/unifiedfleet/api/v1/models"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	ufsUtil "infra/unifiedfleet/app/util"
)

// PrintExistingAsset prints the old asset in update/delete operations.
func PrintExistingAsset(ctx context.Context, ic ufsAPI.FleetClient, name string) (*ufspb.Asset, error) {
	res, err := ic.GetAsset(ctx, &ufsAPI.GetAssetRequest{
		Name: ufsUtil.AddPrefix(ufsUtil.AssetCollection, name),
	})
	if err != nil {
		return nil, errors.Annotate(err, "Failed to get asset").Err()
	}
	if res == nil {
		return nil, errors.Reason("The returned resp is empty").Err()
	}
	res.Name = ufsUtil.RemovePrefix(res.Name)
	fmt.Println("The asset before delete/update:")
	PrintProtoJSON(res, !NoEmitMode(false))
	return res, nil
}

// PrintExistingMachine prints the old machine in update/delete operations.
func PrintExistingMachine(ctx context.Context, ic ufsAPI.FleetClient, name string) (*ufspb.Machine, error) {
	res, err := ic.GetMachine(ctx, &ufsAPI.GetMachineRequest{
		Name: ufsUtil.AddPrefix(ufsUtil.MachineCollection, name),
	})
	if err != nil {
		return nil, errors.Annotate(err, "Failed to get machine").Err()
	}
	if res == nil {
		return nil, errors.Reason("The returned resp is empty").Err()
	}
	res.Name = ufsUtil.RemovePrefix(res.Name)
	fmt.Println("The machine before delete/update:")
	PrintProtoJSON(res, !NoEmitMode(false))
	return res, nil
}

// PrintExistingDrac prints the old drac in update/delete operations
func PrintExistingDrac(ctx context.Context, ic ufsAPI.FleetClient, name string) error {
	res, err := ic.GetDrac(ctx, &ufsAPI.GetDracRequest{
		Name: ufsUtil.AddPrefix(ufsUtil.DracCollection, name),
	})
	if err != nil {
		return errors.Annotate(err, "Failed to get drac").Err()
	}
	if res == nil {
		return errors.Reason("The returned resp is empty").Err()
	}
	res.Name = ufsUtil.RemovePrefix(res.Name)
	fmt.Println("The drac before delete/update:")
	PrintProtoJSON(res, !NoEmitMode(false))
	return nil
}

// PrintExistingNic prints the old nic in update/delete operations
func PrintExistingNic(ctx context.Context, ic ufsAPI.FleetClient, name string) error {
	res, err := ic.GetNic(ctx, &ufsAPI.GetNicRequest{
		Name: ufsUtil.AddPrefix(ufsUtil.NicCollection, name),
	})
	if err != nil {
		return errors.Annotate(err, "Failed to get nic").Err()
	}
	if res == nil {
		return errors.Reason("The returned resp is empty").Err()
	}
	res.Name = ufsUtil.RemovePrefix(res.Name)
	fmt.Println("The nic before delete/update:")
	PrintProtoJSON(res, !NoEmitMode(false))
	return nil
}

// PrintExistingRack prints the old rack in update/delete operations
func PrintExistingRack(ctx context.Context, ic ufsAPI.FleetClient, name string) error {
	res, err := ic.GetRack(ctx, &ufsAPI.GetRackRequest{
		Name: ufsUtil.AddPrefix(ufsUtil.RackCollection, name),
	})
	if err != nil {
		return errors.Annotate(err, "Failed to get rack").Err()
	}
	if res == nil {
		return errors.Reason("The returned resp is empty").Err()
	}
	res.Name = ufsUtil.RemovePrefix(res.Name)
	fmt.Println("The rack before delete/update:")
	PrintProtoJSON(res, !NoEmitMode(false))
	return nil
}

// PrintExistingKVM prints the old kvm in update/delete operations
func PrintExistingKVM(ctx context.Context, ic ufsAPI.FleetClient, name string) error {
	res, err := ic.GetKVM(ctx, &ufsAPI.GetKVMRequest{
		Name: ufsUtil.AddPrefix(ufsUtil.KVMCollection, name),
	})
	if err != nil {
		return errors.Annotate(err, "Failed to get kvm").Err()
	}
	if res == nil {
		return errors.Reason("The returned resp is empty").Err()
	}
	res.Name = ufsUtil.RemovePrefix(res.Name)
	fmt.Println("The kvm before delete/update:")
	PrintProtoJSON(res, !NoEmitMode(false))
	return nil
}

// PrintExistingRPM prints the old rpm in update/delete operations
func PrintExistingRPM(ctx context.Context, ic ufsAPI.FleetClient, name string) error {
	res, err := ic.GetRPM(ctx, &ufsAPI.GetRPMRequest{
		Name: ufsUtil.AddPrefix(ufsUtil.RPMCollection, name),
	})
	if err != nil {
		return errors.Annotate(err, "Failed to get rpm").Err()
	}
	if res == nil {
		return errors.Reason("The returned resp is empty").Err()
	}
	res.Name = ufsUtil.RemovePrefix(res.Name)
	fmt.Println("The rpm before delete/update:")
	PrintProtoJSON(res, !NoEmitMode(false))
	return nil
}

// PrintExistingSwitch prints the old switch in update/delete operations
func PrintExistingSwitch(ctx context.Context, ic ufsAPI.FleetClient, name string) error {
	res, err := ic.GetSwitch(ctx, &ufsAPI.GetSwitchRequest{
		Name: ufsUtil.AddPrefix(ufsUtil.SwitchCollection, name),
	})
	if err != nil {
		return errors.Annotate(err, "Failed to get switch").Err()
	}
	if res == nil {
		return errors.Reason("The returned resp is empty").Err()
	}
	res.Name = ufsUtil.RemovePrefix(res.Name)
	fmt.Println("The switch before delete/update:")
	PrintProtoJSON(res, !NoEmitMode(false))
	return nil
}

// PrintExistingHost prints the old host in update/delete operations
func PrintExistingHost(ctx context.Context, ic ufsAPI.FleetClient, name string) error {
	res, err := ic.GetMachineLSE(ctx, &ufsAPI.GetMachineLSERequest{
		Name: ufsUtil.AddPrefix(ufsUtil.MachineLSECollection, name),
	})
	if err != nil {
		return errors.Annotate(err, "Failed to get host").Err()
	}
	if res == nil {
		return errors.Reason("The returned resp is empty").Err()
	}
	res.Name = ufsUtil.RemovePrefix(res.Name)
	fmt.Println("The host before delete/update:")
	PrintProtoJSON(res, !NoEmitMode(false))
	printDHCP(ctx, ic, name)
	return nil
}

// PrintExistingDUT prints the old host in update/delete operations
func PrintExistingDUT(ctx context.Context, ic ufsAPI.FleetClient, name string) error {
	res, err := ic.GetMachineLSE(ctx, &ufsAPI.GetMachineLSERequest{
		Name: ufsUtil.AddPrefix(ufsUtil.MachineLSECollection, name),
	})
	if err != nil {
		return errors.Annotate(err, "Failed to get DUT").Err()
	}
	if res == nil {
		return errors.Reason("The returned resp is empty").Err()
	}
	res.Name = ufsUtil.RemovePrefix(res.Name)
	fmt.Println("The DUT before delete/update:")
	PrintProtoJSON(res, !NoEmitMode(false))
	return nil
}

// PrintExistingVM prints the old vm in update/delete operations
func PrintExistingVM(ctx context.Context, ic ufsAPI.FleetClient, name string) error {
	res, err := ic.GetVM(ctx, &ufsAPI.GetVMRequest{
		Name: ufsUtil.AddPrefix(ufsUtil.VMCollection, name),
	})
	if err != nil {
		return errors.Annotate(err, "Failed to get vm").Err()
	}
	if res == nil {
		return errors.Reason("The returned resp is empty").Err()
	}
	res.Name = ufsUtil.RemovePrefix(res.Name)
	fmt.Println("The vm before delete/update:")
	PrintProtoJSON(res, !NoEmitMode(false))
	printDHCP(ctx, ic, name)
	return nil
}

func printDHCP(ctx context.Context, ic ufsAPI.FleetClient, hostname string) {
	dhcp, err := ic.GetDHCPConfig(ctx, &ufsAPI.GetDHCPConfigRequest{
		Hostname: hostname,
	})
	if err == nil && dhcp != nil {
		fmt.Println("Associated dhcp:")
		PrintProtoJSON(dhcp, false)
	} else {
		fmt.Printf("Associated dhcp: Not found for host %s (err: %s)\n", hostname, err.Error())
	}
}

// PrintExistingVlan prints the old vlan update/delete operations
func PrintExistingVlan(ctx context.Context, ic ufsAPI.FleetClient, name string) error {
	res, err := ic.GetVlan(ctx, &ufsAPI.GetVlanRequest{
		Name: ufsUtil.AddPrefix(ufsUtil.VlanCollection, name),
	})
	if err != nil {
		return errors.Annotate(err, "Failed to get vlan").Err()
	}
	if res == nil {
		return errors.Reason("The returned resp is empty").Err()
	}
	res.Name = ufsUtil.RemovePrefix(res.Name)
	fmt.Println("The vlan before delete/update:")
	PrintProtoJSON(res, !NoEmitMode(false))
	return nil
}

// PrintExistingPlatform prints the old platform in update/delete operations
func PrintExistingPlatform(ctx context.Context, ic ufsAPI.FleetClient, name string) error {
	res, err := ic.GetChromePlatform(ctx, &ufsAPI.GetChromePlatformRequest{
		Name: ufsUtil.AddPrefix(ufsUtil.ChromePlatformCollection, name),
	})
	if err != nil {
		return errors.Annotate(err, "Failed to get chrome platform").Err()
	}
	if res == nil {
		return errors.Reason("The returned resp is empty").Err()
	}
	res.Name = ufsUtil.RemovePrefix(res.Name)
	fmt.Println("The chrome platform before delete/update:")
	PrintProtoJSON(res, !NoEmitMode(false))
	return nil
}

// PrintExistingMachinePrototype prints the old machine prototype in update/delete operations
func PrintExistingMachinePrototype(ctx context.Context, ic ufsAPI.FleetClient, name string) error {
	res, err := ic.GetMachineLSEPrototype(ctx, &ufsAPI.GetMachineLSEPrototypeRequest{
		Name: ufsUtil.AddPrefix(ufsUtil.MachineLSEPrototypeCollection, name),
	})
	if err != nil {
		return errors.Annotate(err, "Failed to get machine lse prototype").Err()
	}
	if res == nil {
		return errors.Reason("The returned resp is empty").Err()
	}
	res.Name = ufsUtil.RemovePrefix(res.Name)
	fmt.Println("The machine lse prototype before delete/update:")
	PrintProtoJSON(res, !NoEmitMode(false))
	return nil
}

// PrintExistingRackPrototype prints the old rack prototype in update/delete operations
func PrintExistingRackPrototype(ctx context.Context, ic ufsAPI.FleetClient, name string) error {
	res, err := ic.GetRackLSEPrototype(ctx, &ufsAPI.GetRackLSEPrototypeRequest{
		Name: ufsUtil.AddPrefix(ufsUtil.RackLSEPrototypeCollection, name),
	})
	if err != nil {
		return errors.Annotate(err, "Failed to get rack lse prototype").Err()
	}
	if res == nil {
		return errors.Reason("The returned resp is empty").Err()
	}
	res.Name = ufsUtil.RemovePrefix(res.Name)
	fmt.Println("The rack lse prototype before delete/update:")
	PrintProtoJSON(res, !NoEmitMode(false))
	return nil
}

// PrintExistingCachingService prints the old CachingService in update/delete operations
func PrintExistingCachingService(ctx context.Context, ic ufsAPI.FleetClient, name string) error {
	res, err := ic.GetCachingService(ctx, &ufsAPI.GetCachingServiceRequest{
		Name: ufsUtil.AddPrefix(ufsUtil.CachingServiceCollection, name),
	})
	if err != nil {
		return errors.Annotate(err, "Failed to get CachingService").Err()
	}
	if res == nil {
		return errors.Reason("The returned resp is empty").Err()
	}
	res.Name = ufsUtil.RemovePrefix(res.Name)
	fmt.Println("The CachingService before delete/update:")
	PrintProtoJSON(res, !NoEmitMode(false))
	return nil
}

// PrintExistingSchedulingUnit prints the old SchedulingUnit in update/delete operations
func PrintExistingSchedulingUnit(ctx context.Context, ic ufsAPI.FleetClient, name string) error {
	res, err := ic.GetSchedulingUnit(ctx, &ufsAPI.GetSchedulingUnitRequest{
		Name: ufsUtil.AddPrefix(ufsUtil.SchedulingUnitCollection, name),
	})
	if err != nil {
		return errors.Annotate(err, "Failed to get SchedulingUnit").Err()
	}
	if res == nil {
		return errors.Reason("The returned resp is empty").Err()
	}
	res.Name = ufsUtil.RemovePrefix(res.Name)
	fmt.Println("The SchedulingUnit before delete/update:")
	PrintProtoJSON(res, !NoEmitMode(false))
	return nil
}
