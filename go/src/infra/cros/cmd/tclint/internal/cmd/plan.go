// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmd

import (
	"context"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"

	"infra/cmdsupport/cmdlib"

	plan "go.chromium.org/chromiumos/infra/proto/go/test/plan/v1"
)

// Plan subcommand: Lint test plan.
var Plan = &subcommands.Command{
	UsageLine: "plan [FLAGS...] INPUT_FILE_GLOB [INPUT_FILE_GLOB...]",
	ShortDesc: "Lint Chrome OS test plan specification.",
	LongDesc: `Lint a (complete) specification of a Chrome OS test plan.

The test plan must be specified as a plan.Specification payload as defined at
https://chromium.googlesource.com/chromiumos/infra/proto/+/refs/heads/master/src/test/plan/v1/plan.proto

The lint includes some global uniqueness checks. Thus, validation may be
incomplete if a partial test plan specification is provided as input.

The test plan specification may be split over multiple files and directories,
provided via glob patterns in the positional arguments.`,
	CommandRun: func() subcommands.CommandRun {
		c := &planRun{}
		c.Flags.BoolVar(
			&c.binaryFormat,
			"binary",
			false,
			`Decode input protobuf payload from the binary wire format.
By default, input is assumed to be encoded as JSON.`,
		)
		return c
	},
}

type planRun struct {
	subcommands.CommandRunBase
	binaryFormat bool
}

// Run implements the subcommands.CommandRun interface.
func (c *planRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	ctx := cli.GetContext(a, c, env)
	ctx = logging.SetLevel(ctx, logging.Info)
	if err := c.innerRun(ctx, args); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

type planFile struct {
	path    string
	payload *plan.Specification
	errs    errors.MultiError
}

func (c *planRun) innerRun(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return errors.Reason("no input files").Err()
	}
	if _, err := c.load(args); err != nil {
		return err
	}
	return errors.Reason("not implemented").Err()
}

func (c *planRun) load(globs []string) ([]*planFile, error) {
	resp := make([]*planFile, 0, len(globs))
	err := forEachFile(
		globs,
		func(f string) {
			pf := &planFile{
				path: f,
			}
			if p, err := c.loadOne(f); err != nil {
				pf.errs = errors.NewMultiError(err)
			} else {
				pf.payload = p
			}
			resp = append(resp, pf)
		},
	)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *planRun) loadOne(path string) (*plan.Specification, error) {
	var p plan.Specification
	var err error
	if c.binaryFormat {
		err = loadFromBinary(path, &p)
	} else {
		err = loadFromJSON(path, &p)
	}
	return &p, err
}
