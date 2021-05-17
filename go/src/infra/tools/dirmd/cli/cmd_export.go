// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cli

import (
	"context"
	"strings"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/data/text"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/flag"
	"go.chromium.org/luci/common/system/signals"

	"infra/tools/dirmd"
	dirmdpb "infra/tools/dirmd/proto"
)

func cmdExport() *subcommands.Command {
	return &subcommands.Command{
		UsageLine: `export`,
		ShortDesc: "export metadata from a directory tree",
		Advanced:  true,
		LongDesc: text.Doc(`
			DEPRECATED: use "dirmd read" instead.

			Export metadata from a directory tree to stdout or to a file.

			The output format is JSON form of chrome.dir_metadata.Mapping protobuf
			message.
		`),
		CommandRun: func() subcommands.CommandRun {
			r := &exportRun{}
			r.RegisterOutputFlag()
			r.Flags.Var(flag.StringSlice(&r.roots), "root", text.Doc(`
				The directory with metadata files. May be specified multiple times.

				Each directory must reside in a git checkout. Only visible-to-git files
				are considered. Specifically, files outside of the repo are not read,
				as well as files matched by .gitignore files. The set of considered files
				is equivalent to "git ls-files <dir>".
				Note that the directory does not have to be the root of the git repo,
				and multiple directories in the same repo are allowed.

				One of the repos must be the root repo, while other repos must be its
				sub-repos. In other words, all git repos referred to by the directories must
				be subdirectories of one of the repos.
				The root dir of the root repo becomes the metadata root.
			`))
			r.Flags.StringVar(&r.formString, "form", "original", text.Doc(`
				The form of the returned mapping.
				Valid values: "original", "reduced", "computed", "full".
				See chrome.dir_meta.MappingForm protobuf enum for their descriptions.
			`))
			return r
		},
	}
}

type exportRun struct {
	baseCommandRun
	roots      []string
	formString string
}

func (r *exportRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	ctx := cli.GetContext(a, r, env)
	return r.done(ctx, r.run(ctx, args))
}

func (r *exportRun) run(ctx context.Context, args []string) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	defer signals.HandleInterrupt(cancel)()

	if len(args) != 0 {
		return errors.Reason("unexpected positional arguments: %q", args).Err()
	}
	if len(r.roots) == 0 {
		r.roots = []string{"."}
	}

	formInt, ok := dirmdpb.MappingForm_value[strings.ToUpper(r.formString)]
	if !ok {
		return errors.Reason("invalid value of -form").Err()
	}
	form := dirmdpb.MappingForm(formInt)

	mapping, err := dirmd.ReadMapping(ctx, form, r.roots...)
	if err != nil {
		return err
	}
	return r.writeMapping(mapping)
}
