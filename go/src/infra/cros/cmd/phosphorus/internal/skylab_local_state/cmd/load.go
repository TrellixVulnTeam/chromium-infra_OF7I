// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"

	"infra/cros/cmd/phosphorus/internal/botcache"
	"infra/cros/cmd/phosphorus/internal/skylab_local_state/inv"
	"infra/cros/cmd/phosphorus/internal/skylab_local_state/location"
	"infra/cros/cmd/phosphorus/internal/skylab_local_state/ufs"
	androidlbls "infra/libs/skylab/inventory/autotest/attached_device"
	chromeoslbls "infra/libs/skylab/inventory/autotest/labels"
	ufsapi "infra/unifiedfleet/api/v1/rpc"
	ufsutil "infra/unifiedfleet/app/util"

	labapi "go.chromium.org/chromiumos/config/go/test/lab/api"

	"go.chromium.org/chromiumos/infra/proto/go/lab_platform"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/skylab_local_state"
)

// Load subcommand: Gather DUT labels and attributes into a host info file.
func Load(authOpts auth.Options) *subcommands.Command {
	return &subcommands.Command{
		UsageLine: "load -input_json /path/to/input.json -output_json /path/to/output.json",
		ShortDesc: "Gather DUT labels and attributes into a host info file.",
		LongDesc: `Gather DUT labels and attributes into a host info file.

Get static labels and attributes from the inventory service and provisionable
labels and attributes from the local bot state cache file.

Write all labels and attributes as a
test_platform/skylab_local_state/host_info.proto JSON-pb to the host info store
file inside the results directory.

Write provisionable labels and DUT hostname as a LoadResponse JSON-pb to
the file given by -output_json.
`,
		CommandRun: func() subcommands.CommandRun {
			c := &loadRun{}

			c.authFlags.Register(&c.Flags, authOpts)

			c.Flags.StringVar(&c.inputPath, "input_json", "", "Path to JSON LoadRequest to read.")
			c.Flags.StringVar(&c.outputPath, "output_json", "", "Path to JSON LoadResponse to write.")
			return c
		},
	}
}

type loadRun struct {
	subcommands.CommandRunBase

	authFlags authcli.Flags

	inputPath  string
	outputPath string
}

func (c *loadRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.validateArgs(); err != nil {
		fmt.Fprintln(a.GetErr(), err.Error())
		c.Flags.Usage()
		return 1
	}

	err := c.innerRun(a, args, env)
	if err != nil {
		fmt.Fprintf(a.GetErr(), err.Error())
		return 1
	}
	return 0
}

func (c *loadRun) validateArgs() error {
	if c.inputPath == "" {
		return fmt.Errorf("-input_json not specified")
	}

	if c.outputPath == "" {
		return fmt.Errorf("-output_json not specified")
	}

	return nil
}

func (c *loadRun) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	var request skylab_local_state.LoadRequest
	if err := readJSONPb(c.inputPath, &request); err != nil {
		return err
	}

	if err := validateLoadRequest(&request); err != nil {
		return err
	}

	ctx := cli.GetContext(a, c, env)

	client, err := ufs.NewClient(ctx, request.Config.CrosUfsService, &c.authFlags)
	if err != nil {
		return err
	}

	deviceInfos, err := getAllDevicesInfo(ctx, client, request.DutName)
	if err != nil {
		return err
	}

	resultsDir := location.ResultsDir(request.Config.AutotestDir, request.RunId, request.TestId)

	for _, deviceInfo := range deviceInfos {
		deviceState, err := getDeviceState(request.Config.AutotestDir, deviceInfo)
		if err != nil {
			return err
		}
		hostInfo, err := getFullHostInfo(ctx, deviceInfo, deviceState)
		if err != nil {
			return err
		}
		if err := writeHostInfo(resultsDir, getHostname(deviceInfo), hostInfo); err != nil {
			return err
		}
	}

	response := skylab_local_state.LoadResponse{
		ResultsDir:  resultsDir,
		DutTopology: createDutTopology(deviceInfos),
	}

	return writeJSONPb(c.outputPath, &response)
}

