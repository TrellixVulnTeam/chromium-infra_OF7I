// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"

	"go.chromium.org/chromiumos/infra/proto/go/lab_platform"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/skylab_local_state"

	"infra/cros/cmd/skylab_local_state/location"
	"infra/cros/cmd/skylab_local_state/ufs"
	"infra/libs/cros/dutstate"
)

// DUT states that should never be overwritten.
var protectedDUTStates = map[string]bool{
	dutstate.NeedsDeploy.String():      true,
	dutstate.NeedsReplacement.String(): true,
}

type stateLoader func(autotestDir string, dutID string) (*lab_platform.DutState, error)

// Save subcommand: Update the bot state json file.
func Save(authOpts auth.Options) *subcommands.Command {
	return &subcommands.Command{
		UsageLine: "save -input_json /path/to/input.json",
		ShortDesc: "Update the DUT state json file.",
		LongDesc: `Update the DUT state json file.

(Re)Create the DUT state cache file using the state string from the input file
and provisionable labels and attributes from the host info file.
`,
		CommandRun: func() subcommands.CommandRun {
			c := &saveRun{}

			c.authFlags.Register(&c.Flags, authOpts)

			c.Flags.StringVar(&c.inputPath, "input_json", "", "Path to JSON SaveRequest to read.")
			return c
		},
	}
}

type saveRun struct {
	subcommands.CommandRunBase

	authFlags authcli.Flags

	inputPath string
}

func (c *saveRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.validateArgs(); err != nil {
		fmt.Fprintln(a.GetErr(), err.Error())
		c.Flags.Usage()
		return 1
	}

	err := c.innerRun(a, args, env)
	if err != nil {
		fmt.Fprintln(a.GetErr(), err.Error())
		return 1
	}
	return 0
}

func (c *saveRun) validateArgs() error {
	if c.inputPath == "" {
		return fmt.Errorf("-input_json not specified")
	}

	return nil
}

func (c *saveRun) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	var request skylab_local_state.SaveRequest
	if err := readJSONPb(c.inputPath, &request); err != nil {
		return err
	}

	if err := validateSaveRequest(&request); err != nil {
		return err
	}

	ctx := cli.GetContext(a, c, env)
	request, err := ensureNoProtectedStateOverwrite(ctx, &request, GetDutState)
	if err != nil {
		return err
	}

	i, err := getHostInfo(request.ResultsDir, request.DutName)

	if err != nil {
		return err
	}

	s := &lab_platform.DutState{
		State: request.DutState,
	}

	updateDutStateFromHostInfo(s, i)

	// Write DUT state into two files: one by DUT name, one by DUT ID.
	// TODO(crbug.com/994404): Stop saving the DUT ID-based state file.
	writeDutState(request.Config.AutotestDir, request.DutName, s)
	writeDutState(request.Config.AutotestDir, request.DutId, s)

	if request.GetSealResultsDir() {
		if err := sealResultsDir(request.ResultsDir); err != nil {
			return err
		}
	}

	return updateDutStateToUFS(ctx, &c.authFlags, request.Config.CrosUfsService, request.DutName, request.DutState)
}

func validateSaveRequest(request *skylab_local_state.SaveRequest) error {
	if request == nil {
		return fmt.Errorf("nil request")
	}

	var missingArgs []string

	if request.Config.GetAutotestDir() == "" {
		missingArgs = append(missingArgs, "autotest dir")
	}

	if request.ResultsDir == "" {
		missingArgs = append(missingArgs, "results dir")
	}

	if request.DutName == "" {
		missingArgs = append(missingArgs, "DUT hostname")
	}

	if request.DutId == "" {
		missingArgs = append(missingArgs, "DUT ID")
	}

	if request.DutState == "" {
		missingArgs = append(missingArgs, "DUT state")
	}

	if len(missingArgs) > 0 {
		return fmt.Errorf("no %s provided", strings.Join(missingArgs, ", "))
	}

	return nil
}

