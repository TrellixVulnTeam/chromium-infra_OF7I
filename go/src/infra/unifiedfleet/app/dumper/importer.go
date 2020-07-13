// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dumper

import (
	"context"

	api "infra/unifiedfleet/api/v1/rpc"
	frontend "infra/unifiedfleet/app/frontend"

	"go.chromium.org/luci/common/logging"
)

const machineDBHost = "machine-db.appspot.com"
const crosInventoryHost = "cros-lab-inventory.appspot.com"

func importCrimson(ctx context.Context) error {
	sv := &frontend.FleetServerImpl{}
	logging.Debugf(ctx, "Importing chrome platforms")
	respCP, err := sv.ImportChromePlatforms(ctx, &api.ImportChromePlatformsRequest{
		Source: &api.ImportChromePlatformsRequest_ConfigSource{
			ConfigSource: &api.ConfigSource{
				ConfigServiceName: frontend.DefaultMachineDBService,
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
				ConfigServiceName: frontend.DefaultMachineDBService,
				FileName:          "vlans.cfg",
			},
		},
	})
	if err != nil {
		logging.Debugf(ctx, "Fail to importing vlans: %s", string(respVlan.GetDetails()[0].GetValue()))
		return err
	}
	logging.Debugf(ctx, "Finish importing vlans: %#v", respVlan)

	logging.Debugf(ctx, "Reading datacenter configs")
	dcs, err := sv.ImportDatacenterConfigs(ctx)
	for _, dc := range dcs {
		if err := importDCHelper(ctx, sv, dc); err != nil {
			return err
		}
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

func importDCHelper(ctx context.Context, sv *frontend.FleetServerImpl, cfg string) error {
	logging.Debugf(ctx, "Importing datacenter %s", cfg)
	resp, err := sv.ImportDatacenters(ctx, &api.ImportDatacentersRequest{
		Source: &api.ImportDatacentersRequest_ConfigSource{
			ConfigSource: &api.ConfigSource{
				ConfigServiceName: frontend.DefaultMachineDBService,
				FileName:          cfg,
			},
		},
	})
	if err != nil {
		logging.Debugf(ctx, "Fail to importing datacenter: %s", string(resp.GetDetails()[0].GetValue()))
		return err
	}
	logging.Debugf(ctx, "Finish importing datacenters: %#v", resp)
	return nil
}

func importCrosInventory(ctx context.Context) error {
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
				Host: crosInventoryHost,
			},
		},
	})
	if err == nil {
		logging.Debugf(ctx, "Finish importing CrOS network configs")
	}
	return err
}
