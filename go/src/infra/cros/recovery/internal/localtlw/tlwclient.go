// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package localtlw provides local implementation of TLW Access.
package localtlw

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"go.chromium.org/chromiumos/config/go/api/test/xmlrpc"
	"go.chromium.org/luci/common/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	fleet "infra/appengine/crosskylabadmin/api/fleet/v1"
	"infra/cros/recovery/internal/localtlw/dutinfo"

	tlwio "infra/cros/recovery/internal/localtlw/io"
	"infra/cros/recovery/internal/localtlw/servod"
	"infra/cros/recovery/internal/localtlw/ssh"
	"infra/cros/recovery/internal/log"
	"infra/cros/recovery/tlw"
	"infra/libs/sshpool"
	ufspb "infra/unifiedfleet/api/v1/models"
	ufslab "infra/unifiedfleet/api/v1/models/chromeos/lab"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	ufsUtil "infra/unifiedfleet/app/util"
)

const (
	// gsCrosImageBucket is the base URL for the Google Storage bucket for
	// ChromeOS image archives.
	gsCrosImageBucket = "gs://chromeos-image-archive"
	// tlwPort is default port used to run TLW on the drones.
	tlwPort = 7151
)

// UFSClient is a client that knows how to work with UFS RPC methods.
type UFSClient interface {
	// GetSchedulingUnit retrieves the details of the SchedulingUnit.
	GetSchedulingUnit(ctx context.Context, req *ufsAPI.GetSchedulingUnitRequest, opts ...grpc.CallOption) (rsp *ufspb.SchedulingUnit, err error)
	// GetChromeOSDeviceData retrieves requested Chrome OS device data from the UFS and inventoryV2.
	GetChromeOSDeviceData(ctx context.Context, req *ufsAPI.GetChromeOSDeviceDataRequest, opts ...grpc.CallOption) (rsp *ufspb.ChromeOSDeviceData, err error)
	// UpdateDutState updates the state config for a DUT
	UpdateDutState(ctx context.Context, in *ufsAPI.UpdateDutStateRequest, opts ...grpc.CallOption) (*ufslab.DutState, error)
}

// CSAClient is a client that knows how to respond to the GetStableVersion RPC call.
type CSAClient interface {
	GetStableVersion(ctx context.Context, in *fleet.GetStableVersionRequest, opts ...grpc.CallOption) (*fleet.GetStableVersionResponse, error)
}

// tlwClient holds data and represents the local implementation of TLW Access interface.
type tlwClient struct {
	csaClient  CSAClient
	ufsClient  UFSClient
	sshPool    *sshpool.Pool
	servodPool *servod.Pool
	// Cache received devices from inventory
	devices map[string]*ufspb.ChromeOSDeviceData
}

// New build new local TLW Access instance.
func New(ufs UFSClient, csac CSAClient) (tlw.Access, error) {
	c := &tlwClient{
		ufsClient:  ufs,
		csaClient:  csac,
		sshPool:    sshpool.New(ssh.SSHConfig()),
		servodPool: servod.NewPool(),
		devices:    make(map[string]*ufspb.ChromeOSDeviceData),
	}
	return c, nil
}

// Close closes all used resources.
func (c *tlwClient) Close() error {
	if err := c.sshPool.Close(); err != nil {
		return errors.Annotate(err, "tlw client").Err()
	}
	return c.servodPool.Close()
}

// Ping performs ping by resource name.
func (c *tlwClient) Ping(ctx context.Context, resourceName string, count int) error {
	return ping(resourceName, count)
}

// Run executes command on device by SSH related to resource name.
func (c *tlwClient) Run(ctx context.Context, resourceName, command string) *tlw.RunResult {
	host := net.JoinHostPort(resourceName, strconv.Itoa(22))
	return ssh.Run(ctx, c.sshPool, host, command)
}

