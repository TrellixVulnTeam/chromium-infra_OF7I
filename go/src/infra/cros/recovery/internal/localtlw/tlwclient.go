// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package localtlw provides local implementation of TLW Access.
package localtlw

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.chromium.org/chromiumos/config/go/api/test/xmlrpc"
	"go.chromium.org/luci/common/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	fleet "infra/appengine/crosskylabadmin/api/fleet/v1"
	"infra/cros/dutstate"
	"infra/cros/recovery/docker"
	"infra/cros/recovery/internal/localtlw/dutinfo"
	tlwio "infra/cros/recovery/internal/localtlw/io"
	"infra/cros/recovery/internal/localtlw/localinfo"
	"infra/cros/recovery/internal/localtlw/localproxy"
	"infra/cros/recovery/internal/localtlw/rpm"
	"infra/cros/recovery/internal/localtlw/servod"
	"infra/cros/recovery/internal/localtlw/ssh"
	tlw_xmlrpc "infra/cros/recovery/internal/localtlw/xmlrpc"
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
	// tlsPort is default port used to run TLS on the drones.
	tlsPort = 7152
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

// hostType provides information which type of the host.
type hostType string

const (
	hostTypeCros      hostType = "cros-host"
	hostTypeServo     hostType = "servo-host"
	hostTypeBtPeer    hostType = "bluetooth-peer-host"
	hostTypeRouter    hostType = "router-host"
	hostTypeChameleon hostType = "chameleon-host"

	deafultBluetoothPeerServerPort = 9992
)

// tlwClient holds data and represents the local implementation of TLW Access interface.
type tlwClient struct {
	csaClient  CSAClient
	ufsClient  UFSClient
	sshPool    *sshpool.Pool
	servodPool *servod.Pool
	// Cache received devices from inventory
	devices   map[string]*tlw.Dut
	hostTypes map[string]hostType
	// Map to provide name if the DUT host as value and other hosts as key.
	hostToParents map[string]string
}

// New build new local TLW Access instance.
func New(ufs UFSClient, csac CSAClient) (tlw.Access, error) {
	c := &tlwClient{
		ufsClient:     ufs,
		csaClient:     csac,
		sshPool:       sshpool.New(ssh.SSHConfig()),
		servodPool:    servod.NewPool(),
		devices:       make(map[string]*tlw.Dut),
		hostTypes:     make(map[string]hostType),
		hostToParents: make(map[string]string),
	}
	return c, nil
}

// Close closes all used resources.
func (c *tlwClient) Close(ctx context.Context) error {
	if err := c.sshPool.Close(); err != nil {
		return errors.Annotate(err, "tlw client").Err()
	}
	return c.servodPool.Close()
}

// Ping performs ping by resource name.
//
// For containers it checks if it is up.
func (c *tlwClient) Ping(ctx context.Context, resourceName string, count int) error {
	dut, err := c.getDevice(ctx, resourceName)
	if err != nil {
		return errors.Annotate(err, "ping").Err()
	}
	if c.isServoHost(resourceName) && isServodContainer(dut) {
		log.Info(ctx, "Ping: servod container %s starting...", resourceName)
		d, err := c.dockerClient(ctx)
		if err != nil {
			return errors.Annotate(err, "ping").Err()
		}
		containerName := servoContainerName(dut)
		if up, err := d.IsUp(ctx, containerName); err != nil {
			return errors.Annotate(err, "ping").Err()
		} else if up {
			log.Info(ctx, "Ping: servod container %s is up!", containerName)
			return nil
		}
		return errors.Reason("ping: container %q is down", containerName).Err()
	} else {
		err = ping(resourceName, count)
		return errors.Annotate(err, "ping").Err()
	}
}

