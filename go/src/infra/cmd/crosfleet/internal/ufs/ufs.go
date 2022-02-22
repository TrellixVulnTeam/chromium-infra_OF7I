// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package ufs

import (
	"context"
	"infra/cmd/crosfleet/internal/site"
	"infra/cmdsupport/cmdlib"

	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/grpc/prpc"
	"google.golang.org/grpc"

	models "infra/unifiedfleet/api/v1/models"
	ufsapi "infra/unifiedfleet/api/v1/rpc"
)

// Client exposes a deliberately chosen subset of the UFS functionality.
type Client interface {
	GetMachineLSE(context.Context, *ufsapi.GetMachineLSERequest, ...grpc.CallOption) (*models.MachineLSE, error)
	GetMachine(context.Context, *ufsapi.GetMachineRequest, ...grpc.CallOption) (*models.Machine, error)
	CheckFleetTestsPolicy(context.Context, *ufsapi.CheckFleetTestsPolicyRequest, ...grpc.CallOption) (*ufsapi.CheckFleetTestsPolicyResponse, error)
}

// ClientImpl is the concrete implementation of this client.
type clientImpl struct {
	client ufsapi.FleetClient
}

// CheckFleetTestsPolicy checks the fleet test policy for the given test parameters.
func (c *clientImpl) CheckFleetTestsPolicy(ctx context.Context, req *ufsapi.CheckFleetTestsPolicyRequest) (*ufsapi.CheckFleetTestsPolicyResponse, error) {
	return c.client.CheckFleetTestsPolicy(ctx, req)
}

// GetMachineLSE gets information about a DUT.
func (c *clientImpl) GetMachineLSE(ctx context.Context, req *ufsapi.GetMachineLSERequest) (*models.MachineLSE, error) {
	return c.client.GetMachineLSE(ctx, req)
}

// GetMachine retrieves the details of the machine
func (c *clientImpl) GetMachine(ctx context.Context, req *ufsapi.GetMachineRequest) (*models.Machine, error) {
	return c.client.GetMachine(ctx, req)
}

// newUFSClient returns a new client to interact with the Unified Fleet System.
func NewUFSClient(ctx context.Context, ufsService string, authFlags *authcli.Flags) (Client, error) {
	httpClient, err := cmdlib.NewHTTPClient(ctx, authFlags)
	if err != nil {
		return nil, err
	}
	return ufsapi.NewFleetPRPCClient(&prpc.Client{
		C:       httpClient,
		Host:    ufsService,
		Options: site.DefaultPRPCOptions,
	}), nil
}
