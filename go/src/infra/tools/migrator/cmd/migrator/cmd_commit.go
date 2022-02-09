// Copyright 2022 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package main

import (
	"context"

	"github.com/maruel/subcommands"

	"infra/tools/migrator/internal/plugsupport"
)

func cmdCommit(opts cmdBaseOptions) *subcommands.Command {
	return &subcommands.Command{
		UsageLine: "commit",
		ShortDesc: `Commits pending changes into a local commit`,
		LongDesc: `Commits pending changes into a local commit.

Takes the commit message from commit-message.txt file in the migrator project
directory.
`,

		CommandRun: func() subcommands.CommandRun {
			ret := cmdCommitImpl{}
			ret.initFlags(cmdInitParams{
				opts:               opts,
				discoverProjectDir: true,
			})
			return &ret
		},
	}
}

type cmdCommitImpl struct {
	cmdBase
}

func (r *cmdCommitImpl) positionalRange() (min, max int) { return 0, 0 }

func (r *cmdCommitImpl) validateFlags(ctx context.Context, positionals []string, env subcommands.Env) error {
	return nil
}

func (r *cmdCommitImpl) execute(ctx context.Context) error {
	dump, err := plugsupport.ExecuteCommit(ctx, r.projectDir)
	if err != nil {
		return err
	}
	prettyPrintRepoReport(dump)
	return nil
}

func (r *cmdCommitImpl) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	return r.doContextExecute(a, r, args, env)
}