// ensureNoProtectedStateOverwrite checks if the current DUT state is in the
// list of states we should not overwrite, and modifies the request to leave
// the state untouched if so.
func ensureNoProtectedStateOverwrite(ctx context.Context, saveRequest *skylab_local_state.SaveRequest, getDutState stateLoader) (skylab_local_state.SaveRequest, error) {
	currentDUTState, err := getDutState(saveRequest.Config.AutotestDir, saveRequest.DutId)
	if err != nil {
		return skylab_local_state.SaveRequest{}, err
	}

	if protectedDUTStates[currentDUTState.State] {
		logging.Warningf(ctx, "Not saving requested DUT state %s, since current DUT state is %s, which should never be overwritten", saveRequest.DutState, currentDUTState.State)
		updatedSaveRequest := *saveRequest
		updatedSaveRequest.DutState = currentDUTState.State
		return updatedSaveRequest, nil
	}

	return *saveRequest, nil
}

// getHostInfo reads the host info from the store file.
func getHostInfo(resultsDir string, dutName string) (*skylab_local_state.AutotestHostInfo, error) {
	p := location.HostInfoFilePath(resultsDir, dutName)
	i := skylab_local_state.AutotestHostInfo{}

	if err := readJSONPb(p, &i); err != nil {
		return nil, errors.Annotate(err, "get host info").Err()
	}

	return &i, nil
}

// LabelSet provides the whitelist of labels that may change during provision.
// Only these labels can appear in the DUT state file.
var provisionableLabels = map[string]bool{
	"cros-version": true,
	"fwro-version": true,
	"fwrw-version": true,
}

var provisionableAttributes = map[string]bool{
	"job_repo_url":   true,
	"outlet_changed": true,
}

// updateDutStateFromHostInfo populates provisionable labels and provisionable
// attributes inside the DUT state with whitelisted labels and attributes from
// the host info.
func updateDutStateFromHostInfo(s *lab_platform.DutState, i *skylab_local_state.AutotestHostInfo) {
	// Should never happen.
	if s == nil {
		return
	}

	s.ProvisionableLabels = map[string]string{}

	for _, label := range i.GetLabels() {
		parts := strings.SplitN(label, ":", 2)
		if len(parts) != 2 {
			continue
		}
		if provisionableLabels[parts[0]] {
			s.ProvisionableLabels[parts[0]] = parts[1]
		}
	}

	s.ProvisionableAttributes = map[string]string{}

	for attribute, value := range i.GetAttributes() {
		if provisionableAttributes[attribute] {
			s.ProvisionableAttributes[attribute] = value
		}
	}
}

// writeDutState writes a JSON-encoded DutState proto to the cache file inside
// the autotest directory.
func writeDutState(autotestDir string, fileID string, s *lab_platform.DutState) error {
	p := location.CacheFilePath(autotestDir, fileID)

	if err := writeJSONPb(p, s); err != nil {
		return errors.Annotate(err, "write DUT state").Err()
	}

	return nil
}

const gsOffloaderMarker = ".ready_for_offload"

// sealResultsDir drops a special timestamp file in the results directory
// notifying gs_offloader to offload the directory. The results directory
// should not be touched once sealed. This should not be called on an
// already sealed results directory.
func sealResultsDir(dir string) error {
	ts := []byte(fmt.Sprintf("%d", time.Now().Unix()))
	tsFile := filepath.Join(dir, gsOffloaderMarker)
	_, err := os.Stat(tsFile)
	if err == nil {
		return fmt.Errorf("seal results dir %s: already sealed", dir)
	}
	if !os.IsNotExist(err) {
		return fmt.Errorf("seal results dir %s: encountered corrupted file %s", dir, tsFile)
	}
	if err := ioutil.WriteFile(tsFile, ts, 0666); err != nil {
		return errors.Annotate(err, "seal results dir %s", dir).Err()
	}
	return nil
}

// updateDutStateToUFS send DUT state to the UFS service.
func updateDutStateToUFS(ctx context.Context, authFlags *authcli.Flags, crosUfsService, dutName, dutState string) error {
	ufsClient, err := ufs.NewClient(ctx, crosUfsService, authFlags)
	if err != nil {
		return errors.Annotate(err, "save local state").Err()
	}
	err = dutstate.Update(ctx, ufsClient, dutName, dutstate.State(dutState))
	if err != nil {
		return errors.Annotate(err, "save local state").Err()
	}
	return nil
}
