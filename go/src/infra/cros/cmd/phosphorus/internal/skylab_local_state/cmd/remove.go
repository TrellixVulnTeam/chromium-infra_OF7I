// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/maruel/subcommands"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/skylab_local_state"
	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/auth/client/authcli"

	"infra/cros/cmd/phosphorus/internal/skylab_local_state/location"
)

// Remove subcommand: Remove the results parent directory.
func Remove(authOpts auth.Options) *subcommands.Command {
	return &subcommands.Command{
		UsageLine: "remove -input_json /path/to/input.json",
		ShortDesc: "remove the results parent directory",
		LongDesc:  "Remove the results parent directory.",
		CommandRun: func() subcommands.CommandRun {
			c := &removeRun{}
			c.authFlags.Register(&c.Flags, authOpts)
			c.Flags.StringVar(&c.inputPath, "input_json", "", "Path to JSON SaveRequest to read.")
			return c
		},
	}
}

type removeRun struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
	inputPath string
}

func (c *removeRun) Run(a subcommands.Application, _ []string, _ subcommands.Env) int {
	if err := c.validateArgs(); err != nil {
		fmt.Fprintln(a.GetErr(), err.Error())
		c.Flags.Usage()
		return 1
	}
	if err := c.innerRun(); err != nil {
		fmt.Fprintln(a.GetErr(), err.Error())
		return 1
	}
	return 0
}

func (c *removeRun) validateArgs() error {
	if c.inputPath == "" {
		return fmt.Errorf("-input_json not specified")
	}
	return nil
}

func (c *removeRun) innerRun() error {
	var request skylab_local_state.RemoveRequest
	if err := readJSONPb(c.inputPath, &request); err != nil {
		return err
	}
	if err := validateRemoveRequest(&request); err != nil {
		return err
	}
	return removeResultsParentDir(request.Config.AutotestDir, request.RunId)
}

// removeResultsParentDir removes the parent directory containing the autotest
// results for the given Swarming run ID.
func removeResultsParentDir(autotestDir, runID string) error {
	dir := location.ResultsParentDir(autotestDir, runID)
	return os.RemoveAll(dir)
}

func validateRemoveRequest(request *skylab_local_state.RemoveRequest) error {
	if request == nil {
		return fmt.Errorf("nil request")
	}

	var missingArgs []string

	if request.Config.GetAutotestDir() == "" {
		missingArgs = append(missingArgs, "autotest dir")
	}

	if request.RunId == "" {
		missingArgs = append(missingArgs, "Swarming run ID")
	}

	if len(missingArgs) > 0 {
		return fmt.Errorf("no %s provided", strings.Join(missingArgs, ", "))
	}

	return nil
}
