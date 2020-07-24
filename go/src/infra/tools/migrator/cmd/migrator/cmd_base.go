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
	"net/http"

	"github.com/maruel/subcommands"

	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/config/cfgclient"
)

// TODO(iannucci): the 'subcommands' library is a mess, use something better.

type command interface {
	subcommands.CommandRun

	initFlags(opts cmdBaseOptions)

	positionalRange() (min, max int)

	validateFlags(ctx context.Context, positionals []string, env subcommands.Env) error
	execute(ctx context.Context) (err error)
}

type cmdBaseOptions struct {
	authOpts auth.Options
}

type cmdBase struct {
	subcommands.CommandRunBase

	logFlags  logging.Config
	authFlags authcli.Flags

	authenticator *auth.Authenticator
}

func (c *cmdBase) initFlags(opts cmdBaseOptions) {
	c.logFlags.Level = logging.Info
	c.logFlags.AddFlags(&c.Flags)
	c.authFlags.Register(&c.Flags, opts.authOpts)
}

func (c *cmdBase) doContextExecute(a subcommands.Application, cmd command, args []string, env subcommands.Env) int {
	ctx := c.logFlags.Set(cli.GetContext(a, cmd, env))
	authOpts, err := c.authFlags.Options()
	if err != nil {
		logging.Errorf(ctx, "bad auth arguments: %s\n\n", err)
		c.GetFlags().Usage()
		return 1
	}
	c.authenticator = auth.NewAuthenticator(ctx, auth.SilentLogin, authOpts)

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

	if err = cmd.validateFlags(ctx, args, env); err != nil {
		logging.Errorf(ctx, "bad arguments: %s\n\n", err)
		c.GetFlags().Usage()
		return 1
	}

	client, err := cfgclient.New(cfgclient.Options{
		ServiceHost: "luci-config.appspot.com",
		ClientFactory: func(context.Context) (*http.Client, error) {
			return c.authenticator.Client()
		},
	})
	if err != nil {
		logging.Errorf(ctx, "cannot configure LUCI config client: %s", err)
		return 1
	}

	ctx = cfgclient.Use(ctx, client)

	if err := cmd.execute(ctx); err != nil {
		errors.Log(ctx, err)
		return 1
	}

	return 0
}
