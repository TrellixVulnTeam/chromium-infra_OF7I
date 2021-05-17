// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cli

import (
	"context"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/data/text"

	"infra/tools/dirmd"
	dirmdpb "infra/tools/dirmd/proto"
)

func cmdCompute() *subcommands.Command {
	return &subcommands.Command{
		UsageLine: `compute -root ROOT TARGET1 [TARGET2...]`,
		ShortDesc: "compute metadata for the given target directories",
		Advanced:  true,
		LongDesc: text.Doc(`
			DEPRECATED: use "dirmd read" instead.

			Compute metadata for the given target directories.

			The output format is JSON form of chrome.dir_meta.Mapping protobuf
			message, same as "export" subcommand.
			The returned mapping contains entries only for the explicitly
			specified target dirs. Each entry includes inherited metadata.

			Unlike export subcommand, this subcommand reads metadata from only the
			targets and their ancestors up to the ROOT. This is different from export
			subcommand which uses git-ls-files:
			1) compute subcommand normally reads much fewer files.
			2) compute subcommand does not respect git-ignored files.

			The latter indicates a slightly different semantics, but this should not
			make any difference in the vast majority of cases because it is confusing to
			have git-ignored DIR_METADATA in the middle of the ancestry chain, which
			might indicate that DIR_METADATA files are used incorrectly.
			This can be fixed, but it would come with a performance penalty.
		`),
		CommandRun: func() subcommands.CommandRun {
			r := &computeRun{}
			r.RegisterOutputFlag()
			r.Flags.String("root", "", "Deprecated, ignored")
			return r
		},
	}
}

type computeRun struct {
	baseCommandRun
}

func (r *computeRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	ctx := cli.GetContext(a, r, env)
	return r.done(ctx, r.run(ctx, args))
}

func (r *computeRun) run(ctx context.Context, targets []string) error {
	mapping, err := dirmd.ReadMapping(ctx, dirmdpb.MappingForm_SPARSE, targets...)
	if err != nil {
		return err
	}
	return r.writeMapping(mapping)
}
