// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cli

import (
	"context"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/data/text"
	"go.chromium.org/luci/common/errors"

	"infra/tools/dirmeta"
)

func cmdCompute() *subcommands.Command {
	return &subcommands.Command{
		UsageLine: `compute -root ROOT TARGET1 [TARGET2...]`,
		ShortDesc: "compute metadata for the given target directories",
		LongDesc: text.Doc(`
			Compute metadata for the given target directories.

			The output format is JSON form of chrome.dir_meta.Mapping protobuf
			message, same as "export" subcommand.
			The returned mapping contains entries only for the explicitly
			specified target dirs. Each entry includes inherited metadata.
		`),
		CommandRun: func() subcommands.CommandRun {
			r := &computeRun{}
			r.RegisterOutputFlag()

			// -root does not have a default intentionally, otherwise it is easy
			// to run `dirmeta compute` from no a repo root and notice the problem.
			r.Flags.StringVar(&r.root, "root", "", "Path to the root directory")
			return r
		},
	}
}

type computeRun struct {
	baseCommandRun
	root string
}

func (r *computeRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	ctx := cli.GetContext(a, r, env)
	return r.done(ctx, r.run(ctx, args))
}

func (r *computeRun) run(ctx context.Context, targets []string) error {
	if r.root == "" {
		return errors.Reason("-root is required").Err()
	}

	mapping, err := dirmeta.ReadComputed(r.root, targets...)
	if err != nil {
		return err
	}
	return r.writeMapping(mapping)
}