// Run executes command on device by SSH related to resource name.
//
// Foc containers: For backwards compatibility if command provided without arguments
// we assume the whole command in one string and run it in linux shell (/bin/sh -c).
func (c *tlwClient) Run(ctx context.Context, req *tlw.RunRequest) *tlw.RunResult {
	fullCmd := strings.Join(append([]string{req.GetCommand()}, req.GetArgs()...), " ")
	dut, err := c.getDevice(ctx, req.GetResource())
	if err != nil {
		return &tlw.RunResult{
			Command:  fullCmd,
			ExitCode: -1,
			Stderr:   fmt.Sprintf("run: %s", err),
		}
	}
	// For backward compatibility we set max limit 1 hour for any request.
	// 1 hours as some provisioning or download can take longer.
	timeout := time.Hour
	if req.GetTimeout().IsValid() {
		timeout = req.GetTimeout().AsDuration()
	}
	// Servod-container does not have ssh access so to execute any commands
	// we need to use the docker client.
	if c.isServoHost(req.GetResource()) && isServodContainer(dut) {
		d, err := c.dockerClient(ctx)
		if err != nil {
			return &tlw.RunResult{
				Command:  fullCmd,
				ExitCode: -1,
				Stderr:   fmt.Sprintf("run: %s", err),
			}
		}
		eReq := &docker.ExecRequest{
			Timeout: timeout,
			Cmd:     append([]string{req.GetCommand()}, req.GetArgs()...),
		}
		containerName := servoContainerName(dut)
		// For backwards compatibility if only command provide we assume
		// that that is whole command in one line. We will run it in linux shell.
		if strings.Contains(req.GetCommand(), " ") && len(req.GetArgs()) == 0 {
			eReq.Cmd = []string{"/bin/sh", "-c", req.GetCommand()}
			// Quoting is only works because the string created for user
			// representation and logs, not for use for execution.
			fullCmd = fmt.Sprintf("/bin/sh -c %q", req.GetCommand())
		}
		// As container is created and running we can execute the commands.
		// TODO: Receive timeout from request.
		if res, err := d.Exec(ctx, containerName, eReq); err != nil {
			return &tlw.RunResult{
				Command:  fullCmd,
				ExitCode: -1,
				Stderr:   fmt.Sprintf("run: %s", err),
			}
		} else {
			return &tlw.RunResult{
				Command:  fullCmd,
				ExitCode: res.ExitCode,
				Stdout:   res.Stdout,
				Stderr:   res.Stderr,
			}
		}
	} else {
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		cr := make(chan *tlw.RunResult, 1)
		go func() {
			cr <- ssh.Run(ctx, c.sshPool, localproxy.BuildAddr(req.GetResource()), fullCmd)
		}()
		select {
		case r := <-cr:
			return r
		case <-ctx.Done():
			// If we reached timeout first.
			return &tlw.RunResult{
				Command:  fullCmd,
				ExitCode: 124,
				Stderr:   fmt.Sprintf("run: excited timeout %s", timeout),
			}
		}
	}
}

// InitServod initiates servod daemon on servo-host.
func (c *tlwClient) InitServod(ctx context.Context, req *tlw.InitServodRequest) error {
	dut, err := c.getDevice(ctx, req.Resource)
	if err != nil {
		return errors.Annotate(err, "init servod %q", req.Resource).Err()
	}
	if dut.ServoHost == nil || dut.ServoHost.Name == "" {
		return errors.Reason("init servod %q: servo is not found", req.Resource).Err()
	}
	if isServodContainer(dut) {
		err := c.startServodContainer(ctx, dut)
		return errors.Annotate(err, "init servod %q", req.Resource).Err()
	}
	s, err := c.servodPool.Get(
		localproxy.BuildAddr(dut.ServoHost.Name),
		int32(dut.ServoHost.ServodPort),
		func() ([]string, error) {
			return dutinfo.GenerateServodParams(dut, req.Options)
		})
	if err != nil {
		return errors.Annotate(err, "init servod %q", req.Resource).Err()
	}
	if err := s.Prepare(ctx, c.sshPool); err != nil {
		return errors.Annotate(err, "init servod %q", req.Resource).Err()
	}
	return nil
}

// dockerServodImageName provides image for servod when use container.
func dockerServodImageName() string {
	// TODO(otabek): Value has to come here from somewhere.
	return "us-docker.pkg.dev/chromeos-partner-moblab/common-core/servod:release"
}

