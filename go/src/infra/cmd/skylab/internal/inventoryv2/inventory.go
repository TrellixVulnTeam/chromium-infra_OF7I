// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package inventoryv2

import (
	"context"
	"net/http"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/grpc/prpc"

	api "infra/appengine/cros/lab_inventory/api/v1"
	"infra/cmd/skylab/internal/site"
)

// GetDutInfo gets the dut information from inventory v2 service.
func GetDutInfo(ctx context.Context, hc *http.Client, e site.Environment, hostname string) (*api.ExtendedDeviceData, error) {
	ic := api.NewInventoryPRPCClient(&prpc.Client{
		C:       hc,
		Host:    e.InventoryService,
		Options: site.DefaultPRPCOptions,
	})
	rsp, err := ic.GetCrosDevices(ctx, &api.GetCrosDevicesRequest{
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