// InitServod initiates servod daemon on servo-host.
func (c *tlwClient) InitServod(ctx context.Context, req *tlw.InitServodRequest) error {
	dd, err := c.getDevice(ctx, req.Resource)
	if err != nil {
		return errors.Annotate(err, "init servod %q", req.Resource).Err()
	}
	dut := dd.GetLabConfig().GetChromeosMachineLse().GetDeviceLse().GetDut()
	if dut == nil {
		return errors.Reason("init servod %q: dut is not found", req.Resource).Err()
	}
	servo := dut.GetPeripherals().GetServo()
	if servo == nil {
		return errors.Reason("init servod %q: servo is not found", req.Resource).Err()
	}
	s, err := c.servodPool.Get(
		net.JoinHostPort(servo.GetServoHostname(), strconv.Itoa(22)),
		servo.GetServoPort(),
		func() ([]string, error) {
			return dutinfo.GenerateServodParams(dd, req.Options)
		})
	if err != nil {
		return errors.Annotate(err, "init servod %q", req.Resource).Err()
	}
	if err := s.Prepare(ctx, c.sshPool); err != nil {
		return errors.Annotate(err, "init servod %q", req.Resource).Err()
	}
	return nil
}

// StopServod stops servod daemon on servo-host.
func (c *tlwClient) StopServod(ctx context.Context, resourceName string) error {
	dd, err := c.getDevice(ctx, resourceName)
	if err != nil {
		return errors.Annotate(err, "stop servod %q", resourceName).Err()
	}
	dut := dd.GetLabConfig().GetChromeosMachineLse().GetDeviceLse().GetDut()
	if dut == nil {
		return errors.Reason("stop servod %q: dut is not found", resourceName).Err()
	}
	servo := dut.GetPeripherals().GetServo()
	if servo == nil {
		log.Debug(ctx, "Stop servod %q: servo is not specified", resourceName)
		return nil
	}
	host := net.JoinHostPort(servo.GetServoHostname(), strconv.Itoa(22))
	s, err := c.servodPool.Get(host, servo.GetServoPort(), nil)
	if err != nil {
		return errors.Annotate(err, "stop servod %q", resourceName).Err()
	}
	if err := s.Stop(ctx, c.sshPool); err != nil {
		return errors.Annotate(err, "stop servod %q", resourceName).Err()
	}
	return nil
}

// CallServod executes a command on servod related to resource name.
// Commands will be run against servod on servo-host.
func (c *tlwClient) CallServod(ctx context.Context, req *tlw.CallServodRequest) *tlw.CallServodResponse {
	// Translator to convert error to response structure.
	fail := func(err error) *tlw.CallServodResponse {
		return &tlw.CallServodResponse{
			Value: &xmlrpc.Value{
				ScalarOneof: &xmlrpc.Value_String_{
					String_: fmt.Sprintf("call servod %q: %s", req.Resource, err),
				},
			},
			Fault: true,
		}
	}
	dd, err := c.getDevice(ctx, req.Resource)
	if err != nil {
		return fail(err)
	}
	dut := dd.GetLabConfig().GetChromeosMachineLse().GetDeviceLse().GetDut()
	if dut == nil {
		log.Debug(ctx, "Call servod %q: dut is not found", req.Resource)
		return nil
	}
	servo := dut.GetPeripherals().GetServo()
	if servo == nil {
		log.Debug(ctx, "Call servod %q: servo is not specified", req.Resource)
		return nil
	}
	host := net.JoinHostPort(servo.GetServoHostname(), strconv.Itoa(22))
	s, err := c.servodPool.Get(
		host, servo.GetServoPort(), func() ([]string, error) {
			return dutinfo.GenerateServodParams(dd, req.Options)
		})
	if err != nil {
		return fail(err)
	}
	res, err := s.Call(ctx, c.sshPool, req)
	if err != nil {
		return fail(err)
	}
	return res
}

// CopyFileTo copies file to destination device from local.
func (c *tlwClient) CopyFileTo(ctx context.Context, req *tlw.CopyRequest) error {
	return status.Errorf(codes.Unimplemented, "not implemented")
}

