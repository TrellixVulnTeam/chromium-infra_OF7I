// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dumper

import (
	"context"
	"fmt"

	"go.chromium.org/luci/common/logging"

	api "infra/unifiedfleet/api/v1/rpc"
	"infra/unifiedfleet/app/config"
	frontend "infra/unifiedfleet/app/frontend"
)

func importCrimson(ctx context.Context) error {
	machineDBConfigService := config.Get(ctx).MachineDbConfigService
	if machineDBConfigService == "" {
		machineDBConfigService = frontend.DefaultMachineDBService
	}
	machineDBHost := fmt.Sprintf("%s.appspot.com", machineDBConfigService)
	logging.Debugf(ctx, "Querying host %s", machineDBHost)
	logging.Debugf(ctx, "Comparing crimson with UFS before importing")
	compareCrimson(ctx, machineDBHost)
	logging.Debugf(ctx, "Finish exporting diff from crimson to UFS to Google Storage")

	sv := &frontend.FleetServerImpl{}
	logging.Debugf(ctx, "Importing chrome platforms")
	respCP, err := sv.ImportChromePlatforms(ctx, &api.ImportChromePlatformsRequest{
		Source: &api.ImportChromePlatformsRequest_ConfigSource{
			ConfigSource: &api.ConfigSource{
				ConfigServiceName: machineDBConfigService,
				FileName:          "platforms.cfg",
			},
		},
	})
	if err != nil {
		logging.Debugf(ctx, "Fail to importing chrome platform: %s", string(respCP.GetDetails()[0].GetValue()))
		return err
	}
	logging.Debugf(ctx, "Finish importing chrome platforms: %#v", respCP)

	logging.Debugf(ctx, "Importing vlans")
	respVlan, err := sv.ImportVlans(ctx, &api.ImportVlansRequest{
		Source: &api.ImportVlansRequest_ConfigSource{
			ConfigSource: &api.ConfigSource{
				ConfigServiceName: machineDBConfigService,
				FileName:          "vlans.cfg",
			},
		},
	})
	if err != nil {
		logging.Debugf(ctx, "Fail to importing vlans: %s", string(respVlan.GetDetails()[0].GetValue()))
		return err
	}
	logging.Debugf(ctx, "Finish importing vlans: %#v", respVlan)

	logging.Debugf(ctx, "Reading datacenters")
	respDC, err := sv.ImportDatacenters(ctx, &api.ImportDatacentersRequest{
		Source: &api.ImportDatacentersRequest_ConfigSource{
			ConfigSource: &api.ConfigSource{
				ConfigServiceName: machineDBConfigService,
				FileName:          "datacenters.cfg",
			},
		},
	})
	if err != nil {
		logging.Debugf(ctx, "Fail to importing datacenters: %s", string(respDC.GetDetails()[0].GetValue()))
		return err
	}

	logging.Debugf(ctx, "Importing machines")
	respMachine, err := sv.ImportMachines(ctx, &api.ImportMachinesRequest{
		Source: &api.ImportMachinesRequest_MachineDbSource{
			MachineDbSource: &api.MachineDBSource{
				Host: machineDBHost,
			},
		},
	})
	if err != nil {
		logging.Debugf(ctx, "Fail to importing machines: %s", string(respMachine.GetDetails()[0].GetValue()))
		return err
	}
	logging.Debugf(ctx, "Finish importing machines: %#v", respMachine)

	logging.Debugf(ctx, "Importing nics")
	respNic, err := sv.ImportNics(ctx, &api.ImportNicsRequest{
		Source: &api.ImportNicsRequest_MachineDbSource{
			MachineDbSource: &api.MachineDBSource{
				Host: machineDBHost,
			},
		},
	})
	if err != nil {
		logging.Debugf(ctx, "Fail to importing nics: %s", string(respNic.GetDetails()[0].GetValue()))
		return err
	}
	logging.Debugf(ctx, "Finish importing nics: %#v", respNic)

	logging.Debugf(ctx, "Importing machine LSEs")
	respMLSE, err := sv.ImportMachineLSEs(ctx, &api.ImportMachineLSEsRequest{
		Source: &api.ImportMachineLSEsRequest_MachineDbSource{
			MachineDbSource: &api.MachineDBSource{
				Host: machineDBHost,
			},
		},
	})
	if err != nil {
		logging.Debugf(ctx, "Fail to importing machine LSEs: %s", string(respMLSE.GetDetails()[0].GetValue()))
		return err
	}
	logging.Debugf(ctx, "Finish importing machine LSEs: %#v", respMLSE)

	logging.Debugf(ctx, "Importing states")
	respStates, err := sv.ImportStates(ctx, &api.ImportStatesRequest{
		Source: &api.ImportStatesRequest_MachineDbSource{
			MachineDbSource: &api.MachineDBSource{
				Host: machineDBHost,
			},
		},
	})
	if err != nil {
		logging.Debugf(ctx, "Fail to importing states: %s", string(respStates.GetDetails()[0].GetValue()))
		return err
	}
	logging.Debugf(ctx, "Finish importing states: %#v", respStates)

	return nil
}

func importCrosInventory(ctx context.Context) error {
	crosInventoryHost := config.Get(ctx).CrosInventoryHost
	if crosInventoryHost == "" {
		crosInventoryHost = "cros-lab-inventory.appspot.com"
	}
	logging.Debugf(ctx, "Querying host %s", crosInventoryHost)
	sv := &frontend.FleetServerImpl{}
	logging.Debugf(ctx, "Importing ChromeOS inventory")
	_, err := sv.ImportOSMachineLSEs(ctx, &api.ImportOSMachineLSEsRequest{
		Source: &api.ImportOSMachineLSEsRequest_MachineDbSource{
			MachineDbSource: &api.MachineDBSource{
				Host: crosInventoryHost,
			},
		},
	})
	return err
}

func importCrosNetwork(ctx context.Context) error {
	sv := &frontend.FleetServerImpl{}
	logging.Debugf(ctx, "Importing ChromeOS networks")
	_, err := sv.ImportOSVlans(ctx, &api.ImportOSVlansRequest{
		Source: &api.ImportOSVlansRequest_MachineDbSource{
			MachineDbSource: &api.MachineDBSource{
				Host: "",
			},
		},
	})
	if err == nil {
		logging.Debugf(ctx, "Finish importing CrOS network configs")
	}
	return err
}
