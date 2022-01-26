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

// NewHTTPClient creates a new client specifically configured to talk to UFS correctly when run from
// CrOSSkyladAdmin dev or prod. It does not support other environments.
func NewHTTPClient(ctx context.Context) (*http.Client, error) {
	transport, err := auth.GetRPCTransport(ctx, auth.AsSelf)
	if err != nil {
		return nil, errors.Annotate(err, "failed to get RPC transport").Err()
	}
	return &http.Client{
		Transport: transport,
	}, nil
}

// Client exposes a deliberately chosen subset of the UFS functionality.
type Client interface {
	GetMachineLSE(context.Context, *ufsAPI.GetMachineLSERequest, ...grpc.CallOption) (*models.MachineLSE, error)
}

// ClientImpl is the concrete implementation of this client.
type clientImpl struct {
	client ufsAPI.FleetClient
}

// GetMachineLSE gets information about a DUT.
func (c *clientImpl) GetMachineLSE(ctx context.Context, req *ufsAPI.GetMachineLSERequest) (*models.MachineLSE, error) {
	return c.client.GetMachineLSE(ctx, req)
}

// NewClient creates a new UFS client when given a hostname and a http client.
// The hostname should generally be read from the config.
func NewClient(ctx context.Context, hc *http.Client, hostname string) (Client, error) {
	if hc == nil {
		return nil, errors.Reason("new ufs client: hc cannot be nil").Err()
	}
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
func GetPools(ctx context.Context, client Client, botID string) ([]string, error) {
	if client == nil {
		return nil, errors.Reason("get pools: client cannot be nil").Err()
	}
	res, err := client.GetMachineLSE(ctx, &ufsAPI.GetMachineLSERequest{
		Name: ufsUtil.AddPrefix(ufsUtil.MachineLSECollection, heuristics.NormalizeBotNameToDeviceName(botID)),
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
