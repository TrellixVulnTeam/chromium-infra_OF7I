// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/maruel/subcommands"
	"google.golang.org/protobuf/encoding/protojson"

	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/data/text"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/system/signals"
	sinkpb "go.chromium.org/luci/resultdb/sink/proto/v1"

	"infra/tools/dirmd"
	dirmdpb "infra/tools/dirmd/proto"
)

func cmdLocationTags() *subcommands.Command {
	return &subcommands.Command{
		UsageLine: `location-tags`,
		ShortDesc: "generate test location tags from a directory tree",
		LongDesc: text.Doc(`
			Generate test location tags from a directory tree to stdout or to a file.

			The output format is JSON form of luci.resultsink.v1.LocationTags protobuf
			message.
		`),
		CommandRun: func() subcommands.CommandRun {
			r := &tagRun{}
			r.RegisterOutputFlag()
			r.Flags.StringVar(&r.root, "root", ".", "Path to the root directory")
			r.Flags.StringVar(&r.repo, "repo", "", "Gitiles URL as the identifier for a repo. In the format of https://<host>/<project>. Must not end with .git.")
			return r
		},
	}
}

type tagRun struct {
	baseCommandRun
	root string
	repo string
}

func (r *tagRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	ctx := cli.GetContext(a, r, env)
	return r.done(ctx, r.run(ctx, args))
}

func (r *tagRun) run(ctx context.Context, args []string) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	defer signals.HandleInterrupt(cancel)()

	if len(args) != 0 {
		return errors.Reason("unexpected positional arguments: %q", args).Err()
	}

	if err := r.validate(); err != nil {
		return err
	}
	mapping, err := dirmd.ReadMapping(ctx, dirmdpb.MappingForm_REDUCED, r.root)
	if err != nil {
		return err
	}
	tags, err := dirmd.ToLocationTags(mapping)
	if err != nil {
		return err
	}
	return r.writeTags(&sinkpb.LocationTags{
		Repos: map[string]*sinkpb.LocationTags_Repo{
			r.repo: tags,
		},
	})
}

func (r *tagRun) validate() error {
	switch {
	case r.repo == "":
		return fmt.Errorf("-repo is required")
	case strings.HasSuffix(r.repo, ".git"):
		return fmt.Errorf("-repo must not end with .git")
	default:
		return nil
	}
}

func (r *tagRun) writeTags(tags *sinkpb.LocationTags) error {
	data, err := protojson.Marshal(tags)
	if err != nil {
		return err
	}

	return r.writeTextOutput(data)
}
