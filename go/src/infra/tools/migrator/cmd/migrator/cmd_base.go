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

package main

import (
	"context"
	"os"

	"github.com/maruel/subcommands"

	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/hardcoded/chromeinfra"

	"infra/tools/migrator/internal/plugsupport"
)

// TODO(iannucci): the 'subcommands' library is a mess, use something better.

type command interface {
	subcommands.CommandRun

	positionalRange() (min, max int)

	validateFlags(ctx context.Context, positionals []string, env subcommands.Env) error
	execute(ctx context.Context) (err error)
}

type cmdBaseOptions struct {
	authOpts auth.Options
}

type cmdInitParams struct {
	opts               cmdBaseOptions
	discoverProjectDir bool
}

type cmdBase struct {
	subcommands.CommandRunBase

	discoverProjectDir bool // true to initialie projectDir based on cwd

	logFlags          logging.Config
	authFlags         authcli.Flags
	configServiceHost string

	contextConfig plugsupport.ContextConfig
	projectDir    plugsupport.ProjectDir
}

func (c *cmdBase) initFlags(p cmdInitParams) {
	c.discoverProjectDir = p.discoverProjectDir
	c.logFlags.Level = logging.Info
	c.logFlags.AddFlags(&c.Flags)
	c.authFlags.Register(&c.Flags, p.opts.authOpts)
	c.Flags.StringVar(&c.configServiceHost, "config-service-host", chromeinfra.ConfigServiceHost,
		"Hostname of a LUCI Config service to fetch project info from.")
}

func (c *cmdBase) doContextExecute(a subcommands.Application, cmd command, args []string, env subcommands.Env) int {
	// This is basically a plugsupport.RootContext(...).
	ctx := cli.GetContext(a, cmd, env)

	// Collect tweaks for the root context (the same ones we pass to the plugin).
	authOpts, err := c.authFlags.Options()
	if err != nil {
		logging.Errorf(ctx, "bad auth arguments: %s\n\n", err)
		c.GetFlags().Usage()
		return 1
	}
	c.contextConfig = plugsupport.ContextConfig{
		Logging:           c.logFlags,
		Auth:              authOpts,
		ConfigServiceHost: c.configServiceHost,
	}

	// Actually apply them to the root context.
	if ctx, err = c.contextConfig.Apply(ctx); err != nil {
		logging.Errorf(ctx, "failed to prepare the root context: %s\n\n", err)
		c.GetFlags().Usage()
		return 1
	}

	//positional
	min, max := cmd.positionalRange()
	if len(args) < min {
		logging.Errorf(ctx, "expected at least %d positional arguments, got %d", min, len(args))
		c.GetFlags().Usage()
		return 1
	}
	if len(args) > max {
		logging.Errorf(ctx, "expected at most %d positional arguments, got %d", max, len(args))
		c.GetFlags().Usage()
		return 1
	}

	if c.discoverProjectDir {
		c.projectDir, err = findProjectDir()
		if err != nil {
			logging.Errorf(ctx, "can't find the project directory: %s\n\n", err)
			return 1
		}
	}

	if err = cmd.validateFlags(ctx, args, env); err != nil {
		logging.Errorf(ctx, "bad arguments: %s\n\n", err)
		c.GetFlags().Usage()
		return 1
	}

	if err := cmd.execute(ctx); err != nil {
		errors.Log(ctx, err)
		return 1
	}

	return 0
}

func findProjectDir() (plugsupport.ProjectDir, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", errors.Annotate(err, "getting working directory").Err()
	}
	return plugsupport.FindProjectRoot(wd)
}
