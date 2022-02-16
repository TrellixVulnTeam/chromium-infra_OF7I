// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package stableversion

import (
	"context"
	"fmt"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"

	"infra/cros/cmd/satlab/internal/site"
)

var SetStableVersionCmd = &subcommands.Command{
	UsageLine: `set-stable-version`,
	ShortDesc: `Set the stable version using {board, model} or {hostname}.`,
	CommandRun: func() subcommands.CommandRun {
		r := &setStableVersionRun{}

		r.authFlags.Register(&r.Flags, site.DefaultAuthOptions)
		r.envFlags.Register(&r.Flags)
		r.commonFlags.Register(&r.Flags)

		r.Flags.StringVar(&r.board, "board", "", `the board or build target (used with model)`)
		r.Flags.StringVar(&r.model, "model", "", `the model (used with board)`)
		r.Flags.StringVar(&r.hostname, "hostname", "", `the hostname (used by itself)`)
		r.Flags.StringVar(&r.os, "os", "", `the OS version to set (no change if empty)`)
		r.Flags.StringVar(&r.fw, "fw", "", `the firmware version to set (no change if empty)`)
		r.Flags.StringVar(&r.fwImage, "fwImage", "", `the firmware image version to set (no change if empty)`)

		return r
	},
}

// SetStableVersionRun is the command for adminclient set-stable-version.
type setStableVersionRun struct {
	subcommands.CommandRunBase

	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags

	board    string
	model    string
	hostname string
	os       string
	fw       string
	fwImage  string
}

// Run runs the command, prints the error if there is one, and returns an exit status.
func (c *setStableVersionRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	ctx := cli.GetContext(a, c, env)
	if err := c.validateArguments(ctx, a, args, env); err != nil {
		fmt.Fprintf(a.GetErr(), "%s: %s\n", a.GetName(), err)
		return 1
	}
	if err := c.innerRun(ctx, a, args, env); err != nil {
		fmt.Fprintf(a.GetErr(), "%s: %s\n", a.GetName(), err)
		return 1
	}
	return 0
}

// ValidateArguments validates the arguments given to the satlab command.
func (c *setStableVersionRun) validateArguments(ctx context.Context, a subcommands.Application, args []string, env subcommands.Env) error {
	useHostnameStrategy := c.hostname != ""
	useBoardModelStrategy := (c.board != "") && (c.model != "")
	if useHostnameStrategy {
		if useBoardModelStrategy {
			return errors.Reason("board and model should not be set if hostname is provided").Err()
		}
		return nil
	} // Hostname strategy not set.
	if !useBoardModelStrategy {
		return errors.Reason("must use at least one of {board, model} or {hostname} strategy").Err()
	}
	return nil
}

// InnerRun creates a client, sends a GetStableVersion request, and prints the response.
func (c *setStableVersionRun) innerRun(ctx context.Context, a subcommands.Application, args []string, env subcommands.Env) error {
	return errors.Reason("not yet implemented").Err()
}
