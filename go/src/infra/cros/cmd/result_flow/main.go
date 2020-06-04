// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Command result_flow implements the result flow recipe's steps.
package main

import (
	"context"
	"os"

	"github.com/maruel/subcommands"

	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/data/rand/mathrand"
	"go.chromium.org/luci/common/logging/gologger"

	"infra/cros/cmd/result_flow/internal/site"
)

func getApplication() *cli.Application {
	return &cli.Application{
		Name:  "result_flow",
		Title: "Binary steps implementations for result_flow recipe.",
		Context: func(ctx context.Context) context.Context {
			return gologger.StdConfig.Use(ctx)
		},
		Commands: []*subcommands.Command{
			subcommands.CmdHelp,

			subcommands.Section("Authentication"),
			authcli.SubcommandLogin(site.DefaultAuthOptions, "login", false),
			authcli.SubcommandLogout(site.DefaultAuthOptions, "logout", false),
			authcli.SubcommandInfo(site.DefaultAuthOptions, "whoami", false),

			subcommands.Section("Steps"),
			// TODO(lxn): The subcommands will be added in following CLs.
		},
	}
}

func main() {
	mathrand.SeedRandomly()
	os.Exit(subcommands.Run(getApplication(), nil))
}
