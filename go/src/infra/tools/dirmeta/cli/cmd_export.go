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

	"infra/tools/dirmeta"
	dirmetapb "infra/tools/dirmeta/proto"
)

func cmdExport() *subcommands.Command {
	return &subcommands.Command{
		UsageLine: `export`,
		ShortDesc: "export metadata from a directory tree",
		LongDesc: text.Doc(`
			Export metadata from a directory tree to stdout or to a file.

			The output format is JSON form of chrome.dir_metadata.Mapping protobuf
			message.
		`),
		CommandRun: func() subcommands.CommandRun {
			r := &exportRun{}
			r.RegisterOutputFlag()
			r.Flags.StringVar(&r.root, "root", ".", "Path to the root directory")
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
	root       string
	formString string
}

func (r *exportRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	ctx := cli.GetContext(a, r, env)
	return r.done(ctx, r.run(ctx, args))
}

func (r *exportRun) run(ctx context.Context, args []string) error {
	if len(args) != 0 {
		return errors.Reason("unexpected positional arguments: %q", args).Err()
	}

	formInt, ok := dirmetapb.MappingForm_value[strings.ToUpper(r.formString)]
	if !ok {
		return errors.Reason("invalid value of -form").Err()
	}
	form := dirmetapb.MappingForm(formInt)

	mapping, err := dirmeta.ReadMapping(r.root, form)
	if err != nil {
		return err
	}
	return r.writeMapping(mapping)
}
