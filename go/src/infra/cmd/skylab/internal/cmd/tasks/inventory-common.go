// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package tasks

import (
	"context"
	"fmt"

	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/grpc/prpc"

	skycmdlib "infra/cmd/skylab/internal/cmd/cmdlib"
	"infra/cmd/skylab/internal/site"
	"infra/cmdsupport/cmdlib"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	ufsUtil "infra/unifiedfleet/app/util"
)

// getUFSClient produces an UFS client.
func getUFSClient(ctx context.Context, authFlags *authcli.Flags, e site.Environment) (ufsAPI.FleetClient, error) {
	hc, err := cmdlib.NewHTTPClient(ctx, authFlags)
	if err != nil {
		return nil, err
	}
	return ufsAPI.NewFleetPRPCClient(&prpc.Client{
		C:       hc,
		Host:    e.UFSService,
		Options: site.UFSPRPCOptions,
	}), nil
}

// getModelForHost contacts the UFS service and gets the model associated with
// a given hostname.
func getModelForHost(ctx context.Context, ic ufsAPI.FleetClient, host string) (string, error) {
	ctx = skycmdlib.SetupContext(ctx, ufsUtil.OSNamespace)
	lse, err := ic.GetMachineLSE(ctx, &ufsAPI.GetMachineLSERequest{
		Name: ufsUtil.AddPrefix(ufsUtil.MachineLSECollection, host),
	})
	if err != nil {
		return "", err
	}
	if len(lse.GetMachines()) == 0 {
		return "", errors.New(fmt.Sprintf("Invalid host %s, no associated asset/machine found", lse.Name))
	}
	machine, err := ic.GetMachine(ctx, &ufsAPI.GetMachineRequest{
		Name: ufsUtil.AddPrefix(ufsUtil.MachineCollection, lse.GetMachines()[0]),
	})
	if err != nil {
		return "", err
	}
	return machine.GetChromeosMachine().GetModel(), nil
}
