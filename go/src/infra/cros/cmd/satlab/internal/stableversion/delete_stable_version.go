// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package stableversion

import (
	"fmt"
	"infra/cros/cmd/satlab/internal/site"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
)

// DeleteStableVersionCmd deletes a stable version entry.
var DeleteStableVersionCmd = &subcommands.Command{
	UsageLine: `delete-stable-version`,
	ShortDesc: `Delete the stable version using {board, model} or {hostname}. This only deletes satlab entries.`,
	CommandRun: func() subcommands.CommandRun {
		r := &deleteStableVersionRun{}

		r.authFlags.Register(&r.Flags, site.DefaultAuthOptions)
		r.envFlags.Register(&r.Flags)
		r.commonFlags.Register(&r.Flags)

		r.Flags.StringVar(&r.board, "board", "", `the board or build target (used with model)`)
		r.Flags.StringVar(&r.model, "model", "", `the model (used with board)`)
		r.Flags.StringVar(&r.hostname, "hostname", "", `the hostname (used by itself)`)

		return r
	},
}

// DeleteStableVersionRun is the command for adminclient set-stable-version.
type deleteStableVersionRun struct {
	subcommands.CommandRunBase

	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags

	board    string
	model    string
	hostname string
}

// Run runs the command, prints the error if there is one, and returns an exit status.
func (c *deleteStableVersionRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	fmt.Fprintf(a.GetErr(), "%s\n", "not yet implemented")
	return 2
}
