// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Satlab is a wrapper around shivas.

package main

import (
	"context"
	"os"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/data/rand/mathrand"

	"infra/cros/cmd/satlab/internal/meta"
	"infra/cros/cmd/satlab/internal/site"
	"infra/cros/cmd/satlab/internal/subcmds"
)

// GetApplication returns the main application.
func getApplication() *cli.Application {
	return &cli.Application{
		Name:  "satlab",
		Title: `Satlab DUT Management Tool`,
		Context: func(ctx context.Context) context.Context {
			return ctx
		},
		Commands: []*subcommands.Command{
			subcommands.CmdHelp,
			subcommands.Section("Meta"),
			meta.Version,
			meta.Update,
			subcommands.Section("Authentication"),
			authcli.SubcommandInfo(site.DefaultAuthOptions, "whoami", false),
			authcli.SubcommandLogin(site.DefaultAuthOptions, "login", false),
			authcli.SubcommandLogout(site.DefaultAuthOptions, "logout", false),
			subcommands.Section("Resource Management"),
			subcmds.AddCmd,
			subcmds.DeleteCmd,
			subcmds.GetCmd,
		},
	}
}

// Main is the entrypoint for "satlab".
func main() {
	mathrand.SeedRandomly()
	os.Exit(subcommands.Run(getApplication(), nil))
}
