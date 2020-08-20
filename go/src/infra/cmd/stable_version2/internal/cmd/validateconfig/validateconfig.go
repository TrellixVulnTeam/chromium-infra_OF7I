// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package validateconfig

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"

	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"

	"infra/cmd/stable_version2/internal/cmd"
	"infra/cmd/stable_version2/internal/cmd/validateconfig/querygs"
	"infra/cmd/stable_version2/internal/site"
	"infra/cmd/stable_version2/internal/utils"
	gitlib "infra/libs/cros/git"
	vc "infra/libs/cros/stableversion/validateconfig"

	"github.com/maruel/subcommands"
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

		c.Flags.BoolVar(&c.alwaysExitZero, "always-exit-zero", false, "exit successfully regardless of what errors occur.")
		c.Flags.BoolVar(&c.remoteFile, "remote-file", false, "get file from Gitiles instead of locally.")
		return c
	},
}

type command struct {
	subcommands.CommandRunBase
	authFlags      authcli.Flags
	alwaysExitZero bool
	remoteFile     bool
}

func (c *command) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	failure := 1
	if c.alwaysExitZero {
		failure = 0
	}
	if err := c.innerRun(a, args, env); err != nil {
		cmd.PrintError(a.GetErr(), err)
		return failure
	}
	return 0
}

func (c *command) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	ctx := cli.GetContext(a, c, env)
	ctx = cmd.SetupLogging(ctx)

	var contents []byte
	if c.remoteFile {
		var err error
		if len(args) != 0 {
			return errors.New("cannot provide explicit file when using remote file")
		}
		contents, err = fetchGitPath(ctx, cmd.StableVersionConfigPath, &c.authFlags)
		if err != nil {
			return fmt.Errorf("getting remote file contents: %s", err)
		}
	} else {
		var err error
		if len(args) == 0 {
			return errors.New("need at least one file")
		}
		if len(args) > 1 {
			return errors.New("validating multiple files not yet supported")
		}
		contents, err = ioutil.ReadFile(args[0])
		if err != nil {
			return fmt.Errorf("reading local file: %s", err)
		}
	}

	sv, err := vc.InspectBuffer(contents)
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

	res, err := r.ValidateConfig(ctx, sv)
	if err != nil {
		return fmt.Errorf("valdating config using Google Storage: %s", err)
	}
	res.RemoveAllowedDUTs()
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

func fetchGitPath(ctx context.Context, path string, f *authcli.Flags) ([]byte, error) {
	hc, err := cmd.NewHTTPClient(ctx, f)
	if err != nil {
		return nil, err
	}
	gc, err := gitlib.NewClient(ctx, hc, cmd.GerritHost, cmd.GitilesHost, cmd.Project, cmd.Branch)
	if err != nil {
		return nil, fmt.Errorf("creating client: %s", err.Error())
	}
	res, err := gc.GetFile(ctx, cmd.StableVersionConfigPath)
	if err != nil {
		return nil, fmt.Errorf("getting file: %s", err.Error())
	}
	return []byte(res), nil
}
