// Copyright 2020 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/maruel/subcommands"

	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/client/versioncli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/data/rand/mathrand"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/logging/gologger"
	"go.chromium.org/luci/common/system/signals"
	"go.chromium.org/luci/hardcoded/chromeinfra"
)

const (
	// AppName is the name of the application (as it would appear on the command
	// line).
	AppName = "migrator"

	// Version is the 'symver' of the migrator application.
	Version = "0.0.1"
)

func main() {
	mathrand.SeedRandomly()

	defaults := cmdBaseOptions{
		authOpts: chromeinfra.DefaultAuthOptions(),
	}

	var application = cli.Application{
		Name: AppName,
		Title: fmt.Sprintf(
			`LUCI config migrator (%s v%s)

Tool to assist in 'global' migrations for LUCI configuration.

This tool can be used to generate a local 'migrator project' and then iterate
on it until the migration is complete. This includes semi-automated scanning to
find all LUCI projects affected by the migration, managing checkouts, sending
fixit CLs en-masse, and exporting migration status to e.g. CSV for project
management purposes.
`, AppName, Version),

		Context: func(ctx context.Context) context.Context {
			ctx = gologger.StdConfig.Use(ctx)
			ctx = logging.SetLevel(ctx, logging.Info)
			ctx, cancel := context.WithCancel(ctx)
			signals.HandleInterrupt(cancel)
			return ctx
		},

		Commands: []*subcommands.Command{
			cmdInit(defaults),
			cmdScan(defaults),

			{}, // spacer

			subcommands.CmdHelp,
			versioncli.CmdVersion(AppName + " " + Version),

			{}, // spacer

			authcli.SubcommandLogin(defaults.authOpts, "auth-login", false),
			authcli.SubcommandLogout(defaults.authOpts, "auth-logout", false),
			authcli.SubcommandInfo(defaults.authOpts, "auth-info", false),
		},
	}

	os.Exit(subcommands.Run(&application, nil))
}
