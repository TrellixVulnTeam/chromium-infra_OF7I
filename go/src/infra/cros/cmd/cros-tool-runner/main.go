// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// cros-tool-runner is handling requests and run corresponding tools at various stages of build/test workflow.
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
	serverauth "go.chromium.org/luci/server/auth"

	"infra/cros/cmd/cros-tool-runner/internal/tasks"
)

func main() {
	auth := chromeinfra.DefaultAuthOptions()
	auth.Scopes = append(auth.Scopes, serverauth.CloudOAuthScopes...)
	auth.Scopes = append(auth.Scopes, gs.ReadWriteScopes...)
	os.Exit(subcommands.Run(getApplication(auth), nil))
}

func getApplication(authOpts auth.Options) *cli.Application {
	return &cli.Application{
		Name:  "cros-tool-runner",
		Title: "A tool for running corresponding tools.",
		Context: func(ctx context.Context) context.Context {
			return gologger.StdConfig.Use(ctx)
		},
		Commands: []*subcommands.Command{
			subcommands.CmdHelp,
			subcommands.Section("Authentication"),
			authcli.SubcommandInfo(authOpts, "whoami", false),
			authcli.SubcommandLogin(authOpts, "login", false),
			authcli.SubcommandLogout(authOpts, "logout", false),
			tasks.Provision(authOpts),
			tasks.Test(authOpts),
		},
	}
}
