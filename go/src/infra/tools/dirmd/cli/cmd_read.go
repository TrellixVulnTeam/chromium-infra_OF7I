// Copyright 2021 The Chromium Authors. All rights reserved.
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
	"go.chromium.org/luci/common/system/signals"

	"infra/tools/dirmd"
	dirmdpb "infra/tools/dirmd/proto"
)

func cmdRead() *subcommands.Command {
	return &subcommands.Command{
		UsageLine: `read [DIR1 [DIR2]...]`,
		ShortDesc: "read metadata from the specified directories",
		Advanced:  true,
		LongDesc: text.Doc(`
			Read metadata from the specified directories and print to stdout or a file.

			Each directory must reside in a git checkout.
			One of the repos must be the root repo, while other repos must be its
			sub-repos. In other words, all git repos referred to by the directories
			must be subdirectories of one of the repos.
			The root dir of the root repo becomes the metadata root.

			Unless -form is sparse, only visible-to-git files are considered.
			Specifically, files outside of the repo are not read, as well as files
			matched by .gitignore files.
			The set of considered files is equivalent to "git ls-files <dir>".
			Note that the directory does not have to be the root of the repo,
			and multiple directories in the same repo are allowed.

			However, if -form is sparse, then metadata is read
			only from the specified directories and their ancestors up to the repo
			root. This is different from other forms:
			1) much fewer files are read, so it is faster.
			2) git-ignored files *are* read.

			The latter should not make a difference in the vast majority of cases
			because it is confusing to have git-ignored DIR_METADATA in the middle of
			the ancestry chain. Such a case might indicate that DIR_METADATA files are used incorrectly.
			This behavior can be changed, but it would come with a performance penalty.

			The output format is JSON form of chrome.dir_metadata.Mapping protobuf message.
		`),
		CommandRun: func() subcommands.CommandRun {
			r := &readRun{}
			r.RegisterOutputFlag()
			r.Flags.StringVar(&r.formString, "form", "original", text.Doc(`
				The form of the returned mapping.
				Valid values: "original", "reduced", "computed", "sparse", "full".
				See chrome.dir_meta enum MappingForm for their descriptions.
			`))
			return r
		},
	}
}

type readRun struct {
	baseCommandRun
	formString string
}

func (r *readRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	ctx := cli.GetContext(a, r, env)
	return r.done(ctx, r.run(ctx, args))
}

func (r *readRun) run(ctx context.Context, dirs []string) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	defer signals.HandleInterrupt(cancel)()

	if len(dirs) == 0 {
		dirs = []string{"."}
	}

	formInt, ok := dirmdpb.MappingForm_value[strings.ToUpper(r.formString)]
	if !ok {
		return errors.Reason("invalid value of -form").Err()
	}
	form := dirmdpb.MappingForm(formInt)

	mapping, err := dirmd.ReadMapping(ctx, form, dirs...)
	if err != nil {
		return err
	}
	return r.writeMapping(mapping)
}