// CopyFileFrom copies file from remote device to local.
func (c *tlwClient) CopyFileFrom(ctx context.Context, req *tlw.CopyRequest) error {
	return tlwio.CopyFileFrom(ctx, c.sshPool, req)
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

// GetImageUrl provides URL to the image requested to load.
// URL will use to download image to USB-drive and provisioning.
func (c *tlwClient) GetImageUrl(ctx context.Context, resourceName, imageName string) (string, error) {
	addr := fmt.Sprintf("0.0.0.0:%d", tlwPort)
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		return "", errors.Annotate(err, "connect to background TLS").Err()
	}
	defer conn.Close()
	gsImagePath := fmt.Sprintf("%s/%s", gsCrosImageBucket, imageName)
	return CacheForDut(ctx, conn, gsImagePath, resourceName)
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
			log.Debug(ctx, "List resources %q: record not found.", name)
		} else {
			return nil, errors.Reason("list resources %q", name).Err()
		}
	} else if dd.GetLabConfig() == nil {
		return nil, errors.Reason("list resources %q: device data is empty", name).Err()
	} else {
		log.Debug(ctx, "List resources %q: cached received device.", name)
		c.devices[name] = dd
		return []string{dd.GetLabConfig().GetName()}, nil
	}
	suName := ufsUtil.AddPrefix(ufsUtil.SchedulingUnitCollection, name)
	log.Debug(ctx, "list resources %q: trying to find scheduling unit by name %q.", name, suName)
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
	dd, err := c.getDevice(ctx, name)
	if err != nil {
		return nil, errors.Annotate(err, "get DUT %q", name).Err()
	}
	dut, err := dutinfo.ConvertDut(dd)
	if err != nil {
		return nil, errors.Annotate(err, "get DUT %q", name).Err()
	}
	gv, err := c.getStableVersion(ctx, dut)
	if err != nil {
		log.Info(ctx, "Get DUT %q: failed to receive stable-version. Error: %s", name, err)
	} else {
		dut.StableVersion = gv
	}
	return dut, nil
}

// getDevice receives device from inventory.
func (c *tlwClient) getDevice(ctx context.Context, name string) (*ufspb.ChromeOSDeviceData, error) {
	if d, ok := c.devices[name]; ok {
		log.Debug(ctx, "Get device %q: received from cache.", name)
		return d, nil
	}
	req := &ufsAPI.GetChromeOSDeviceDataRequest{Hostname: name}
	dd, err := c.ufsClient.GetChromeOSDeviceData(ctx, req)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, errors.Reason("get device %q: record not found", name).Err()
		}
		return nil, errors.Annotate(err, "get device %q", name).Err()
	} else if dd.GetLabConfig() == nil {
		return nil, errors.Reason("get device %q: received empty data", name).Err()
	}
	c.devices[name] = dd
	log.Debug(ctx, "Get device %q: cached received device.", name)
	return dd, nil
}

// getStableVersion receives stable versions of device.
func (c *tlwClient) getStableVersion(ctx context.Context, dut *tlw.Dut) (*tlw.StableVersion, error) {
	req := &fleet.GetStableVersionRequest{Hostname: dut.Name}
	res, err := c.csaClient.GetStableVersion(ctx, req)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, errors.Reason("get stable-version %q: record not found", dut.Name).Err()
		}
		return nil, errors.Annotate(err, "get stable-version %q", dut.Name).Err()
	}
	if res.GetCrosVersion() == "" {
		return nil, errors.Reason("get stable-version %q: version is empty", dut.Name).Err()
	}
	return &tlw.StableVersion{
		CrosImage:           fmt.Sprintf("%s-release/%s", dut.Board, res.GetCrosVersion()),
		CrosFirmwareVersion: res.GetFirmwareVersion(),
		CrosFirmwareImage:   res.GetFaftVersion(),
	}, nil
}

// UpdateDut updates DUT info into inventory.
func (c *tlwClient) UpdateDut(ctx context.Context, dut *tlw.Dut) error {
	if dut == nil {
		return errors.Reason("update DUT: DUT is not provided").Err()
	}
	dd, err := c.getDevice(ctx, dut.Name)
	if err != nil {
		return errors.Annotate(err, "update DUT %q", dut.Name).Err()
	}
	dutID := dd.GetMachine().GetName()
	req, err := dutinfo.CreateUpdateDutRequest(dutID, dut)
	if err != nil {
		return errors.Annotate(err, "update DUT %q", dut.Name).Err()
	}
	log.Debug(ctx, "Update DUT: update request: %s", req)
	if _, err := c.ufsClient.UpdateDutState(ctx, req); err != nil {
		return errors.Annotate(err, "update DUT %q", dut.Name).Err()
	}
	delete(c.devices, dut.Name)
	return nil
}