func validateLoadRequest(request *skylab_local_state.LoadRequest) error {
	if request == nil {
		return fmt.Errorf("nil request")
	}

	var missingArgs []string

	if request.Config.GetAdminService() == "" {
		missingArgs = append(missingArgs, "admin service")
	}

	if request.Config.GetAutotestDir() == "" {
		missingArgs = append(missingArgs, "autotest dir")
	}

	if request.DutName == "" {
		missingArgs = append(missingArgs, "DUT hostname")
	}

	if request.RunId == "" {
		missingArgs = append(missingArgs, "Swarming run ID")
	}

	if request.DutId == "" {
		missingArgs = append(missingArgs, "Swarming run ID")
	}

	if len(missingArgs) > 0 {
		return fmt.Errorf("no %s provided", strings.Join(missingArgs, ", "))
	}

	return nil
}

// getAllDevicesInfo fetches inventory entry of all DUTs / attached devices by a resource name from UFS.
func getAllDevicesInfo(ctx context.Context, client ufsapi.FleetClient, resourceName string) ([]*ufsapi.GetDeviceDataResponse, error) {
	resp, err := getDeviceInfo(ctx, client, resourceName)
	if err != nil {
		return nil, err
	}
	if resp.GetResourceType() == ufsapi.GetDeviceDataResponse_RESOURCE_TYPE_SCHEDULING_UNIT {
		return getSchedulingUnitInfo(ctx, client, resp.GetSchedulingUnit().GetMachineLSEs())
	}
	return appendDeviceData([]*ufsapi.GetDeviceDataResponse{}, resp, resourceName)
}

// getSchedulingUnitInfo fetches device info for every DUT / attached device in the scheduling unit.
func getSchedulingUnitInfo(ctx context.Context, client ufsapi.FleetClient, hostnames []string) ([]*ufsapi.GetDeviceDataResponse, error) {
	// Get device info for every DUT / attached device in the scheduling unit.
	var deviceData []*ufsapi.GetDeviceDataResponse
	for _, hostname := range hostnames {
		resp, err := getDeviceInfo(ctx, client, hostname)
		if err != nil {
			return nil, err
		}
		if deviceData, err = appendDeviceData(deviceData, resp, hostname); err != nil {
			return nil, err
		}
	}
	return deviceData, nil
}

// getDeviceInfo fetches a device entry from UFS.
func getDeviceInfo(ctx context.Context, client ufsapi.FleetClient, hostname string) (*ufsapi.GetDeviceDataResponse, error) {
	osctx := ufs.SetupContext(ctx, ufsutil.OSNamespace)
	req := &ufsapi.GetDeviceDataRequest{
		Hostname: hostname,
	}
	resp, err := client.GetDeviceData(osctx, req)
	if err != nil {
		return nil, errors.Annotate(err, "get device info for %s", hostname).Err()
	}
	return resp, nil
}

// appendDeviceData appends a device data response to the list of responses
// after validation. Returns error if the device type is different from ChromeOs
// or Android device.
func appendDeviceData(deviceData []*ufsapi.GetDeviceDataResponse, resp *ufsapi.GetDeviceDataResponse, hostname string) ([]*ufsapi.GetDeviceDataResponse, error) {
	switch resp.GetResourceType() {
	case ufsapi.GetDeviceDataResponse_RESOURCE_TYPE_CHROMEOS_DEVICE:
		fallthrough
	case ufsapi.GetDeviceDataResponse_RESOURCE_TYPE_ATTACHED_DEVICE:
		return append(deviceData, resp), nil
	}
	return nil, fmt.Errorf("invalid device type for %s", hostname)
}

