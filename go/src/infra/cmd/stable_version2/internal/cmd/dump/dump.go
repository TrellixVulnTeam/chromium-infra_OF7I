// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dump

import (
	"context"
	"fmt"

	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"

	"github.com/maruel/subcommands"
	"infra/cmd/stable_version2/internal/cmd"
	"infra/cmd/stable_version2/internal/site"
	"infra/libs/cros/stableversion/git"
)

// Cmd is the top-level runnable for the dump subcommand of stable_version2
var Cmd = &subcommands.Command{
	UsageLine: `dump`,
	ShortDesc: "show remote stable_versions.cfg file",
	LongDesc:  `Show remote stable_versions.cfg file.`,
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

	var contents []byte
	contents, err := fetchStableVersionFile(ctx, &c.authFlags)

	if err != nil {
		return fmt.Errorf("getting remote file contents: %s", err)
	}

	fmt.Fprintf(a.GetOut(), "%s\n", string(contents))
	return nil
}

func fetchStableVersionFile(ctx context.Context, f *authcli.Flags) ([]byte, error) {
	hc, err := cmd.NewHTTPClient(ctx, f)
	if err != nil {
		return nil, err
	}
	gc, err := git.NewClient(ctx, hc, cmd.GerritHost, cmd.GitilesHost, cmd.Project, cmd.Branch)
	if err != nil {
		return nil, fmt.Errorf("creating client: %s", err.Error())
	}
	res, err := gc.GetFile(ctx, cmd.StableVersionConfigPath)
	if err != nil {
		return nil, fmt.Errorf("getting file: %s", err.Error())
	}
	return []byte(res), nil
}