// startServodContainer start servod container if required.
func (c *tlwClient) startServodContainer(ctx context.Context, dut *tlw.Dut) error {
	containerName := servoContainerName(dut)
	d, err := c.dockerClient(ctx)
	if err != nil {
		return errors.Annotate(err, "start servod container").Err()
	}
	if up, err := d.IsUp(ctx, containerName); err != nil {
		return errors.Annotate(err, "start servod container").Err()
	} else if up {
		log.Debug(ctx, "Servod container %s is already up!", containerName)
		return nil
	}
	// TODO: Receive timeout from request.
	sp := fmt.Sprintf("%d", dut.ServoHost.ServodPort)
	// TODO(otabek): move servod param preparation to separate method.
	envVar := []string{
		fmt.Sprintf("PORT=%s", sp),
		fmt.Sprintf("BOARD=%s", dut.Board),
		fmt.Sprintf("MODEL=%s", dut.Model),
		"REC_MODE=1",
	}
	if sn := dut.ServoHost.Servo.SerialNumber; sn != "" {
		envVar = append(envVar, fmt.Sprintf("SERIAL=%s", sn))
	}
	if vs, ok := dut.ExtraAttributes[dutinfo.ExtraAttributesServoSetup]; ok {
		for _, v := range vs {
			if v == dutinfo.ExtraAttributesServoSetupDual {
				envVar = append(envVar, "DUAL_V4=1")
				break
			}
		}
	}
	if pools, ok := dut.ExtraAttributes[dutinfo.ExtraAttributesPools]; ok {
		for _, p := range pools {
			if strings.Contains(p, "faft-cr50") {
				envVar = append(envVar, "CONFIG=cr50.xml")
				break
			}
		}
	}
	res, err := d.Start(ctx, containerName, &docker.ContainerArgs{
		Detached:   true,
		ImageName:  dockerServodImageName(),
		EnvVar:     envVar,
		Volumes:    []string{"/dev:/dev"},
		Privileged: true,
		Exec:       []string{"bash", "/start_servod.sh"},
	}, time.Hour)
	if err != nil {
		return errors.Annotate(err, "start servod container").Err()
	}
	log.Debug(ctx, "Container started with id:%s\n with errout: %s", res.Stdout, res.Stderr)
	// Wait 3 seconds as sometimes container is not fully initialized and fail
	// when start ing working with servod or tooling.
	// TODO(otabek): Move to servod-container wrapper.
	time.Sleep(3 * time.Second)
	// Waiting to finish servod initialization.
	eReq := &docker.ExecRequest{
		Timeout: 2 * time.Minute,
		Cmd:     []string{"servodtool", "instance", "wait-for-active", "-p", sp},
	}
	if _, err := d.Exec(ctx, containerName, eReq); err != nil {
		return errors.Annotate(err, "start servod container").Err()
	}
	log.Debug(ctx, "Servod container %s started and up!", containerName)
	return nil
}

// StopServod stops servod daemon on servo-host.
func (c *tlwClient) StopServod(ctx context.Context, resourceName string) error {
	dut, err := c.getDevice(ctx, resourceName)
	if err != nil {
		return errors.Annotate(err, "stop servod %q", resourceName).Err()
	}
	if dut.ServoHost == nil || dut.ServoHost.Name == "" {
		return errors.Reason("stop servod %q: servo is not found", resourceName).Err()
	}
	if isServodContainer(dut) {
		if d, err := c.dockerClient(ctx); err != nil {
			return errors.Annotate(err, "stop servod %q", resourceName).Err()
		} else {
			err := d.Remove(ctx, servoContainerName(dut), true)
			return errors.Annotate(err, "stop servod %q", resourceName).Err()
		}
	}
	s, err := c.servodPool.Get(
		localproxy.BuildAddr(dut.ServoHost.Name),
		int32(dut.ServoHost.ServodPort),
		nil)
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
	dut, err := c.getDevice(ctx, req.Resource)
	if err != nil {
		return fail(err)
	}
	if dut.ServoHost == nil || dut.ServoHost.Name == "" {
		return fail(errors.Reason("call servod %q: servo not found", req.Resource).Err())
	}
	var val *xmlrpc.Value
	// For container connect to the container as it running on the same host.
	if isServodContainer(dut) {
		d, err := c.dockerClient(ctx)
		if err != nil {
			return fail(err)
		}
		var addr string
		if addr, err = d.IPAddress(ctx, servoContainerName(dut)); err != nil {
			return fail(err)
		}
		rpc := tlw_xmlrpc.New(addr, dut.ServoHost.ServodPort)
		val, err = servod.Call(ctx, rpc, req.Method, req.Args)
	} else {
		// For labstation using port forward by ssh.
		s, err := c.servodPool.Get(
			localproxy.BuildAddr(dut.ServoHost.Name),
			int32(dut.ServoHost.ServodPort),
			func() ([]string, error) {
				return dutinfo.GenerateServodParams(dut, req.Options)
			})
		if err != nil {
			return fail(err)
		}
		val, err = s.Call(ctx, c.sshPool, req.Method, req.Args)
	}
	if err != nil {
		return fail(err)
	}
	return &tlw.CallServodResponse{
		Value: val,
		Fault: false,
	}
}