// hostInfoFromDeviceInfo extracts attributes and labels from an inventory
// entry and assembles them into a host info file proto.
func hostInfoFromDeviceInfo(deviceInfo *ufsapi.GetDeviceDataResponse) *skylab_local_state.AutotestHostInfo {
	labels := make([]string, 0)
	attributes := make(map[string]string)
	switch deviceInfo.GetResourceType() {
	case ufsapi.GetDeviceDataResponse_RESOURCE_TYPE_CHROMEOS_DEVICE:
		dut := deviceInfo.GetChromeOsDeviceData().GetDutV1()
		labels = chromeoslbls.Convert(dut.Common.GetLabels())
		for _, attribute := range dut.Common.GetAttributes() {
			attributes[attribute.GetKey()] = attribute.GetValue()
		}
	case ufsapi.GetDeviceDataResponse_RESOURCE_TYPE_ATTACHED_DEVICE:
		labels = androidlbls.Convert(deviceInfo.GetAttachedDeviceData())
	}
	return &skylab_local_state.AutotestHostInfo{
		Attributes:        attributes,
		Labels:            labels,
		SerializerVersion: currentSerializerVersion,
	}
}

// getDUTTopology fetches the DUT topology from the inventory server
func getDUTTopology(ctx context.Context, hostname string) (*labapi.DutTopology, error) {
	invService, err := inv.NewClient()
	if err != nil {
		return nil, errors.Annotate(err, "Start InventoryServer client").Err()
	}
	dutid := &labapi.DutTopology_Id{Value: hostname}
	stream, err := invService.Client.GetDutTopology(ctx, &labapi.GetDutTopologyRequest{Id: dutid})
	if err != nil {
		return nil, errors.Annotate(err, "InventoryServer.GetDutTopology").Err()
	}
	response := &labapi.GetDutTopologyResponse{}
	err = stream.RecvMsg(response)
	if err != nil {
		return nil, errors.Annotate(err, "InventoryServer get response").Err()
	}
	inv.CloseClient(invService)
	return response.GetSuccess().DutTopology, nil
}

// addDeviceStateToHostInfo adds provisionable labels and attributes from
// the bot state to the host info labels and attributes.
func addDeviceStateToHostInfo(hostInfo *skylab_local_state.AutotestHostInfo, dutState *lab_platform.DutState) {
	for label, value := range dutState.GetProvisionableLabels() {
		hostInfo.Labels = append(hostInfo.Labels, label+":"+value)
	}
	for attribute, value := range dutState.GetProvisionableAttributes() {
		hostInfo.Attributes[attribute] = value
	}
}

// writeHostInfo writes a JSON-encoded AutotestHostInfo proto to the
// DUT host info file inside the results directory.
func writeHostInfo(resultsDir string, dutName string, i *skylab_local_state.AutotestHostInfo) error {
	p := location.HostInfoFilePath(resultsDir, dutName)

	if err := writeJSONPb(p, i); err != nil {
		return errors.Annotate(err, "write host info for %s", dutName).Err()
	}

	return nil
}

// getFullHostInfo aggregates data from local and admin services state into one hostinfo object
func getFullHostInfo(ctx context.Context, deviceInfo *ufsapi.GetDeviceDataResponse, deviceState *lab_platform.DutState) (*skylab_local_state.AutotestHostInfo, error) {
	var hostInfo *skylab_local_state.AutotestHostInfo
	useDutTopo := os.Getenv("USE_DUT_TOPO")
	if strings.ToLower(useDutTopo) == "true" {
		hostname := getHostname(deviceInfo)
		dutTopo, err := getDUTTopology(ctx, hostname)
		// Output dutTopo to stdout during development
		log.Printf(proto.MarshalTextString(dutTopo))
		if err != nil {
			// Output error to stdout during testing
			fmt.Println("Error getting DUT topology: ", err.Error())
			return nil, errors.Annotate(err, "get dut topology").Err()
		}

		hostInfo, err = convertDutTopologyToHostInfo(dutTopo)
		if err != nil {
			return nil, errors.Annotate(err, "convert dut topology to host info").Err()
		}
		// Output hostInfo to stdout during development
		log.Printf("Host info from DutTopology:\n")
		log.Printf(proto.MarshalTextString(hostInfo))

		// This is done only for testing purposes.
		// TODO(b/201424819): Remove this part once testing is done.
		oldHostInfo := hostInfoFromDeviceInfo(deviceInfo)
		addDeviceStateToHostInfo(oldHostInfo, deviceState)
		log.Printf("Old Host info:\n")
		log.Printf(proto.MarshalTextString(oldHostInfo))
	} else {
		hostInfo = hostInfoFromDeviceInfo(deviceInfo)
		addDeviceStateToHostInfo(hostInfo, deviceState)
	}
	return hostInfo, nil
}

