// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmd

import (
	"fmt"
	"infra/cmd/cros_test_platform/internal/backfill"

	"github.com/maruel/subcommands"

	"go.chromium.org/chromiumos/infra/proto/go/test_platform/steps"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"
)

// ComputeBackfill subcommand: Create the backfill request for the current run.
var ComputeBackfill = &subcommands.Command{
	UsageLine: "compute-backfill -input_json /path/to/input.json -output_json /path/to/output.json",
	ShortDesc: "Compute the backfill requests for the current run.",
	LongDesc:  `Compute the backfill requests for the current run.`,
	CommandRun: func() subcommands.CommandRun {
		c := &computeBackfillRun{}
		c.Flags.StringVar(&c.inputPath, "input_json", "", "Path that contains JSON encoded test_platform.steps.ComputeBackfillRequests")
		c.Flags.StringVar(&c.outputPath, "output_json", "", "Path where JSON encoded test_platform.steps.ComputeBackfillResponses should be written.")
		return c
	},
}

type computeBackfillRun struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags

	inputPath  string
	outputPath string
}

func (c *computeBackfillRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	err := c.innerRun(a, args, env)
	if err != nil {
		fmt.Fprintf(a.GetErr(), "%s\n", err)
	}
	return exitCode(err)
}

func (c *computeBackfillRun) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	if err := c.processCLIArgs(args); err != nil {
		return err
	}
	ctx := cli.GetContext(a, c, env)
	ctx = setupLogging(ctx)

	requests, err := c.readRequests()
	if err != nil {
		return err
	}
	resps := make(map[string]*steps.ComputeBackfillResponse)
	for t, req := range requests {
		resp, err := c.computeFor(req)
		if err != nil {
			return err
		}
		resps[t] = resp
	}
	return c.writeResponses(resps)
}

func (c *computeBackfillRun) processCLIArgs(args []string) error {
	if len(args) > 0 {
		return errors.Reason("have %d positional args, want 0", len(args)).Err()
	}
	if c.inputPath == "" {
		return errors.Reason("-input_json not specified").Err()
	}
	if c.outputPath == "" {
		return errors.Reason("-output_json not specified").Err()
	}
	return nil
}

func (c *computeBackfillRun) readRequests() (map[string]*steps.ComputeBackfillRequest, error) {
	var rs steps.ComputeBackfillRequests
	if err := readRequest(c.inputPath, &rs); err != nil {
		return nil, err
	}
	return rs.TaggedRequests, nil
}

func (c *computeBackfillRun) writeResponses(resps map[string]*steps.ComputeBackfillResponse) error {
	return writeResponse(c.outputPath, &steps.ComputeBackfillResponses{
		TaggedResponses: resps,
	})
}

func (c *computeBackfillRun) computeFor(r *steps.ComputeBackfillRequest) (*steps.ComputeBackfillResponse, error) {
	resp, err := backfill.Compute(r)
	if err != nil {
		return nil, errors.Annotate(err, "compute for %s", r).Err()
	}
	return resp, nil
}
