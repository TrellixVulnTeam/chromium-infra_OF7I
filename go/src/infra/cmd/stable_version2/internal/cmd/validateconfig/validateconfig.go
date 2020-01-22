// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package validateconfig

import (
	"encoding/json"
	"errors"
	"fmt"

	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"

	"github.com/maruel/subcommands"
	"infra/cmd/stable_version2/internal/cmd"
	"infra/cmd/stable_version2/internal/cmd/validateconfig/internal/querygs"
	"infra/cmd/stable_version2/internal/site"
	"infra/cmd/stable_version2/internal/utils"
	vc "infra/libs/cros/stableversion/validateconfig"
)

// Cmd is the top-level runnable for the validate-config subcommand of stable_version2
var Cmd = &subcommands.Command{
	UsageLine: `validate-config /path/to/stable_versions.cfg`,
	ShortDesc: "check that a stable_versions.cfg file is well-formed",
	LongDesc: `check that a stable_versions.cfg file is well-formed.

This command exists solely to validate a stable_versions.cfg file.
Its intended consumer is a submit hook that runs in the infra/config repo.
`,
	CommandRun: func() subcommands.CommandRun {
		c := &command{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)

		return c
	},
}

type command struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
}

func (c *command) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmd.PrintError(a.GetErr(), err)
		return 1
	}
	return 0
}

func (c *command) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	ctx := cli.GetContext(a, c, env)
	ctx = cmd.SetupLogging(ctx)

	if len(args) == 0 {
		return errors.New("need at least one file")
	}
	if len(args) > 1 {
		return errors.New("validating multiple files not yet supported")
	}
	sv, err := vc.InspectFile(args[0])
	if err != nil {
		return fmt.Errorf("inspecting file: %s", err)
	}

	t, err := cmd.NewAuthenticatedTransport(ctx, &c.authFlags)
	if err != nil {
		return fmt.Errorf("creating authenticated transport: %s", err)
	}
	var r querygs.Reader
	if err := r.Init(ctx, t, utils.Unmarshaller, "validate-config"); err != nil {
		return fmt.Errorf("initializing Google Storage client: %s", err)
	}

	res, err := r.ValidateConfig(sv)
	if err != nil {
		return fmt.Errorf("valdating config using Google Storage: %s", err)
	}
	res.RemoveWhitelistedDUTs()
	msg, err := json.MarshalIndent(res, "", "    ")
	if err != nil {
		panic("failed to marshal JSON")
	}
	if count := res.AnomalyCount(); count > 0 {
		fmt.Printf("%s\n", msg)
		return fmt.Errorf("(%d) errors detected", count)
	}

	fmt.Printf("%s\n", vc.FileSeemsLegit)
	return nil
}
