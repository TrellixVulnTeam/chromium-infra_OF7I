// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cli

import (
	"fmt"

	"github.com/maruel/subcommands"

	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/data/text"
	"go.chromium.org/luci/common/errors"
)

var cmdPath = &subcommands.Command{
	UsageLine: `path [flags] SOURCE_FILE TARGET_FILE`,
	ShortDesc: "print the shortest path from SOURCE_FILE to TARGET_FILE",
	LongDesc: text.Doc(`
		Print the shortest path from SOURCE_FILE to TARGET_FILE

		Each output line has format "<distance> (+<delta>) <filename>",
		where the filename is forward-slash-separated and has "//" prefix.
		Output example:
			0.00 (+0.00) //source_file.cc
			1.00 (+1.00) //intermediate_file.cc
			3.00 (+2.00) //target_file.cc

		Both files must be in the same git repository.
	`),
	CommandRun: func() subcommands.CommandRun {
		r := &pathRun{}
		r.gitGraph.RegisterFlags(&r.Flags)
		return r
	},
}

type pathRun struct {
	baseCommandRun
	gitGraph
}

func (r *pathRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	ctx := cli.GetContext(a, r, env)

	if len(args) != 2 {
		return r.done(errors.Reason("usage: filegraph path SOURCE_FILE TARGET_FILE").Err())
	}

	nodes, err := r.loadSyncedNodes(ctx, args[0], args[1])
	if err != nil {
		return r.done(err)
	}

	shorest := r.query(nodes[0]).ShortestPath(nodes[1])
	if shorest == nil {
		return r.done(errors.New("not reachable"))
	}

	prevDist := 0.0
	for _, sp := range shorest.Path() {
		fmt.Printf("%.2f (+%.2f) %s\n", sp.Distance, sp.Distance-prevDist, sp.Node.Name())
		prevDist = sp.Distance
	}
	return 0
}
