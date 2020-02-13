// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"io/ioutil"
	"log"
	"os"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/data/rand/mathrand"
	"go.chromium.org/luci/hardcoded/chromeinfra"

	"infra/cmd/labtool/audit"
)

func getApplication() *cli.Application {
	return &cli.Application{
		Name:  "labtool",
		Title: "Asset tracking and management",
		Context: func(ctx context.Context) context.Context {
			return ctx
		},
		Commands: []*subcommands.Command{
			subcommands.CmdHelp,
			subcommands.Section("Authentication"),
			authcli.SubcommandInfo(chromeinfra.DefaultAuthOptions(), "whoami", false),
			authcli.SubcommandLogin(chromeinfra.DefaultAuthOptions(), "login", false),
			authcli.SubcommandLogout(chromeinfra.DefaultAuthOptions(), "logout", false),
			subcommands.Section("Asset Management"),
			audit.AuditCmd,
		},
	}
}

func main() {
	mathrand.SeedRandomly()
	log.SetOutput(ioutil.Discard)
	os.Exit(subcommands.Run(getApplication(), nil))
}
