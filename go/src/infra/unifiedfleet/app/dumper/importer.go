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
	return err
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
