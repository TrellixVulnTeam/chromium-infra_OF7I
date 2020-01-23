// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Command autotest_status_parser extracts individual test case results from status.log.
package main

import (
	"context"
	"os"

	"github.com/maruel/subcommands"

	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/gcloud/gs"
	"go.chromium.org/luci/common/logging/gologger"
	"go.chromium.org/luci/hardcoded/chromeinfra"

	"infra/cros/cmd/phosphorus/internal/cmd"
)

func getApplication(authOpts auth.Options) *cli.Application {
	return &cli.Application{
		Name:  "phosphorus",
		Title: "A tool for running Autotest tests and uploading their results.",
		Context: func(ctx context.Context) context.Context {
			return gologger.StdConfig.Use(ctx)
		},
		Commands: []*subcommands.Command{
			subcommands.CmdHelp,

			subcommands.Section("Authentication"),
			authcli.SubcommandInfo(authOpts, "whoami", false),
			authcli.SubcommandLogin(authOpts, "login", false),
			authcli.SubcommandLogout(authOpts, "logout", false),

			subcommands.Section("Main commands"),
			cmd.Prejob,
			cmd.RunTest,
			cmd.UploadToTKO,
			cmd.UploadToGS(authOpts),
		},
	}
}

func main() {
	auth := chromeinfra.DefaultAuthOptions()
	auth.Scopes = append(auth.Scopes, gs.ReadWriteScopes...)
	os.Exit(subcommands.Run(getApplication(auth), nil))
}
