// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package tasks

import (
	"encoding/json"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"

	"infra/cmd/mallet/internal/site"
	"infra/cmdsupport/cmdlib"
	"infra/cros/recovery"
	"infra/cros/recovery/tasknames"
	"infra/cros/recovery/tlw"
)

// RecoveryConfig subcommand: For now, print the config file content to terminal/file.
var RecoveryConfig = &subcommands.Command{
	UsageLine: "config [-deploy] [-cros] [-labstation]",
	ShortDesc: "print the JSON plan configuration file",
	LongDesc:  "print the JSON plan configuration file.",
	CommandRun: func() subcommands.CommandRun {
		c := &printConfigRun{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.Flags.BoolVar(&c.deployTask, "deploy", false, "Used deploy task.")
		c.Flags.BoolVar(&c.cros, "cros", false, "Print for ChromeOS devices.")
		c.Flags.BoolVar(&c.labstation, "labstation", false, "Print for labstations.")
		return c
	},
}

type printConfigRun struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags

	deployTask bool
	cros       bool
	labstation bool
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
	ctx := cli.GetContext(a, c, env)
	tn := tasknames.Recovery
	if c.deployTask {
		tn = tasknames.Deploy
	}
	var dsl []tlw.DUTSetupType
	if c.cros {
		dsl = append(dsl, tlw.DUTSetupTypeCros)
	}
	if c.labstation {
		dsl = append(dsl, tlw.DUTSetupTypeLabstation)
	}
	for _, ds := range dsl {
		if c, err := recovery.ParsedDefaultConfiguration(ctx, tn, ds); err != nil {
			return errors.Annotate(err, "inner run").Err()
		} else if s, err := json.MarshalIndent(c, "", "\t"); err != nil {
			return errors.Annotate(err, "inner run").Err()
		} else {
			a.GetOut().Write(s)
		}
	}
	return nil
}
