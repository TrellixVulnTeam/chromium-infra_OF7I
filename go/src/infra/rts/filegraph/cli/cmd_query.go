// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cli

import (
	"errors"
	"fmt"

	"github.com/maruel/subcommands"

	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/data/text"

	"infra/rts/filegraph"
)

var cmdQuery = &subcommands.Command{
	UsageLine: `query [flags] SOURCE_FILE [SOURCE_FILE...]`,
	ShortDesc: "print graph files in the distance-ascending order",
	LongDesc: text.Doc(`
		Print graph files in the distance-ascending order from SOURCE_FILEs.

		Each output line has format "<distance> <filename>",
		where the filename is forward-slash-separated and has "//" prefix.
		Example: "0.4 //foo/bar.cpp".

		All SOURCE_FILEs must be in the same git repository.
		Does not print unreachable files.
	`),
	CommandRun: func() subcommands.CommandRun {
		r := &queryRun{}
		r.gitGraph.RegisterFlags(&r.Flags)
		return r
	},
}

type queryRun struct {
	baseCommandRun
	gitGraph
}

func (r *queryRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	ctx := cli.GetContext(a, r, env)
	if err := r.gitGraph.Validate(); err != nil {
		return r.done(err)
	}
	if len(args) == 0 {
		return r.done(errors.New("expected filenames as positional arguments"))
	}

	sources, err := r.loadSyncedNodes(ctx, args...)
	if err != nil {
		return r.done(err)
	}

	r.query(sources...).Run(func(sp *filegraph.ShortestPath) bool {
		fmt.Printf("%.2f %s\n", sp.Distance, sp.Node.Name())
		return ctx.Err() == nil
	})
	return 0
}
