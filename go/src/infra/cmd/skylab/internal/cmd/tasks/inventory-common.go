// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package tasks

import (
	"context"
	"go.chromium.org/luci/auth/client/authcli"
	"infra/cmd/skylab/internal/site"

	inv "infra/cmd/skylab/internal/inventory"
	"infra/cmdsupport/cmdlib"
)

// getInventoryClient produces an inventory client.
func getInventoryClient(ctx context.Context, authFlags *authcli.Flags, e site.Environment) (inv.Client, error) {
	hc, err := cmdlib.NewHTTPClient(ctx, authFlags)
	if err != nil {
		return nil, err
	}
	return inv.NewInventoryClient(hc, e), nil
}

// getModelForHost contacts the inventory v2 service and gets the model associated with
// a given hostname.
func getModelForHost(ctx context.Context, ic inv.Client, host string) (string, error) {
	dut, err := ic.GetDutInfo(ctx, host, true)
	if err != nil {
		return "", err
	}
	return dut.GetCommon().GetLabels().GetModel(), nil
}
