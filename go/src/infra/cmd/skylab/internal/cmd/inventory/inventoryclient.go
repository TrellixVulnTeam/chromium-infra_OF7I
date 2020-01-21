// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package inventory

import (
	"context"
	"io"
	"net/http"

	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/grpc/prpc"

	invV2Api "infra/appengine/cros/lab_inventory/api/v1"
	invV1Api "infra/appengine/crosskylabadmin/api/fleet/v1"
	"infra/cmd/skylab/internal/cmd/cmdlib"
	"infra/cmd/skylab/internal/site"
	"infra/libs/skylab/inventory"
)

// Client defines the common interface for the inventory client used by
// various command line tools.
type Client interface {
	GetDutInfo(context.Context, string, bool) (*inventory.DeviceUnderTest, error)
	removeDUTs(context.Context, string, []string, cmdlib.RemovalReason, io.Writer) (bool, error)
	deleteDUTs(context.Context, []string, *authcli.Flags, io.Writer) (bool, error)
}

type inventoryClientV1 struct {
	ic invV1Api.InventoryClient
}
type inventoryClientV2 struct {
	ic invV2Api.InventoryClient
}

// NewInventoryClient creates a new instance of inventory client.
func NewInventoryClient(hc *http.Client, env site.Environment, version2 bool) Client {
	if version2 {
		return &inventoryClientV2{
			ic: invV2Api.NewInventoryPRPCClient(&prpc.Client{
				C:       hc,
				Host:    env.InventoryService,
				Options: site.DefaultPRPCOptions,
			}),
		}
	}
	return &inventoryClientV1{
		ic: invV1Api.NewInventoryPRPCClient(&prpc.Client{
			C:       hc,
			Host:    env.AdminService,
			Options: site.DefaultPRPCOptions,
		}),
	}
}
