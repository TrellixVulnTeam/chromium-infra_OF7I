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

	"infra/cros/cmd/phosphorus/internal/botcache"
	"infra/cros/cmd/phosphorus/internal/skylab_local_state/location"
	"infra/cros/cmd/phosphorus/internal/skylab_local_state/ufs"
	"infra/libs/skylab/inventory"
	"infra/libs/skylab/inventory/autotest/labels"
	ufsapi "infra/unifiedfleet/api/v1/rpc"
	ufsutil "infra/unifiedfleet/app/util"

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
		ResultsDir: resultsDir,
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

// getDutInfo fetches the DUT inventory entry from UFS.
func getDutInfo(ctx context.Context, client ufsapi.FleetClient, dutName string) (*inventory.DeviceUnderTest, error) {
	osctx := ufs.SetupContext(ctx, ufsutil.OSNamespace)
	req := &ufsapi.GetChromeOSDeviceDataRequest{
		Hostname: dutName,
	}
	resp, err := client.GetChromeOSDeviceData(osctx, req)
	if err != nil {
		return nil, errors.Annotate(err, fmt.Sprintf("get DUT info for %s", dutName)).Err()
	}
	return resp.GetDutV1(), nil
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
