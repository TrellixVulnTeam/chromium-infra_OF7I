// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cli

import (
	"context"
	"os"

	"cloud.google.com/go/storage"
	"github.com/maruel/subcommands"

	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/flag/fixflagpos"
	"go.chromium.org/luci/common/logging/gologger"
)

// Params is the parameters for the dirmd application.
type Params struct {
	Auth auth.Options
}

var logCfg = gologger.LoggerConfig{
	Format: `%{message}`,
	Out:    os.Stderr,
}

// application creates the application and configures its subcommands.
// Ignores p.Auth.Scopes.
func application(p Params) *cli.Application {
	p.Auth.Scopes = []string{storage.ScopeReadWrite}

	return &cli.Application{
		Name:  "dirmd",
		Title: "A tool to work with DIR_METADATA files",
		Context: func(ctx context.Context) context.Context {
			return logCfg.Use(ctx)
		},
		Commands: []*subcommands.Command{
			cmdValidate(),
			cmdExport(),
			cmdCompute(),
			cmdChromiumUpdate(&p),
			cmdMigrate(),

			{},
			authcli.SubcommandLogin(p.Auth, "auth-login", false),
			authcli.SubcommandLogout(p.Auth, "auth-logout", false),
			authcli.SubcommandInfo(p.Auth, "auth-info", false),

			{},
			subcommands.CmdHelp,
		},
	}
}

// Main is the main function of the dirmd application.
func Main(p Params, args []string) int {
	return subcommands.Run(application(p), fixflagpos.FixSubcommands(args))
}
