// Copyright 2021 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package main

import (
	"context"

	"github.com/maruel/subcommands"

	"infra/tools/migrator/internal/plugsupport"
)

func cmdStatus(opts cmdBaseOptions) *subcommands.Command {
	return &subcommands.Command{
		UsageLine: "status",
		ShortDesc: "Shows status of checked out repositories.",
		LongDesc: `Shows status of checked out repositories.

This command must be run within a migrator project after it has been scanned
via "scan" subcommand. It shows if the checkouts have uncommitted changes,
pending CLs, etc.
`,

		CommandRun: func() subcommands.CommandRun {
			ret := cmdStatusImpl{}
			ret.initFlags(cmdInitParams{
				opts:               opts,
				discoverProjectDir: true,
			})
			return &ret
		},
	}
}

type cmdStatusImpl struct {
	cmdBase
}

func (r *cmdStatusImpl) positionalRange() (min, max int) { return 0, 0 }

func (r *cmdStatusImpl) validateFlags(ctx context.Context, positionals []string, env subcommands.Env) error {
	return nil
}

func (r *cmdStatusImpl) execute(ctx context.Context) error {
	dump, err := plugsupport.ExecuteStatusCheck(ctx, r.projectDir)
	if err != nil {
		return err
	}
	prettyPrintRepoReport(dump)
	return nil
}

func (r *cmdStatusImpl) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	return r.doContextExecute(a, r, args, env)
}
