// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package inventory

import (
	"context"
	"fmt"
	"net/http"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/grpc/prpc"
	"go.chromium.org/luci/server/auth"

	api "infra/appengine/cros/lab_inventory/api/v1"
	"infra/libs/skylab/inventory"
)

type invServiceClient struct {
	client api.InventoryClient
}

func newInvServiceClient(ctx context.Context, host string) (inventoryClient, error) {
	t, err := auth.GetRPCTransport(ctx, auth.AsCredentialsForwarder)
	if err != nil {
		return nil, errors.Annotate(err, "failed to get RPC transport to inventory service").Err()
	}
	ic := api.NewInventoryPRPCClient(&prpc.Client{
		C:    &http.Client{Transport: t},
		Host: host,
	})

	return &invServiceClient{client: ic}, nil
}

func (c *invServiceClient) logInfo(ctx context.Context, t string, s ...interface{}) {
	logging.Infof(ctx, fmt.Sprintf("[InventoryV2Client]: %s", t), s...)
}

func (c *invServiceClient) addManyDUTsToFleet(ctx context.Context, nds []*inventory.CommonDeviceSpecs, pickServoPort bool) (string, []*inventory.CommonDeviceSpecs, error) {
	// In case there's any panic happens in the new code.
	defer func() {
		if r := recover(); r != nil {
			logging.Errorf(ctx, "Recovered in addManyDUTsToFleet(%s)", r)
		}
	}()

	c.logInfo(ctx, "Access inventory service as user: %s", auth.CurrentUser(ctx))
	c.logInfo(ctx, "Adapter old data to inventory v2 proto")
	c.logInfo(ctx, "Call server RPC to add devices")
	c.logInfo(ctx, "Adapt the result back to old data format")
	return "No URL provided by inventory v2", nds, nil
}
func (c *invServiceClient) updateDUTSpecs(ctx context.Context, od, nd *inventory.CommonDeviceSpecs, pickServoPort bool) (string, error) {
	return "", nil
}
