// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Command result_flow implements the result flow recipe's steps.
package main

import (
	"context"
	"os"

	"github.com/maruel/subcommands"

	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/data/rand/mathrand"
	"go.chromium.org/luci/common/logging/gologger"

	"infra/cros/cmd/build-parallels-image/internal/cmd"
)

func getApplication() *cli.Application {
	return &cli.Application{
		Name:  "build-parallels-image",
		Title: "Binary steps implementations for build_parallels_image recipe.",
		Context: func(ctx context.Context) context.Context {
			return gologger.StdConfig.Use(ctx)
		},
		Commands: []*subcommands.Command{
			subcommands.CmdHelp,

			subcommands.Section("Steps"),
			cmd.Provision,
		},
	}
}

func main() {
	mathrand.SeedRandomly()
	os.Exit(subcommands.Run(getApplication(), nil))
}
