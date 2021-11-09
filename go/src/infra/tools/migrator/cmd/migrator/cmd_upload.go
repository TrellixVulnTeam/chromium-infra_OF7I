// Copyright 2021 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package main

import (
	"context"

	"github.com/maruel/subcommands"

	"infra/tools/migrator/internal/plugsupport"
)

func cmdUpload(opts cmdBaseOptions) *subcommands.Command {
	return &subcommands.Command{
		UsageLine: "upload",
		ShortDesc: `Uploads pending changes using "git cl upload".`,
		LongDesc: `Uploads pending change using "git cl upload".

Takes the commit message from commit-message.txt file in the migrator project
directory. Skips running upload hooks. By default assigns reviewers based on
OWNERS files.
`,

		CommandRun: func() subcommands.CommandRun {
			ret := cmdUploadImpl{}
			ret.initFlags(cmdInitParams{
				opts:               opts,
				discoverProjectDir: true,
			})
			ret.Flags.StringVar(&ret.reviewers, "reviewers", "", "Comma-separated list of reviewers (default: repo owners).")
			ret.Flags.StringVar(&ret.cc, "cc", "", "Comma-separated list of emails to CC (optional).")
			ret.Flags.BoolVar(&ret.force, "force", false, "Initiate upload even if nothing has changed.")
			return &ret
		},
	}
}

type cmdUploadImpl struct {
	cmdBase

	reviewers string
	cc        string
	force     bool
}

func (r *cmdUploadImpl) positionalRange() (min, max int) { return 0, 0 }

func (r *cmdUploadImpl) validateFlags(ctx context.Context, positionals []string, env subcommands.Env) error {
	return nil
}

func (r *cmdUploadImpl) execute(ctx context.Context) error {
	dump, err := plugsupport.ExecuteUpload(ctx, r.projectDir, plugsupport.UploadOptions{
		Reviewers: r.reviewers,
		CC:        r.cc,
		Force:     r.force,
	})
	if err != nil {
		return err
	}
	prettyPrintRepoReport(dump)
	return nil
}

func (r *cmdUploadImpl) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	return r.doContextExecute(a, r, args, env)
}
