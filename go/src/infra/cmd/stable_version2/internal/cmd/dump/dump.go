// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dump

import (
	"context"
	"fmt"

	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"

	"infra/cmd/stable_version2/internal/cmd"
	"infra/cmd/stable_version2/internal/site"
	"infra/libs/cros/git"
	svlib "infra/libs/cros/stableversion"
	filter "infra/libs/cros/stableversion/filter"
	vc "infra/libs/cros/stableversion/validateconfig"

	"github.com/maruel/subcommands"
	labPlatform "go.chromium.org/chromiumos/infra/proto/go/lab_platform"
)

// Cmd is the top-level runnable for the dump subcommand of stable_version2
var Cmd = &subcommands.Command{
	UsageLine: `dump -model [MODEL]`,
	ShortDesc: "show remote stable_versions.cfg file",
	LongDesc:  `Show remote stable_versions.cfg file.`,
	CommandRun: func() subcommands.CommandRun {
		c := &command{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.Flags.StringVar(&c.model, "model", "", "restrict output to stable version records associated with `model`")
		return c
	},
}

type command struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags

	model string
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

	sv, err := filterByModel(contents, c.model)
	if err != nil {
		return fmt.Errorf("filtering out model (%s): %s", c.model, err)
	}

	jsonStr, err := svlib.WriteSVToString(sv)
	if err != nil {
		return fmt.Errorf("converting to JSON: %s", err)
	}

	fmt.Fprintf(a.GetOut(), "%s\n", jsonStr)
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

// filterByModel produces a parsed stableversion file with the entries restricted to those
// from the specified model. If the model is "", then all entries are returned.
func filterByModel(contents []byte, model string) (*labPlatform.StableVersions, error) {
	parsed, err := vc.ParseStableVersions(contents)
	if err != nil {
		return nil, err
	}
	return filter.WithModel(parsed, model), nil
}
