// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package labstation

import (
	"context"

	"github.com/golang/protobuf/proto"

	"infra/cmd/shivas/utils"
	"infra/cmd/shivas/utils/rename"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	ufsUtil "infra/unifiedfleet/app/util"
)

// RenameLabstationCmd rename labstation by given name.
var RenameLabstationCmd = rename.GenGenericRenameCmd("labstation", renameLabstation, printLabstation)

// renameLabstation calls the RPC that renames the given dut
func renameLabstation(ctx context.Context, ic ufsAPI.FleetClient, name, newName string) (interface{}, error) {
	// Set os namespace
	ctx = utils.SetupContext(ctx, ufsUtil.OSNamespace)
	// Change  this  API if you want to reuse the command somewhere else.
	return ic.RenameMachineLSE(ctx, &ufsAPI.RenameMachineLSERequest{
		Name:    ufsUtil.AddPrefix(ufsUtil.MachineLSECollection, name),
		NewName: ufsUtil.AddPrefix(ufsUtil.MachineLSECollection, newName),
	})
}

// printLabstation prints the result of the operation
func printLabstation(dut proto.Message) {
	utils.PrintProtoJSON(dut, !utils.NoEmitMode(false))
}
