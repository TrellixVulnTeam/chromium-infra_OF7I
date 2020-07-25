// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package inventory

import (
	"context"
	"net/http"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/grpc/prpc"

	invV2Api "infra/appengine/cros/lab_inventory/api/v1"
	"infra/libs/skylab/inventory"
)

// InvHostName is the inventory service endpoint.
const InvHostName = "cros-lab-inventory.appspot.com"

// Client defines the common interface for the inventory client.
type Client interface {
	GetDutInfoFromHostname(context.Context, string) (*inventory.DeviceUnderTest, error)
}

type inventoryClientV2 struct {
	ic invV2Api.InventoryClient
}

// NewInventoryClient creates a new instance of inventory client.
func NewInventoryClient(hc *http.Client) Client {
	return &inventoryClientV2{
		ic: invV2Api.NewInventoryPRPCClient(&prpc.Client{
			C:    hc,
			Host: InvHostName,
		}),
	}
}

// GetDutInfo gets the dut information from inventory v2 service.
func (client *inventoryClientV2) GetDutInfoFromHostname(ctx context.Context, id string) (*inventory.DeviceUnderTest, error) {
	devID := &invV2Api.DeviceID{Id: &invV2Api.DeviceID_Hostname{Hostname: id}}
	rsp, err := client.ic.GetCrosDevices(ctx, &invV2Api.GetCrosDevicesRequest{
		Ids: []*invV2Api.DeviceID{devID},
	})
	if err != nil {
		return nil, errors.Annotate(err, "get dutinfo for %s", id).Err()
	}
	if len(rsp.FailedDevices) > 0 {
		result := rsp.FailedDevices[0]
		return nil, errors.Reason("get dutinfo for %s: %s", result.Hostname, result.ErrorMsg).Err()
	}
	// TODO: raise different error for len(rsp.Data) > 1.
	if len(rsp.Data) != 1 {
		return nil, errors.Reason("get dut info for %s: empty info", id).Err()
	}
	return invV2Api.AdaptToV1DutSpec(rsp.Data[0])
}