// CallBluetoothPeer executes a command on bluetooth-peer service.
func (c *tlwClient) CallBluetoothPeer(ctx context.Context, req *tlw.CallBluetoothPeerRequest) *tlw.CallBluetoothPeerResponse {
	// Translator to convert error to response structure.
	fail := func(err error) *tlw.CallBluetoothPeerResponse {
		return &tlw.CallBluetoothPeerResponse{
			Value: &xmlrpc.Value{
				ScalarOneof: &xmlrpc.Value_String_{
					String_: fmt.Sprintf("call servod %q: %s", req.GetResource(), err),
				},
			},
			Fault: true,
		}
	}
	// Check if the name was detected by loaded device.
	_, err := c.getDevice(ctx, req.GetResource())
	if err != nil {
		return fail(err)
	}
	s, err := c.servodPool.Get(
		localproxy.BuildAddr(req.GetResource()),
		int32(deafultBluetoothPeerServerPort),
		func() ([]string, error) { return nil, nil })
	if err != nil {
		return fail(err)
	}
	val, err := s.Call(ctx, c.sshPool, req.GetMethod(), req.GetArgs())
	if err != nil {
		return fail(err)
	}
	return &tlw.CallBluetoothPeerResponse{
		Value: val,
		Fault: false,
	}
}

// CopyFileTo copies file to remote device from local.
func (c *tlwClient) CopyFileTo(ctx context.Context, req *tlw.CopyRequest) error {
	if err := tlwio.CopyFileTo(ctx, c.sshPool, req); err != nil {
		return errors.Annotate(err, "copy file to").Err()
	}
	return nil
}

// CopyFileFrom copies file from remote device to local.
func (c *tlwClient) CopyFileFrom(ctx context.Context, req *tlw.CopyRequest) error {
	if err := tlwio.CopyFileFrom(ctx, c.sshPool, req); err != nil {
		return errors.Annotate(err, "copy file from").Err()
	}
	return nil
}

// CopyDirectoryTo copies directory to remote device from local, recursively.
func (c *tlwClient) CopyDirectoryTo(ctx context.Context, req *tlw.CopyRequest) error {
	if err := tlwio.CopyDirectoryTo(ctx, c.sshPool, req); err != nil {
		return errors.Annotate(err, "copy directory to").Err()
	}
	return nil
}

// CopyDirectoryFrom copies directory from remote device to local, recursively.
func (c *tlwClient) CopyDirectoryFrom(ctx context.Context, req *tlw.CopyRequest) error {
	if err := tlwio.CopyDirectoryFrom(ctx, c.sshPool, req); err != nil {
		return errors.Annotate(err, "copy directory from").Err()
	}
	return nil
}

