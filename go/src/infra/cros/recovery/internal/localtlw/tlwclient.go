// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package localtlw provides local implementation of TLW Access.
package localtlw

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"strconv"

	"go.chromium.org/chromiumos/config/go/api/test/xmlrpc"
	"go.chromium.org/luci/common/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	fleet "infra/appengine/crosskylabadmin/api/fleet/v1"
	"infra/cros/recovery/internal/localtlw/dutinfo"
	"infra/cros/recovery/internal/localtlw/ssh"
	"infra/cros/recovery/tlw"
	"infra/libs/sshpool"
	ufspb "infra/unifiedfleet/api/v1/models"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	ufsUtil "infra/unifiedfleet/app/util"
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
	sshPool   *sshpool.Pool
}

// New build new local TLW Access instance.
func New(ufs UFSClient, csac CSAClient) (tlw.Access, error) {
	c := &tlwClient{
		ufsClient: ufs,
		csaClient: csac,
		sshPool:   sshpool.New(ssh.SSHConfig()),
	}
	return c, nil
}

// Close closes all used resources.
func (c *tlwClient) Close() error {
	if c != nil {
		return c.sshPool.Close()
	}
	return nil
}

// Ping performs ping by resource name.
func (c *tlwClient) Ping(ctx context.Context, resourceName string, count int) error {
	return ping(resourceName, count)
}

// Run executes command on device by SSH related to resource name.
func (c *tlwClient) Run(ctx context.Context, resourceName, command string) *tlw.RunResult {
	host := net.JoinHostPort(resourceName, strconv.Itoa(22))
	return ssh.Run(c.sshPool, host, command)
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
	if name == "" {
		return nil, errors.Reason("list resources: unit name is expected").Err()
	}
	dd, err := c.ufsClient.GetChromeOSDeviceData(ctx, &ufsAPI.GetChromeOSDeviceDataRequest{
		Hostname: name,
	})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			log.Printf("list resources %q: record not found", name)
		} else {
			return nil, errors.Reason("list resources %q", name).Err()
		}
	} else if dd.GetLabConfig() == nil {
		return nil, errors.Reason("list resources %q: device data is empty", name).Err()
	} else {
		return []string{dd.GetLabConfig().GetName()}, nil
	}
	suName := ufsUtil.AddPrefix(ufsUtil.SchedulingUnitCollection, name)
	log.Printf("list resources %q: trying to find scheduling unit by name %q.", name, suName)
	su, err := c.ufsClient.GetSchedulingUnit(ctx, &ufsAPI.GetSchedulingUnitRequest{
		Name: suName,
	})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, errors.Annotate(err, "list resources %q: record not found", name).Err()
		}
		return nil, errors.Annotate(err, "list resources %q", name).Err()
	}
	var resourceNames []string
	for _, hostname := range su.GetMachineLSEs() {
		resourceNames = append(resourceNames, hostname)
	}
	return resourceNames, nil
}

// GetDut provides DUT info per requested resource name from inventory.
func (c *tlwClient) GetDut(ctx context.Context, name string) (*tlw.Dut, error) {
	req := &ufsAPI.GetChromeOSDeviceDataRequest{Hostname: name}
	dd, err := c.ufsClient.GetChromeOSDeviceData(ctx, req)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, errors.Reason("get DUT %q: record not found", name).Err()
		}
		return nil, errors.Annotate(err, "get DUT %q", name).Err()
	} else if dd.GetLabConfig() == nil {
		return nil, errors.Reason("get DUT %q: received empty data", name).Err()
	}
	d, err := dutinfo.ConvertDut(dd)
	printAsJSON("DUT", d)
	return d, errors.Annotate(err, "get DUT %q", name).Err()
}

// UpdateDut updates DUT info into inventory.
func (c *tlwClient) UpdateDut(ctx context.Context, dut *tlw.Dut) error {
	return status.Errorf(codes.Unimplemented, "not implemented")
}

// printAsJSON prints JSON representation of the struct.
func printAsJSON(name string, d interface{}) {
	if d != nil {
		s, _ := json.MarshalIndent(d, "", "\t")
		log.Printf("%s: %s", name, s)
	}
}
