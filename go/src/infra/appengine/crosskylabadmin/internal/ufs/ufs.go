// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package ufs

import (
	"context"
	"fmt"
	"net/http"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/grpc/prpc"
	"go.chromium.org/luci/server/auth"

	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	ufsUtil "infra/unifiedfleet/app/util"
)

// NewUFS client creates a new UFS client when given a hostname.
// The hostname should generally be read from the config.
func NewUFSClient(ctx context.Context, hostname string) (ufsAPI.FleetClient, error) {
	t, err := auth.GetRPCTransport(ctx, auth.AsSelf)
	if err != nil {
		return nil, errors.Annotate(err, "failed to get RPC transport for host %s", hostname).Err()
	}
	hc := &http.Client{Transport: t}
	return ufsAPI.NewFleetPRPCClient(&prpc.Client{
		C:       hc,
		Host:    hostname,
		Options: nil,
	}), nil
}

// GetPools gets the pools associated with a particular bot.
// UFSClient may be nil.
func GetPools(ctx context.Context, ufsClient ufsAPI.FleetClient, botID string) ([]string, error) {
	if ufsClient == nil {
		return nil, fmt.Errorf("get pools: ufsClient cannot be nil")
	}
	res, err := ufsClient.GetMachineLSE(ctx, &ufsAPI.GetMachineLSERequest{
		Name: ufsUtil.AddPrefix(ufsUtil.MachineLSECollection, botID),
	})
	if err != nil {
		return nil, errors.Annotate(err, "get pools").Err()
	}
	if res.GetChromeosMachineLse().GetDeviceLse().GetDut() != nil {
		// We have a non-labstation DUT.
		return res.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPools(), nil
	}
	// We have a labstation DUT.
	return res.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetPools(), nil
}
