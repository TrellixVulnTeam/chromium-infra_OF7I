// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Binary gaedeploy is a helper to deploy GAE apps from tarballs.
package main

import (
	"context"
	"os"

	"github.com/maruel/subcommands"

	"go.chromium.org/luci/client/versioncli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/data/rand/mathrand"
	"go.chromium.org/luci/common/flag/fixflagpos"
	"go.chromium.org/luci/common/logging/gologger"
)

const (
	// Version is the version of gaedeploy tool.
	Version = "1.0.0"
	// UserAgent is used in HTTP headers of requests from cloudbuildhelper.
	UserAgent = "gaedeploy v" + Version
)

func getApplication() *cli.Application {
	return &cli.Application{
		Name:  "gaedeploy",
		Title: "Helper for building GAE application (" + UserAgent + ")",

		Context: func(ctx context.Context) context.Context {
			return gologger.StdConfig.Use(ctx)
		},

		Commands: []*subcommands.Command{
			subcommands.CmdHelp,
			versioncli.CmdVersion(UserAgent),
			cmdModule,
			cmdYaml,

			// Advanced.
			cmdCacheTrim,
		},
	}
}

func main() {
	mathrand.SeedRandomly()
	os.Exit(subcommands.Run(getApplication(), fixflagpos.FixSubcommands(os.Args[1:])))
}
