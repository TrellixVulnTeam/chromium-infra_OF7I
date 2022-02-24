// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// This is the client (called "adminclient") for the GAEv2 version of CrOSSkylabAdmin.
// This client is intended to support testing, development work, and manual administrative tasks.
// The command line interface presented here should very closely follow the RPC interface for CSA.
package main

import (
	"context"
	"os"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/data/rand/mathrand"
	"go.chromium.org/luci/common/logging/gologger"

	"infra/appengine/crosskylabadmin/gaev2/adminclient/cmds"
	"infra/appengine/crosskylabadmin/site"
)

// NewApplication returns an application object for the adminclient tool.
func newApplication() *cli.Application {
	return &cli.Application{
		Name:  "adminclient",
		Title: "CrOSSkylabAdmin client",
		Context: func(ctx context.Context) context.Context {
			return gologger.StdConfig.Use(ctx)
		},
		Commands: []*subcommands.Command{
			subcommands.Section("Authentication"),
			authcli.SubcommandInfo(site.DefaultAuthOptions, "whoami", false),
			authcli.SubcommandLogin(site.DefaultAuthOptions, "login", false),
			authcli.SubcommandLogout(site.DefaultAuthOptions, "logout", false),
			subcommands.Section("CrosAdm RPCs"),
			cmds.GetStableVersion,
			cmds.SetStableVersion,
			cmds.DeleteStableVersion,
			subcommands.Section("UFS RPCs"),
			cmds.GetMachineLSE,
		},
	}
}

// Main is the main entrypoint for adminclient.
func main() {
	mathrand.SeedRandomly()
	os.Exit(subcommands.Run(newApplication(), nil))
}
