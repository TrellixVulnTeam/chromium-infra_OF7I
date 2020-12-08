// Copyright 2020 The LUCI Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// This provides a command `provenance` to generate provenance metadata
// for luci built artifacts. Provenance format here is based upon BinAuthz
// Attestation format.

package main

import (
	"context"
	"os"

	"github.com/maruel/subcommands"

	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/client/versioncli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/data/rand/mathrand"
	"go.chromium.org/luci/common/logging/gologger"
	"go.chromium.org/luci/hardcoded/chromeinfra"
)

// Please update version for each major change {behavior, arguments,
// supported commands}. eg. major.minor.patch
const version = "0.1.0"

func getApplication(defaultAuthOpts auth.Options) *cli.Application {
	defaultAuthOpts.Scopes = []string{auth.OAuthScopeEmail, "https://www.googleapis.com/auth/cloud-platform"}
	return &cli.Application{
		Name:  "provenance",
		Title: "Client for interfacing with BCID for Software Verification Service",
		Context: func(ctx context.Context) context.Context {
			return gologger.StdConfig.Use(ctx)
		},
		// Subcommands available under provenance binary.
		Commands: []*subcommands.Command{
			subcommands.CmdHelp,
			cmdGenerate(defaultAuthOpts),
			authcli.SubcommandInfo(defaultAuthOpts, "whoami", false),
			authcli.SubcommandLogin(defaultAuthOpts, "login", false),
			authcli.SubcommandLogout(defaultAuthOpts, "logout", false),
			versioncli.CmdVersion(version),
		},
	}
}

func main() {
	mathrand.SeedRandomly()
	app := getApplication(chromeinfra.DefaultAuthOptions())
	os.Exit(subcommands.Run(app, nil))
}
