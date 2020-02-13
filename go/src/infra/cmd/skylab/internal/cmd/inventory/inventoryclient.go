// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package inventory

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"

	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/grpc/prpc"

	invV2Api "infra/appengine/cros/lab_inventory/api/v1"
	invV1Api "infra/appengine/crosskylabadmin/api/fleet/v1"
	"infra/cmd/skylab/internal/cmd/cmdlib"
	skycmdlib "infra/cmd/skylab/internal/cmd/cmdlib"
	"infra/cmd/skylab/internal/site"
	"infra/libs/skylab/inventory"
)

// AddSwitchInventoryFlag add flag "-v1" or "-v2" to the subcommands according
// to current setting in site.go.
func AddSwitchInventoryFlag(theFlag *bool, flags flag.FlagSet, envFlags skycmdlib.EnvFlags) {
	backupInventory := "v1"
	if envFlags.Env().DefaultInventory == "v1" {
		backupInventory = "v2"
	}
	flags.BoolVar(theFlag, backupInventory, false, fmt.Sprintf("[INTERNAL ONLY] Use ChromeOS Lab inventory %s service.", backupInventory))
}

// Client defines the common interface for the inventory client used by
// various command line tools.
type Client interface {
	GetDutInfo(context.Context, string, bool, bool) (*inventory.DeviceUnderTest, error)
	removeDUTs(context.Context, string, []string, cmdlib.RemovalReason, io.Writer) (bool, error)
	deleteDUTs(context.Context, []string, *authcli.Flags, io.Writer) (bool, error)
	batchUpdateDUTs(context.Context, *invV1Api.BatchUpdateDutsRequest, io.Writer) error
}

type inventoryClientV1 struct {
	ic invV1Api.InventoryClient
}
type inventoryClientV2 struct {
	ic invV2Api.InventoryClient
}

// NewInventoryClient creates a new instance of inventory client.
func NewInventoryClient(hc *http.Client, env site.Environment, useDefaultInventory bool) Client {
	if env.DefaultInventory == "v2" && useDefaultInventory || env.DefaultInventory == "v1" && !useDefaultInventory {
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
