// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package tlw

import (
	"context"

	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"

	"go.chromium.org/chromiumos/config/go/api/test/tls"
	"go.chromium.org/chromiumos/config/go/api/test/xmlrpc"
)

type tlwClient struct {
}

func NewTLW(ufs, csa string) Access {
	c := &tlwClient{}
	return c
}

// Ping the device related to resource name.
func (c *tlwClient) Ping(ctx context.Context, resourceName string) (bool, error) {
	// return Ping(resourceName)
	return true, nil
}

// Execute command on the device related to resource name.
func (c *tlwClient) Run(ctx context.Context, resourceName, command string) *RunResult {
	return &RunResult{
		Command:  command,
		ExitCode: 1,
		Stdout:   "",
		Stderr:   "Not implemented",
	}
}

// Execute command on servo related to resource name.
// Commands will be run against servod on servo-host.
func (c *tlwClient) CallServod(ctx context.Context, resourceName, command string) *tls.CallServoResponse {
	return &tls.CallServoResponse{
		Value: &xmlrpc.Value{
			ScalarOneof: &xmlrpc.Value_String_{
				String_: "Not Implemented",
			},
		},
		Fault: true,
	}
}

// Copy file to destination device from local.
func (c *tlwClient) CopyFileTo(ctx context.Context, req *CopyRequest) error {
	return status.Errorf(codes.Unimplemented, "Not implemented")
}

// Copy file from destination device to local.
func (c *tlwClient) CopyFileFrom(ctx context.Context, req *CopyRequest) error {
	return status.Errorf(codes.Unimplemented, "Not implemented")
}

// Copy directory to destination device from local, recursively.
func (c *tlwClient) CopyDirectoryTo(ctx context.Context, req *CopyRequest) error {
	return status.Errorf(codes.Unimplemented, "Not implemented")
}

// Copy directory from destination device to local, recursively.
func (c *tlwClient) CopyDirectoryFrom(ctx context.Context, req *CopyRequest) error {
	return status.Errorf(codes.Unimplemented, "Not implemented")
}

// Manage power supply for requested.
func (c *tlwClient) SetPowerSupply(ctx context.Context, req *SetPowerSupplyRequest) *SetPowerSupplyResponse {
	return &SetPowerSupplyResponse{
		Status: PowerSupplyResponseStatusError,
		// Error details
		Reason: "Not Implemented",
	}
}

// Provide list of resources names related to target unit.
func (c *tlwClient) ListResourcesForUnit(ctx context.Context, unitName string) ([]string, error) {
	return nil, status.Errorf(codes.Unimplemented, "Not implemented")
}

// Get DUT info per requested resource name from inventory.
func (c *tlwClient) GetDut(ctx context.Context, resourceName string) (*Dut, error) {
	return nil, status.Errorf(codes.Unimplemented, "Not implemented")
}

// Update DUT info into inventory.
func (c *tlwClient) UpdateDut(ctx context.Context, dut *Dut) error {
	return status.Errorf(codes.Unimplemented, "Not implemented")
}
