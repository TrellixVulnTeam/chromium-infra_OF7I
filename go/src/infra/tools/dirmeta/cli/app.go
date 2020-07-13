// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cli

import (
	"context"
	"os"

	"github.com/maruel/subcommands"

	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/flag/fixflagpos"
	"go.chromium.org/luci/common/logging/gologger"
)

var logCfg = gologger.LoggerConfig{
	Format: `%{message}`,
	Out:    os.Stderr,
}

// application creates the application and configures its subcommands.
func application() *cli.Application {
	return &cli.Application{
		Name:  "dirmeta",
		Title: "A tool to work with DIR_METADATA files",
		Context: func(ctx context.Context) context.Context {
			return logCfg.Use(ctx)
		},
		Commands: []*subcommands.Command{
			cmdValidate(),
			cmdExport(),

			{},
			subcommands.CmdHelp,
		},
	}
}

// Main is the main function of the dirmeta application.
func Main(args []string) int {
	return subcommands.Run(application(), fixflagpos.FixSubcommands(args))
}
