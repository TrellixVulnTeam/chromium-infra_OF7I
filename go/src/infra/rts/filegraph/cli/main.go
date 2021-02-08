// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/maruel/subcommands"

	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/data/text"
	"go.chromium.org/luci/common/flag/fixflagpos"
	"go.chromium.org/luci/common/logging/gologger"
)

var logCfg = gologger.LoggerConfig{
	Format: `%{message}`,
	Out:    os.Stderr,
}

// Main runs the filegraph program.
func Main() {
	app := &cli.Application{
		Name: "filegraph",
		Title: text.Doc(`
			Derive and analyze a directed file graph where an edge weight (distance)
			represents how much changes in one file affects another.
		`),
		Context: func(ctx context.Context) context.Context {
			return logCfg.Use(ctx)
		},
		Commands: []*subcommands.Command{
			cmdQuery,
			cmdPath,

			{},
			subcommands.CmdHelp,
		},
	}

	os.Exit(subcommands.Run(app, fixflagpos.FixSubcommands(os.Args[1:])))
}

type baseCommandRun struct {
	subcommands.CommandRunBase
}

func (r *baseCommandRun) done(err error) int {
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	return 0
}
