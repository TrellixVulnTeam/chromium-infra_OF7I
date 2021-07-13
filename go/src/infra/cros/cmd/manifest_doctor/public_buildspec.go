// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package main

import (
	"context"
	"net/http"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/auth/client/authcli"

	"infra/cros/internal/gs"
)

type publicBuildspec struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
	push      bool
}

func cmdPublicBuildspec(authOpts auth.Options) *subcommands.Command {
	return &subcommands.Command{
		UsageLine: "public-buildspec",
		ShortDesc: "Create public buildspecs for all private buildspecs " +
			"created since the previous run of this tool.",
		CommandRun: func() subcommands.CommandRun {
			b := &publicBuildspec{}
			b.authFlags = authcli.Flags{}
			b.authFlags.Register(b.GetFlags(), authOpts)
			b.Flags.BoolVar(&b.push, "push", false,
				"Push changes to remote as well (i.e. not a dryrun).")
			return b
		}}
}

func (b *publicBuildspec) validate() error {
	return nil
}

func (b *publicBuildspec) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	// Common setup (argument validation, logging, etc.)
	ret := SetUp(b, a, args, env)
	if ret != 0 {
		return ret
	}

	ctx := context.Background()
	authOpts, err := b.authFlags.Options()
	if err != nil {
		LogErr(err.Error())
		return 2
	}
	authedClient, err := auth.NewAuthenticator(ctx, auth.SilentLogin, authOpts).Client()
	if err != nil {
		LogErr(err.Error())
		return 3
	}

	gsClient, err := gs.NewProdClient(ctx, authedClient)
	if err != nil {
		LogErr(err.Error())
		return 4
	}

	if err := b.CreatePublicBuildspec(authedClient, gsClient); err != nil {
		LogErr(err.Error())
		return 5
	}

	return 0
}

// CreateProjectBuildspec creates a public buildspec as outlined in go/per-project-buildspecs.
func (b *publicBuildspec) CreatePublicBuildspec(authedClient *http.Client, gsClient gs.Client) error {
	return nil
}
