// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package inventoryv2

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/grpc/prpc"

	api "infra/appengine/cros/lab_inventory/api/v1"
	"infra/cmd/skylab/internal/site"
)

func newInventoryClient(hc *http.Client, e site.Environment) api.InventoryClient {
	return api.NewInventoryPRPCClient(&prpc.Client{
		C:       hc,
		Host:    e.InventoryService,
		Options: site.DefaultPRPCOptions,
	})
}

// GetDutInfo gets the dut information from inventory v2 service.
func GetDutInfo(ctx context.Context, hc *http.Client, e site.Environment, hostname string) (*api.ExtendedDeviceData, error) {
	rsp, err := newInventoryClient(hc, e).GetCrosDevices(ctx, &api.GetCrosDevicesRequest{
		Ids: []*api.DeviceID{
			{
				Id: &api.DeviceID_Hostname{Hostname: hostname},
			},
		},
	})
	if err != nil {
		return nil, errors.Annotate(err, "[v2] get dutinfo for %s", hostname).Err()
	}
	if len(rsp.FailedDevices) > 0 {
		result := rsp.FailedDevices[0]
		return nil, errors.Reason("[v2] failed to get device %s: %s", result.Hostname, result.ErrorMsg).Err()
	}
	return rsp.Data[0], nil
}

// RemoveDevices removes devices from drones, and optionally removes them from
// the inventory.
// TODO (guocb) implement removing from drone only.
func RemoveDevices(ctx context.Context, hc *http.Client, e site.Environment, hostnames []string, deleteFromInventoryAlso bool) (bool, error) {
	var devIds []*api.DeviceID
	for _, h := range hostnames {
		devIds = append(devIds, &api.DeviceID{Id: &api.DeviceID_Hostname{Hostname: h}})
	}
	rsp, err := newInventoryClient(hc, e).DeleteCrosDevices(ctx, &api.DeleteCrosDevicesRequest{
		Ids: devIds,
	})
	if err != nil {
		return false, errors.Annotate(err, "[v2] remove devices for %s ...", hostnames[0]).Err()
	}
	if len(rsp.FailedDevices) > 0 {
		var reasons []string
		for _, d := range rsp.FailedDevices {
			reasons = append(reasons, fmt.Sprintf("%s:%s", d.Hostname, d.ErrorMsg))
		}
		return false, errors.Reason("[v2] failed to remove device: %s", strings.Join(reasons, ", ")).Err()
	}
	return len(rsp.RemovedDevices) > 0, nil

}