// SetPowerSupply manages power supply for requested.
func (c *tlwClient) SetPowerSupply(ctx context.Context, req *tlw.SetPowerSupplyRequest) *tlw.SetPowerSupplyResponse {
	if req == nil || req.Resource == "" {
		return &tlw.SetPowerSupplyResponse{
			Status: tlw.PowerSupplyResponseStatusError,
			Reason: "resource is not specified",
		}
	}
	dut, err := c.getDevice(ctx, req.Resource)
	if err != nil {
		return &tlw.SetPowerSupplyResponse{
			Status: tlw.PowerSupplyResponseStatusError,
			Reason: err.Error(),
		}
	}
	hostname, outlet := dutinfo.GetRpmInfo(dut)
	log.Debug(ctx, "Set power supply %s: has rpm info %s:%s.", req.Resource, hostname, outlet)
	if hostname == "" || outlet == "" {
		return &tlw.SetPowerSupplyResponse{
			Status: tlw.PowerSupplyResponseStatusNoConfig,
			Reason: "power supply config missing or incorrect",
		}
	}
	var s rpm.PowerState
	switch req.State {
	case tlw.PowerSupplyActionOn:
		s = rpm.PowerStateOn
	case tlw.PowerSupplyActionOff:
		s = rpm.PowerStateOff
	case tlw.PowerSupplyActionCycle:
		s = rpm.PowerStateCycle
	default:
		return &tlw.SetPowerSupplyResponse{
			Status: tlw.PowerSupplyResponseStatusError,
			Reason: fmt.Sprintf("unknown rpm state: %s", string(req.State)),
		}
	}
	log.Debug(ctx, "Set power supply %s: state: %q for %s:%s.", req.Resource, s, hostname, outlet)
	rpmReq := &rpm.RPMPowerRequest{
		Hostname:          dut.Name,
		PowerUnitHostname: hostname,
		PowerunitOutlet:   outlet,
		State:             s,
	}
	if err := rpm.SetPowerState(ctx, rpmReq); err != nil {
		return &tlw.SetPowerSupplyResponse{
			Status: tlw.PowerSupplyResponseStatusError,
			Reason: err.Error(),
		}
	}
	return &tlw.SetPowerSupplyResponse{
		Status: tlw.PowerSupplyResponseStatusOK,
	}
}

// GetCacheUrl provides URL to download requested path to file.
// URL will use to download image to USB-drive and provisioning.
func (c *tlwClient) GetCacheUrl(ctx context.Context, resourceName, filePath string) (string, error) {
	// TODO(otabek@): Add logic to understand local file and just return it back.
	addr := fmt.Sprintf("0.0.0.0:%d", tlwPort)
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		return "", errors.Annotate(err, "connect to background TLW").Err()
	}
	defer func() { conn.Close() }()
	return CacheForDut(ctx, conn, filePath, resourceName)
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
		dut, err := dutinfo.ConvertDut(dd)
		if err != nil {
			return nil, errors.Annotate(err, "list resources %q", name).Err()
		}
		c.cacheDevice(dut)
		return []string{dut.Name}, nil
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
	dut, err := c.getDevice(ctx, name)
	if err != nil {
		return nil, errors.Annotate(err, "get DUT %q", name).Err()
	}
	gv, err := c.getStableVersion(ctx, dut)
	if err != nil {
		log.Info(ctx, "Get DUT %q: failed to receive stable-version. Error: %s", name, err)
	} else {
		dut.StableVersion = gv
	}
	dut.ProvisionedInfo, err = localinfo.ReadProvisionInfo(ctx, dut.Name)
	return dut, errors.Annotate(err, "get dut").Err()
}

// getDevice receives device from inventory.
func (c *tlwClient) getDevice(ctx context.Context, name string) (*tlw.Dut, error) {
	if dutName, ok := c.hostToParents[name]; ok {
		// the device was previously
		name = dutName
	}
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
	dut, err := dutinfo.ConvertDut(dd)
	if err != nil {
		return nil, errors.Annotate(err, "get device %q", name).Err()
	}
	c.cacheDevice(dut)
	log.Debug(ctx, "Get device %q: cached received device.", name)
	return dut, nil
}

// cacheDevice puts device to local cache and set list host name knows for DUT.
func (c *tlwClient) cacheDevice(dut *tlw.Dut) {
	if dut == nil {
		// Skip as DUT not found.
		return
	}
	name := dut.Name
	c.devices[name] = dut
	c.hostToParents[name] = name
	c.hostTypes[dut.Name] = hostTypeCros
	if dut.ServoHost != nil && dut.ServoHost.Name != "" {
		c.hostTypes[dut.ServoHost.Name] = hostTypeServo
		c.hostToParents[dut.ServoHost.Name] = name
	}
	for _, bt := range dut.BluetoothPeerHosts {
		if bt.Name != "" {
			c.hostTypes[bt.Name] = hostTypeBtPeer
			c.hostToParents[bt.Name] = name
		}
	}
	for _, router := range dut.WifiRouterHosts {
		if router != nil && router.Name != "" {
			c.hostTypes[router.Name] = hostTypeRouter
			c.hostToParents[router.Name] = name
		}
	}
	if dut.ChameleonHost != nil && dut.ChameleonHost.Name != "" {
		c.hostTypes[dut.ChameleonHost.Name] = hostTypeChameleon
		c.hostToParents[dut.ChameleonHost.Name] = name
	}
}

