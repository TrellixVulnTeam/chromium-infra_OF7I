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

			Unless -form is sparse, the output includes metadata of ancestors and
			descendants of the specified directories.
			In the sparse form, metadata of only the specified directories is
			returned, which is usually much faster.

			Descendants of the specified directories are discovered using
			"git ls-files <dir>" and not FS walk.
			This means files outside of the repo are ignored, as well as files
			matched by .gitignore files.
			Note that when reading ancestors of the specified directories,
			the .gitignore files are not respected.
			This inconsistency should not make a difference in
			the vast majority of cases because it is confusing to have
			git-ignored DIR_METADATA in the middle of the ancestry chain.
			Such a case might indicate that DIR_METADATA files are used incorrectly.
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
				https://source.chromium.org/chromium/infra/infra/+/main:go/src/infra/tools/dirmd/proto/mapping.proto;l=51?q=mappingform
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