// getHostname returns a hostname extracted from GetDeviceDataResponse proto
func getHostname(deviceInfo *ufsapi.GetDeviceDataResponse) string {
	if deviceInfo.GetResourceType() == ufsapi.GetDeviceDataResponse_RESOURCE_TYPE_CHROMEOS_DEVICE {
		return deviceInfo.GetChromeOsDeviceData().GetLabConfig().GetHostname()
	}
	return deviceInfo.GetAttachedDeviceData().GetLabConfig().GetHostname()
}

// getBoard returns a board name extracted from GetDeviceDataResponse proto
func getBoard(deviceInfo *ufsapi.GetDeviceDataResponse) string {
	if deviceInfo.GetResourceType() == ufsapi.GetDeviceDataResponse_RESOURCE_TYPE_CHROMEOS_DEVICE {
		return deviceInfo.GetChromeOsDeviceData().GetMachine().GetChromeosMachine().GetBuildTarget()
	}
	return deviceInfo.GetAttachedDeviceData().GetMachine().GetAttachedDevice().GetBuildTarget()
}

// getModel returns a DUT model extracted from GetDeviceDataResponse proto
func getModel(deviceInfo *ufsapi.GetDeviceDataResponse) string {
	if deviceInfo.GetResourceType() == ufsapi.GetDeviceDataResponse_RESOURCE_TYPE_CHROMEOS_DEVICE {
		return deviceInfo.GetChromeOsDeviceData().GetMachine().GetChromeosMachine().GetModel()
	}
	return deviceInfo.GetAttachedDeviceData().GetMachine().GetAttachedDevice().GetModel()
}

// getDeviceState returns a lab_platform.DutState object with a bot state cached on drone.
func getDeviceState(autotestDir string, deviceInfo *ufsapi.GetDeviceDataResponse) (*lab_platform.DutState, error) {
	bcs := botcache.Store{
		CacheDir: autotestDir,
		Name:     getHostname(deviceInfo),
	}
	return bcs.Load()
}

// createDutTopology construct a DutTopology will be wrapped into LoadResponse.
func createDutTopology(deviceInfos []*ufsapi.GetDeviceDataResponse) []*skylab_local_state.Dut {
	var dt []*skylab_local_state.Dut
	for _, deviceInfo := range deviceInfos {
		dt = append(dt, &skylab_local_state.Dut{
			Hostname: getHostname(deviceInfo),
			Board:    getBoard(deviceInfo),
			Model:    getModel(deviceInfo),
		})
	}
	return dt
}

// convert dut topology to host info.
func convertDutTopologyToHostInfo(dutTopo *labapi.DutTopology) (*skylab_local_state.AutotestHostInfo, error) {
	// Should always have one dut info as this being called for each dut individually.
	if len(dutTopo.Duts) != 1 {
		return nil, fmt.Errorf("exactly one dut expected but found %d", len(dutTopo.Duts))
	}
	dut := dutTopo.Duts[0]

	// Add attributes & labels
	attrMap := make(map[string]string)
	labels := make([]string, 0)

	switch dut.GetDutType().(type) {
	case *labapi.Dut_Chromeos:
		attrMap, labels = appendChromeOsLabels(attrMap, labels, dut.GetChromeos())
	case *labapi.Dut_Android_:
		attrMap, labels = appendAndroidLabels(attrMap, labels, dut.GetAndroid())
	}

	return &skylab_local_state.AutotestHostInfo{
		Attributes:        attrMap,
		Labels:            labels,
		SerializerVersion: currentSerializerVersion,
	}, nil
}

