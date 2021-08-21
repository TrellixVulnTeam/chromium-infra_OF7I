// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package tasks

import (
	"io"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/errors"

	"infra/cmd/mallet/internal/site"
	"infra/cmdsupport/cmdlib"
	"infra/cros/recovery"
)

// RecoveryConfig subcommand: For now, print the config file content to terminal/file.
var RecoveryConfig = &subcommands.Command{
	UsageLine: "recovery-config",
	ShortDesc: "print the JSON plan configuration file",
	LongDesc:  "print the JSON plan configuration file.",
	CommandRun: func() subcommands.CommandRun {
		c := &printConfigRun{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		return c
	},
}

type printConfigRun struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
}

// Run output the content of the recovery config file.
func (c *printConfigRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

// innerRun executes internal logic of output file content.
func (c *printConfigRun) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	r := recovery.DefaultConfig()
	b, err := io.ReadAll(r)
	if err != nil {
		return errors.Annotate(err, "inner run").Err()
	}
	a.GetOut().Write(b)
	return nil
}
