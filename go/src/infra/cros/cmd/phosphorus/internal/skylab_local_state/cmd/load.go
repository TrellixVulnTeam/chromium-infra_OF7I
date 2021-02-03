// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/grpc/prpc"
	"go.chromium.org/luci/lucictx"

	fleet "infra/appengine/cros/lab_inventory/api/v1"
	"infra/cros/cmd/phosphorus/internal/botcache"
	"infra/cros/cmd/phosphorus/internal/skylab_local_state/location"
	"infra/libs/skylab/inventory"
	"infra/libs/skylab/inventory/autotest/labels"

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

	client, err := newInventoryClient(ctx, request.Config.CrosInventoryService, &c.authFlags)
	if err != nil {
		return err
	}

	dut, err := getDutInfo(ctx, client, request.DutName)
	if err != nil {
		return err
	}

	bcs := botcache.Store{
		CacheDir: request.Config.AutotestDir,
		Name:     request.DutName,
	}
	dutState, err := bcs.Load()
	if err != nil {
		return err
	}

	hostInfo := getFullHostInfo(dut, dutState)

	resultsDir := location.ResultsDir(request.Config.AutotestDir, request.RunId, request.TestId)

	if err := writeHostInfo(resultsDir, request.DutName, hostInfo); err != nil {
		return err
	}

	response := skylab_local_state.LoadResponse{
		ProvisionableLabels: dutState.ProvisionableLabels,
		ResultsDir:          resultsDir,
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

// newInventoryClient creates an cros inventory service client.
func newInventoryClient(ctx context.Context, crosInventoryService string, authFlags *authcli.Flags) (fleet.InventoryClient, error) {
	authOpts, err := authFlags.Options()
	if err != nil {
		return nil, errors.Annotate(err, "create new inventory client").Err()
	}

	authCtx, err := lucictx.SwitchLocalAccount(ctx, "system")
	if err == nil {
		// If there's a system account use it (the case of running on Swarming).
		// Otherwise default to user credentials (the local development case).
		ctx = authCtx
		authOpts.Method = auth.LUCIContextMethod
	}

	a := auth.NewAuthenticator(ctx, auth.SilentLogin, authOpts)

	httpClient, err := a.Client()
	if err != nil {
		return nil, errors.Annotate(err, "create new inventory client").Err()
	}

	pc := prpc.Client{
		C:    httpClient,
		Host: crosInventoryService,
	}

	return fleet.NewInventoryPRPCClient(&pc), nil
}

// getDutInfo fetches the DUT inventory entry from the admin service.
func getDutInfo(ctx context.Context, client fleet.InventoryClient, dutName string) (*inventory.DeviceUnderTest, error) {
	devID := &fleet.DeviceID{Id: &fleet.DeviceID_Hostname{Hostname: dutName}}
	req := &fleet.GetCrosDevicesRequest{
		Ids: []*fleet.DeviceID{devID},
	}
	resp, err := client.GetCrosDevices(ctx, req)
	if err != nil {
		return nil, errors.Annotate(err, fmt.Sprintf("get DUT info for %s", dutName)).Err()
	}

	devices := resp.GetData()
	if len(devices) == 1 {
		dut, err := fleet.AdaptToV1DutSpec(devices[0])
		if err != nil {
			return nil, errors.Annotate(err, fmt.Sprintf("get DUT info for %s", dutName)).Err()
		}
		return dut, nil
	} else if len(devices) > 1 {
		return nil, errors.Reason("get DUT info for %s: more than 1 DUT was returned in passed list!", dutName).Err()
	}

	failedDevies := resp.GetFailedDevices()
	if len(failedDevies) == 1 {
		return nil, errors.Reason(failedDevies[0].ErrorMsg).Err()
	} else if len(failedDevies) > 1 {
		return nil, errors.Reason("get DUT info for %s: more than 1 DUT was returned in failed list!", dutName).Err()
	}

	return nil, errors.Reason("get DUT info for %s: no data responded in either passed or failed list!", dutName).Err()
}

// hostInfoFromDutInfo extracts attributes and labels from an inventory
// entry and assembles them into a host info file proto.
func hostInfoFromDutInfo(dut *inventory.DeviceUnderTest) *skylab_local_state.AutotestHostInfo {
	i := skylab_local_state.AutotestHostInfo{
		Attributes:        map[string]string{},
		Labels:            labels.Convert(dut.Common.GetLabels()),
		SerializerVersion: currentSerializerVersion,
	}

	for _, attribute := range dut.Common.GetAttributes() {
		i.Attributes[attribute.GetKey()] = attribute.GetValue()
	}
	return &i
}

// addDutStateToHostInfo adds provisionable labels and attributes from
// the bot state to the host info labels and attributes.
func addDutStateToHostInfo(hostInfo *skylab_local_state.AutotestHostInfo, dutState *lab_platform.DutState) {
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
		return errors.Annotate(err, "write host info").Err()
	}

	return nil
}

// getFullHostInfo aggregates data from local and admin services state into one hostinfo object
func getFullHostInfo(dut *inventory.DeviceUnderTest, dutState *lab_platform.DutState) *skylab_local_state.AutotestHostInfo {
	hostInfo := hostInfoFromDutInfo(dut)

	addDutStateToHostInfo(hostInfo, dutState)
	return hostInfo
}
