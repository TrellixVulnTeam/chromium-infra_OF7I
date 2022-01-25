// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package ufs

import (
	"context"
	"net/http"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/grpc/prpc"
	"go.chromium.org/luci/server/auth"
	"google.golang.org/grpc"

	"infra/appengine/crosskylabadmin/site"
	"infra/libs/skylab/common/heuristics"
	models "infra/unifiedfleet/api/v1/models"
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
		Options: site.DefaultPRPCOptions,
	}), nil
}

// GetPoolsClient expsoes the subset of the UFS client API needed by GetPools.
type GetPoolsClient interface {
	GetMachineLSE(ctx context.Context, in *ufsAPI.GetMachineLSERequest, opts ...grpc.CallOption) (*models.MachineLSE, error)
}

// GetPools gets the pools associated with a particular bot or dut.
// UFSClient may be nil.
func GetPools(ctx context.Context, ufsClient GetPoolsClient, name string) ([]string, error) {
	if ufsClient == nil {
		return nil, errors.Reason("get pools: ufsClient cannot be nil").Err()
	}
	name = heuristics.NormalizeBotNameToDeviceName(name)
	res, err := ufsClient.GetMachineLSE(ctx, &ufsAPI.GetMachineLSERequest{
		Name: ufsUtil.AddPrefix(ufsUtil.MachineLSECollection, name),
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
