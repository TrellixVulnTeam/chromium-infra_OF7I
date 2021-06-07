// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

// CrosGrep is a tool to search through logs and swarming tables and fetch
// or analyze their contents.

import (
	"context"
	"os"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/data/rand/mathrand"
	"go.chromium.org/luci/common/logging/gologger"

	"infra/cros/cmd/crosgrep/internal/cmds"
	"infra/cros/karte/site"
)

// NewApplication returns an application object for the crosgrep command line client.
func newApplication() *cli.Application {
	return &cli.Application{
		Name:  "crosgrep",
		Title: "The crosgrep client application",
		Context: func(ctx context.Context) context.Context {
			return gologger.StdConfig.Use(ctx)
		},
		Commands: []*subcommands.Command{
			cmds.ListAllTasks,
			cmds.BrokenBy,
			authcli.SubcommandLogin(site.DefaultAuthOptions, "login", false),
			authcli.SubcommandLogout(site.DefaultAuthOptions, "logout", false),
			authcli.SubcommandInfo(site.DefaultAuthOptions, "whoami", false),
		},
	}
}

// Main is the main entrypoint to the application.
func main() {
	mathrand.SeedRandomly()
	os.Exit(subcommands.Run(newApplication(), nil))
}