// appendChromeOsLabels appends labels extracted from ChromeOS device.
func appendChromeOsLabels(attrMap map[string]string, labels []string, dut *labapi.Dut_ChromeOS) (map[string]string, []string) {
	// - DutModel
	// - DutModel.BuildTarget (Board name)
	if board := dut.GetDutModel().GetBuildTarget(); board != "" {
		labels = append(labels, "board:"+strings.ToLower(board))
	}
	// - DutModel.ModelName (ChromeOS DUT model name)
	if model := dut.GetDutModel().GetModelName(); model != "" {
		labels = append(labels, "model:"+strings.ToLower(model))
	}

	// - Servo
	if servo := dut.GetServo(); servo != nil {
		if servo.GetPresent() {
			labels = append(labels, "servo")
		}

		if servoIpEndpoint := servo.GetServodAddress(); servoIpEndpoint != nil {
			if address := servoIpEndpoint.GetAddress(); address != "" {
				attrMap["servo_host"] = strings.ToLower(address)
			}
			if port := servoIpEndpoint.GetPort(); port != 0 {
				attrMap["servo_port"] = fmt.Sprintf("%v", port)
			}
		}
		if serial := servo.GetSerial(); serial != "" {
			attrMap["servo_serial"] = serial
		}
	}
	// - Chameleon
	if chameleon := dut.GetChameleon(); chameleon != nil {
		if chameleon.AudioBoard {
			labels = append(labels, "audio_board")
		}

		if chamelonPeriphs := chameleon.GetPeripherals(); len(chamelonPeriphs) > 0 {
			labels = append(labels, "chameleon")
			for _, v := range chamelonPeriphs {
				lv := "chameleon:" + strings.ToLower(v.String())
				labels = append(labels, lv)
			}
		}
	}
	// - RPM
	// - ExternalCamera
	// - Audio
	if audio := dut.GetAudio(); audio != nil {
		if audio.GetAtrus() {
			labels = append(labels, "atrus")
		}
	}
	// - Wifi
	// - Touch

	if touch := dut.GetTouch(); touch != nil {
		if touch.GetMimo() {
			labels = append(labels, "mimo")
		}
	}
	// - Camerabox
	if camerabox := dut.GetCamerabox(); camerabox != nil {
		facing := camerabox.GetFacing()
		labels = append(labels, "camerabox_facing:"+strings.ToLower(facing.String()))
	}
	// - Cable
	// - Cellular
	return attrMap, labels
}

// appendAndroidLabels appends labels extracted from Android device.
func appendAndroidLabels(attrMap map[string]string, labels []string, dut *labapi.Dut_Android) (map[string]string, []string) {
	// Associated hostname.
	if hostname := dut.GetAssociatedHostname(); hostname != nil {
		if hostname.GetAddress() != "" {
			labels = append(labels, "associated_hostname:"+strings.ToLower(hostname.GetAddress()))
		}
	}
	// Android DUT name.
	if name := dut.GetName(); name != "" {
		labels = append(labels, "name:"+strings.ToLower(name))
	}
	// Android DUT serial number.
	if serialNumber := dut.GetSerialNumber(); serialNumber != "" {
		labels = append(labels, "serial_number:"+strings.ToLower(serialNumber))
	}
	// Android DUT model codename.
	if model := dut.GetDutModel().GetModelName(); model != "" {
		labels = append(labels, "model:"+strings.ToLower(model))
	}
	// Board name
	if board := dut.GetDutModel().GetBuildTarget(); board != "" {
		labels = append(labels, "board:"+strings.ToLower(board))
	}
	// We need to add os label for Android device in legacy workflow
	// as tests access device metadata through host-info instead of
	// dut topology directly, so they don't know the device type.
	labels = append(labels, "os:android")
	return attrMap, labels
}
