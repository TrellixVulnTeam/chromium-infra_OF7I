// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

// CrosGrep is a tool to search through logs and swarming tables and fetch
// or analyze their contents.

import (
	"os"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/data/rand/mathrand"

	"infra/cros/cmd/crosgrep/internal/cmds"
)

// NewApplication returns an application object for the crosgrep command line client.
func newApplication() *cli.Application {
	return &cli.Application{
		Name:  "crosgrep",
		Title: "The crosgrep client application",
		Commands: []*subcommands.Command{
			subcommands.Section("Queries"),
			cmds.BrokenBy,
			cmds.ListAllTasks,
			cmds.RandTask,
			cmds.StatusLog,
			// TODO(gregorynisbet): Add authentication subcommands here.
			// Specifically add the commands "login", "logout", and "whoami".
		},
	}
}

// Main is the main entrypoint to the application.
func main() {
	mathrand.SeedRandomly()
	os.Exit(subcommands.Run(newApplication(), nil))
}
