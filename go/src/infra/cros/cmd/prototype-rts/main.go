// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Application prototype-rts is a local demo Remote Test Server.
package main

import (
	"context"
	"os"

	"infra/cros/cmd/prototype-rts/internal/cmd"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/logging/gologger"
)

func getApplication() *cli.Application {
	return &cli.Application{
		Name:  "prototype-rts",
		Title: "A prototype Remote Test Server.",
		Context: func(ctx context.Context) context.Context {
			return gologger.StdConfig.Use(ctx)
		},
		Commands: []*subcommands.Command{
			subcommands.CmdHelp,
			cmd.StartServices(),
			cmd.InvokeRTD(),
		},
	}
}

func main() {
	os.Exit(subcommands.Run(getApplication(), nil))
}
