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

const stopImport = true

func importCrimson(ctx context.Context) (err error) {
	defer func() {
		dumpCrimsonTick.Add(ctx, 1, err == nil)
	}()
	machineDBConfigService := config.Get(ctx).MachineDbConfigService
	if machineDBConfigService == "" {
		machineDBConfigService = frontend.DefaultMachineDBService
	}
	logging.Infof(ctx, "Querying config source %s", machineDBConfigService)
	machineDBHost := config.Get(ctx).GetMachineDbHost()
	if machineDBHost == "" {
		machineDBHost = fmt.Sprintf("%s.appspot.com", machineDBConfigService)
	}
	logging.Infof(ctx, "Querying host %s", machineDBHost)
	logging.Infof(ctx, "Comparing crimson with UFS before importing")
	if err := compareCrimson(ctx, machineDBHost); err != nil {
		logging.Warningf(ctx, "Fail to generate sync diff: %s", err.Error())
	}
	logging.Infof(ctx, "Finish exporting diff from crimson to UFS to Google Storage")

	if stopImport {
		logging.Infof(ctx, "Syncing from crimson is stopped")
		return nil
	}

	sv := &frontend.FleetServerImpl{}
	logging.Infof(ctx, "Importing chrome platforms")
	respCP, err := sv.ImportChromePlatforms(ctx, &api.ImportChromePlatformsRequest{
		Source: &api.ImportChromePlatformsRequest_ConfigSource{
			ConfigSource: &api.ConfigSource{
				ConfigServiceName: machineDBConfigService,
				FileName:          "platforms.cfg",
			},
		},
	})
	if err != nil {
		if len(respCP.GetDetails()) > 0 {
			logging.Warningf(ctx, "Fail to importing chrome platform: %s", string(respCP.GetDetails()[0].GetValue()))
		} else {
			logging.Warningf(ctx, "Fail to importing chrome platform: %s", err.Error())
		}
		return err
	}
	logging.Infof(ctx, "Finish importing chrome platforms: %#v", respCP)

	logging.Infof(ctx, "Importing os versions")
	respOS, err := sv.ImportOSVersions(ctx, &api.ImportOSVersionsRequest{
		Source: &api.ImportOSVersionsRequest_MachineDbSource{
			MachineDbSource: &api.MachineDBSource{
				Host: machineDBHost,
			},
		},
	})
	if err != nil {
		if len(respOS.GetDetails()) > 0 {
			logging.Warningf(ctx, "Fail to importing chrome platform: %s", string(respOS.GetDetails()[0].GetValue()))
		} else {
			logging.Warningf(ctx, "Fail to importing chrome platform: %s", err.Error())
		}
		return err
	}
	logging.Infof(ctx, "Finish importing os: %#v", respOS)

	logging.Infof(ctx, "Importing vlans")
	respVlan, err := sv.ImportVlans(ctx, &api.ImportVlansRequest{
		Source: &api.ImportVlansRequest_ConfigSource{
			ConfigSource: &api.ConfigSource{
				ConfigServiceName: machineDBConfigService,
				FileName:          "vlans.cfg",
			},
		},
	})
	if err != nil {
		if len(respVlan.GetDetails()) > 0 {
			logging.Warningf(ctx, "Fail to importing vlans: %s", string(respVlan.GetDetails()[0].GetValue()))
		} else {
			logging.Warningf(ctx, "Fail to importing vlans: %s", err.Error())
		}
		return err
	}
	logging.Infof(ctx, "Finish importing vlans: %#v", respVlan)

	logging.Infof(ctx, "Reading datacenters")
	respDC, err := sv.ImportDatacenters(ctx, &api.ImportDatacentersRequest{
		Source: &api.ImportDatacentersRequest_ConfigSource{
			ConfigSource: &api.ConfigSource{
				ConfigServiceName: machineDBConfigService,
				FileName:          "datacenters.cfg",
			},
		},
	})
	if err != nil {
		if len(respDC.GetDetails()) > 0 {
			logging.Warningf(ctx, "Fail to importing datacenters: %s", string(respDC.GetDetails()[0].GetValue()))
		} else {
			logging.Warningf(ctx, "Fail to importing datacenters: %s", err.Error())
		}
		return err
	}

	logging.Infof(ctx, "Importing machines")
	respMachine, err := sv.ImportMachines(ctx, &api.ImportMachinesRequest{
		Source: &api.ImportMachinesRequest_MachineDbSource{
			MachineDbSource: &api.MachineDBSource{
				Host: machineDBHost,
			},
		},
	})
	if err != nil {
		if len(respMachine.GetDetails()) > 0 {
			logging.Warningf(ctx, "Fail to importing machines: %s", string(respMachine.GetDetails()[0].GetValue()))
		} else {
			logging.Warningf(ctx, "Fail to importing machines: %s", err.Error())
		}
		return err
	}
	logging.Infof(ctx, "Finish importing machines: %#v", respMachine)

	logging.Infof(ctx, "Importing nics")
	respNic, err := sv.ImportNics(ctx, &api.ImportNicsRequest{
		Source: &api.ImportNicsRequest_MachineDbSource{
			MachineDbSource: &api.MachineDBSource{
				Host: machineDBHost,
			},
		},
	})
	if err != nil {
		if len(respNic.GetDetails()) > 0 {
			logging.Warningf(ctx, "Fail to importing nics: %s", string(respNic.GetDetails()[0].GetValue()))
		} else {
			logging.Warningf(ctx, "Fail to importing nics: %s", err.Error())
		}
		return err
	}
	logging.Infof(ctx, "Finish importing nics: %#v", respNic)

	logging.Infof(ctx, "Importing machine LSEs")
	respMLSE, err := sv.ImportMachineLSEs(ctx, &api.ImportMachineLSEsRequest{
		Source: &api.ImportMachineLSEsRequest_MachineDbSource{
			MachineDbSource: &api.MachineDBSource{
				Host: machineDBHost,
			},
		},
	})
	if err != nil {
		if len(respMLSE.GetDetails()) > 0 {
			logging.Warningf(ctx, "Fail to importing machine LSEs: %s", string(respMLSE.GetDetails()[0].GetValue()))
		} else {
			logging.Warningf(ctx, "Fail to importing machine LSEs: %s", err.Error())
		}
		return err
	}
	logging.Infof(ctx, "Finish importing machine LSEs: %#v", respMLSE)

	logging.Infof(ctx, "Importing states")
	respStates, err := sv.ImportStates(ctx, &api.ImportStatesRequest{
		Source: &api.ImportStatesRequest_MachineDbSource{
			MachineDbSource: &api.MachineDBSource{
				Host: machineDBHost,
			},
		},
	})
	if err != nil {
		if len(respStates.GetDetails()) > 0 {
			logging.Warningf(ctx, "Fail to importing states: %s", string(respStates.GetDetails()[0].GetValue()))
		} else {
			logging.Warningf(ctx, "Fail to importing states: %s", err.Error())
		}
		return err
	}
	logging.Infof(ctx, "Finish importing states: %#v", respStates)

	return nil
}

func importCrosInventory(ctx context.Context, crosInventoryHost string) error {
	logging.Infof(ctx, "Querying host %s", crosInventoryHost)
	sv := &frontend.FleetServerImpl{}
	logging.Infof(ctx, "Importing ChromeOS inventory")
	_, err := sv.ImportOSMachineLSEs(ctx, &api.ImportOSMachineLSEsRequest{
		Source: &api.ImportOSMachineLSEsRequest_MachineDbSource{
			MachineDbSource: &api.MachineDBSource{
				Host: crosInventoryHost,
			},
		},
	})
	logging.Infof(ctx, "Finshed Importing ChromeOS inventory(labconfig+dutstates)")
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
