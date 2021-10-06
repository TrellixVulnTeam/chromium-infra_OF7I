// Copyright 2020 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package main

import (
	"context"
	"os"

	"github.com/maruel/subcommands"

	"go.chromium.org/luci/common/errors"

	"infra/tools/migrator/internal/plugsupport"
)

func cmdScan(opts cmdBaseOptions) *subcommands.Command {
	return &subcommands.Command{
		UsageLine: "scan",
		ShortDesc: "Scans the current state of the world and checks out non-compliant projects.",
		LongDesc: `Scans current state of data in luci-config.

This command must be run within a migrator project. The scan will run all LUCI
projects through the project's plugin. Note that this scans the state of the
files in the luci-config service, NOT the state of the files in the migrator
project.

If the plugin's 'FindProblems' function is undefined OR it makes any Report
calls, this will ensure that the project is checked out locally on disk. If
FindProblems is defined and does NOT make any Report calls, this will inform
you that the checkout can be removed (pass '-clean' to automatically delete
them).

If scan does a new checkout, and the plugin defines ApplyFix, it will be invoked
once on the checked-out project.

If a checkout already exists on disk, this will NOT attempt to update it; It's
recommended to use standard git tooling to pull/rebase/etc. If you really want
a new checkout, you can delete the checked-out project and run 'scan' again
to get a fresh top-of-tree version.
`,

		CommandRun: func() subcommands.CommandRun {
			ret := cmdScanImpl{}
			ret.initFlags(opts)
			ret.Flags.BoolVar(&ret.squeaky, "squeaky", false,
				"If set in conjunction with `clean`, will checkout all repos from scratch.")
			ret.Flags.BoolVar(&ret.clean, "clean", false,
				"If set, will automatically delete project checkouts which have no reported problems.")

			ret.Flags.BoolVar(&ret.reapply, "re-apply", false,
				"If set, will re-run ApplyFix, even if no new checkout was made.")
			return &ret
		},
	}
}

type cmdScanImpl struct {
	cmdBase

	projectDir plugsupport.ProjectDir
	squeaky    bool
	clean      bool
	reapply    bool
}

func (r *cmdScanImpl) positionalRange() (min, max int) { return 0, 0 }

func (r *cmdScanImpl) validateFlags(ctx context.Context, positionals []string, env subcommands.Env) error {
	if r.squeaky && !r.clean {
		return errors.New("you can't be squeaky without being clean! (pass -clean flag)")
	}

	wd, err := os.Getwd()
	if err != nil {
		return errors.Annotate(err, "getting working directory").Err()
	}
	r.projectDir, err = plugsupport.FindProjectRoot(wd)
	return err
}

func (r *cmdScanImpl) execute(ctx context.Context) error {
	return invokePlugin(ctx, r.projectDir, plugsupport.Command{
		Action:        "scan",
		ContextConfig: r.contextConfig,
		ScanConfig: plugsupport.ScanConfig{
			Squeaky: r.squeaky,
			Clean:   r.clean,
			Reapply: r.reapply,
		},
	})
}

func (r *cmdScanImpl) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	return r.doContextExecute(a, r, args, env)
}