// cacheDevice puts device to local cache and set list host name knows for DUT.
func (c *tlwClient) unCacheDevice(dut *tlw.Dut) {
	if dut == nil {
		// Skip as DUT not provided.
		return
	}
	name := dut.Name
	delete(c.hostToParents, name)
	delete(c.hostTypes, name)
	if dut.ServoHost != nil && dut.ServoHost.Name != "" {
		delete(c.hostTypes, dut.ServoHost.Name)
		delete(c.hostToParents, dut.ServoHost.Name)
	}
	for _, bt := range dut.BluetoothPeerHosts {
		if bt.Name != "" {
			delete(c.hostTypes, bt.Name)
			delete(c.hostToParents, bt.Name)
		}
	}
	if dut.ChameleonHost != nil && dut.ChameleonHost.Name != "" {
		delete(c.hostTypes, dut.ChameleonHost.Name)
		delete(c.hostToParents, dut.ChameleonHost.Name)
	}
	delete(c.devices, name)
}

// isServoHost tells if host is servo-host.
func (c *tlwClient) isServoHost(host string) bool {
	if v, ok := c.hostTypes[host]; ok {
		return v == hostTypeServo
	}
	return false
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
	dut, err := c.getDevice(ctx, dut.Name)
	if err != nil {
		return errors.Annotate(err, "update DUT %q", dut.Name).Err()
	}
	req, err := dutinfo.CreateUpdateDutRequest(dut.Id, dut)
	if err != nil {
		return errors.Annotate(err, "update DUT %q", dut.Name).Err()
	}
	log.Debug(ctx, "Update DUT: update request: %s", req)
	if _, err := c.ufsClient.UpdateDutState(ctx, req); err != nil {
		return errors.Annotate(err, "update DUT %q", dut.Name).Err()
	}
	c.unCacheDevice(dut)
	if ufs, ok := c.ufsClient.(dutstate.UFSClient); ok {
		if err := dutstate.Update(ctx, ufs, dut.Name, dut.State); err != nil {
			return errors.Annotate(err, "update DUT %q", dut.Name).Err()
		}
	} else {
		return errors.Reason("update DUT %q: dutstate.UFSClient interface is not implemented by client", dut.Name).Err()
	}
	return errors.Annotate(localinfo.UpdateProvisionInfo(ctx, dut), "udpate dut").Err()
}

// Provision triggers provisioning of the device.
func (c *tlwClient) Provision(ctx context.Context, req *tlw.ProvisionRequest) error {
	if req == nil {
		return errors.Reason("provision: request is empty").Err()
	}
	if req.GetResource() == "" {
		return errors.Reason("provision: resource is not specified").Err()
	}
	if req.GetSystemImagePath() == "" {
		return errors.Reason("provision: system image path is not specified").Err()
	}
	log.Debug(ctx, "Started provisioning by TLS: %s", req)
	addr := fmt.Sprintf("0.0.0.0:%d", tlsPort)
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		return errors.Annotate(err, "provision: connect to TLS").Err()
	}
	defer func() { conn.Close() }()
	err = TLSProvision(ctx, conn, req)
	return errors.Annotate(err, "provision").Err()
}

// dockerClient provides docker client for target container by expected name of container.
func (c *tlwClient) dockerClient(ctx context.Context) (docker.Client, error) {
	d, err := docker.NewClient(ctx)
	if err != nil {
		return nil, errors.Annotate(err, "docker client").Err()
	}
	// Print the containers to test connectivity.
	if err = d.PrintAll(ctx); err != nil {
		return nil, errors.Annotate(err, "docker client").Err()
	}
	return d, nil
}

// isServodContainer checks if DUT using servod-container.
// For now just simple check if servod container is provided.
// Later need distinguish when container running on the same host or remove one.
func isServodContainer(d *tlw.Dut) bool {
	return servoContainerName(d) != ""
}

// servoContainerName returns container name specified for servo-host.
func servoContainerName(d *tlw.Dut) string {
	if d == nil || d.ServoHost == nil {
		return ""
	}
	return d.ServoHost.ContainerName
}
