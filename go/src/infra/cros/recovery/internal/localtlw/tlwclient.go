// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package localtlw provides local implementation of TLW Access.
package localtlw

import (
	"context"

	"go.chromium.org/chromiumos/config/go/api/test/xmlrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	fleet "infra/appengine/crosskylabadmin/api/fleet/v1"
	"infra/cros/recovery/tlw"
	ufspb "infra/unifiedfleet/api/v1/models"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
)

// UFSClient is a client that knows how to work with UFS RPC methods.
type UFSClient interface {
	GetSchedulingUnit(ctx context.Context, req *ufsAPI.GetSchedulingUnitRequest, opts ...grpc.CallOption) (rsp *ufspb.SchedulingUnit, err error)
	GetChromeOSDeviceData(ctx context.Context, req *ufsAPI.GetChromeOSDeviceDataRequest, opts ...grpc.CallOption) (rsp *ufspb.ChromeOSDeviceData, err error)
}

// CSAClient is a client that knows how to respond to the GetStableVersion RPC call.
type CSAClient interface {
	GetStableVersion(ctx context.Context, in *fleet.GetStableVersionRequest, opts ...grpc.CallOption) (*fleet.GetStableVersionResponse, error)
}

// tlwClient holds data and represents the local implmentation of TLW Access interface.
type tlwClient struct {
	csaClient CSAClient
	ufsClient UFSClient
}

// New build new local TLW Access instance.
func New(ufs UFSClient, csac CSAClient) (tlw.Access, error) {
	c := &tlwClient{
		ufsClient: ufs,
		csaClient: csac,
	}
	return c, nil
}

// Close closes all used resources.
func (c *tlwClient) Close() error {
	return nil
}

// Ping performs ping by resource name.
func (c *tlwClient) Ping(ctx context.Context, resourceName string, count int) error {
	return status.Errorf(codes.Unimplemented, "not implemented")
}

// Run executes command on device by SSH related to resource name.
func (c *tlwClient) Run(ctx context.Context, resourceName, command string) *tlw.RunResult {
	return &tlw.RunResult{
		Command:  command,
		ExitCode: 1,
		Stderr:   "not implemented",
	}
}

// CallServod executes a command on servod related to resource name.
// Commands will be run against servod on servo-host.
func (c *tlwClient) CallServod(ctx context.Context, req *tlw.CallServoRequest) *tlw.CallServoResponse {
	return &tlw.CallServoResponse{
		Value: &xmlrpc.Value{
			ScalarOneof: &xmlrpc.Value_String_{
				String_: "not implemented",
			},
		},
		Fault: true,
	}
}

// CopyFileTo copies file to destination device from local.
func (c *tlwClient) CopyFileTo(ctx context.Context, req *tlw.CopyRequest) error {
	return status.Errorf(codes.Unimplemented, "not implemented")
}

// CopyFileFrom copies file from destination device to local.
func (c *tlwClient) CopyFileFrom(ctx context.Context, req *tlw.CopyRequest) error {
	return status.Errorf(codes.Unimplemented, "not implemented")
}

// CopyDirectoryTo copies directory to destination device from local, recursively.
func (c *tlwClient) CopyDirectoryTo(ctx context.Context, req *tlw.CopyRequest) error {
	return status.Errorf(codes.Unimplemented, "not implemented")
}

// CopyDirectoryFrom copies directory from destination device to local, recursively.
func (c *tlwClient) CopyDirectoryFrom(ctx context.Context, req *tlw.CopyRequest) error {
	return status.Errorf(codes.Unimplemented, "not implemented")
}

// SetPowerSupply manages power supply for requested.
func (c *tlwClient) SetPowerSupply(ctx context.Context, req *tlw.SetPowerSupplyRequest) *tlw.SetPowerSupplyResponse {
	return &tlw.SetPowerSupplyResponse{
		Status: tlw.PowerSupplyResponseStatusError,
		Reason: "not implemented",
	}
}

// ListResourcesForUnit provides list of resources names related to target unit.
func (c *tlwClient) ListResourcesForUnit(ctx context.Context, name string) ([]string, error) {
	return nil, status.Errorf(codes.Unimplemented, "not implemented")
}

// GetDut provides DUT info per requested resource name from inventory.
func (c *tlwClient) GetDut(ctx context.Context, name string) (*tlw.Dut, error) {
	return nil, status.Errorf(codes.Unimplemented, "not implemented")
}

// UpdateDut updates DUT info into inventory.
func (c *tlwClient) UpdateDut(ctx context.Context, dut *tlw.Dut) error {
	return status.Errorf(codes.Unimplemented, "not implemented")
}
