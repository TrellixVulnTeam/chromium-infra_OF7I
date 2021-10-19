// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package ufs

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	labapi "go.chromium.org/chromiumos/config/go/test/lab/api"

	ufspb "infra/unifiedfleet/api/v1/models"
	ufsapi "infra/unifiedfleet/api/v1/rpc"
)

// GetDutTopology returns a DutTopology constructed from UFS.
// The returned error, if any, has gRPC status information.
func GetDutTopology(ctx context.Context, c ufsapi.FleetClient, id string) (*labapi.DutTopology, error) {
	// Assume the "scheduling unit" is a single DUT first.
	// If we can't find the DUT, try to look up the scheduling
	// unit for multiple DUT setups.
	dd, err := c.GetChromeOSDeviceData(ctx, &ufsapi.GetChromeOSDeviceDataRequest{Hostname: id})
	if err != nil {
		if s, ok := status.FromError(err); ok && s.Code() == codes.NotFound {
			return getSchedulingUnitDutTopology(ctx, c, id)
		} else {
			// Use the gRPC status from the UFS call.
			return nil, err
		}
	}
	lse := dd.GetLabConfig()
	d, err := makeDutProto(lse)
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "ID %q: %s", id, err)
	}
	dt := &labapi.DutTopology{
		Id:   &labapi.DutTopology_Id{Value: id},
		Duts: []*labapi.Dut{d},
	}
	return dt, nil
}

// getSchedulingUnitDutTopology returns a DutTopology constructed from
// UFS for a scheduling unit.
// The returned error, if any, has gRPC status information.
// You should use getDutTopology instead of this.
func getSchedulingUnitDutTopology(ctx context.Context, c ufsapi.FleetClient, id string) (*labapi.DutTopology, error) {
	resp, err := c.GetSchedulingUnit(ctx, &ufsapi.GetSchedulingUnitRequest{Name: id})
	if err != nil {
		// Use the gRPC status from the UFS call.
		return nil, err
	}
	dt := &labapi.DutTopology{}
	for _, name := range resp.GetMachineLSEs() {
		lse, err := c.GetMachineLSE(ctx, &ufsapi.GetMachineLSERequest{Name: name})
		if err != nil {
			return nil, status.Errorf(codes.FailedPrecondition, "%s", err)
		}
		d, err := makeDutProto(lse)
		if err != nil {
			return nil, status.Errorf(codes.FailedPrecondition, "ID %q: %s", id, err)
		}
		dt.Duts = append(dt.Duts, d)
	}
	return dt, nil
}

// makeDutProto makes a DutTopology Dut protobuf.
func makeDutProto(lse *ufspb.MachineLSE) (*labapi.Dut, error) {
	hostname := lse.GetHostname()
	if hostname == "" {
		return nil, errors.New("make dut proto: empty hostname")
	}
	return &labapi.Dut{
		Id: &labapi.Dut_Id{Value: hostname},
		DutType: &labapi.Dut_Chromeos{
			Chromeos: &labapi.Dut_ChromeOS{
				Ssh: &labapi.IpEndpoint{
					Address: hostname,
					Port:    22,
				},
			},
		},
	}, nil
}
