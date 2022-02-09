// Copyright 2022 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package main

import (
	"context"

	"github.com/maruel/subcommands"

	"infra/tools/migrator/internal/plugsupport"
)

func cmdRebase(opts cmdBaseOptions) *subcommands.Command {
	return &subcommands.Command{
		UsageLine: "rebase",
		ShortDesc: `Fetches the most recent repo state and rebases the CL on top`,
		LongDesc: `Fetches the most recent repo state and rebases the CL on top.

		Rebases only clean checkouts of "fix_config" local branch. Any conflicts are
		left unresolved and surface in the status report. They must be fixed
		manually. Additionally you may need to run "scan --re-apply" after rebasing
		to make sure all migrations are applied on top of the rebased state.
		`,

		CommandRun: func() subcommands.CommandRun {
			ret := cmdRebaseImpl{}
			ret.initFlags(cmdInitParams{
				opts:               opts,
				discoverProjectDir: true,
			})
			return &ret
		},
	}
}

type cmdRebaseImpl struct {
	cmdBase
}

func (r *cmdRebaseImpl) positionalRange() (min, max int) { return 0, 0 }

func (r *cmdRebaseImpl) validateFlags(ctx context.Context, positionals []string, env subcommands.Env) error {
	return nil
}

func (r *cmdRebaseImpl) execute(ctx context.Context) error {
	dump, err := plugsupport.ExecuteRebase(ctx, r.projectDir)
	if err != nil {
		return err
	}
	prettyPrintRepoReport(dump)
	return nil
}

func (r *cmdRebaseImpl) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	return r.doContextExecute(a, r, args, env)
}
