// Copyright 2020 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package main

import (
	"context"
	"os"

	"github.com/maruel/subcommands"

	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/data/rand/mathrand"
	"go.chromium.org/luci/common/flag/fixflagpos"
	"go.chromium.org/luci/common/logging/gologger"
)

func main() {
	mathrand.SeedRandomly()

	application := cli.Application{
		Name:  "result_adapter",
		Title: "A CLI tool to convert test results to ResultSink native format then upload them to ResultDB via ResultSink.",
		Context: func(ctx context.Context) context.Context {
			logCfg := gologger.LoggerConfig{
				Format: `%{message}`,
				Out:    os.Stderr,
			}
			return logCfg.Use(ctx)
		},
		Commands: []*subcommands.Command{
			cmdGtest(),
			cmdJSON(),
			cmdSingle(),
			cmdGo(),

			{}, // a separator
			subcommands.CmdHelp,
		},
	}
	os.Exit(subcommands.Run(&application, fixflagpos.FixSubcommands(os.Args[1:])))
}
