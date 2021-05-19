// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/data/text"
)

const (
	chromeInternalHost      = "chrome-internal.googlesource.com"
	manifestInternalProject = "chromeos/manifest-internal"
)

type projectBuildspec struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
	buildspec string
	project   string
}

func cmdProjectBuildspec(authOpts auth.Options) *subcommands.Command {
	return &subcommands.Command{
		UsageLine: "project-buildspec --buildspec=85/13277.0.0.xml --project_name=chromeos/project/galaxy/milkyway",
		ShortDesc: "Create a project-specific buildspec for a specific project/program and version",
		CommandRun: func() subcommands.CommandRun {
			b := &projectBuildspec{}
			b.authFlags = authcli.Flags{}
			b.authFlags.Register(b.GetFlags(), authOpts)
			b.Flags.StringVar(&b.buildspec, "buildspec", "",
				text.Doc(`
				Path to manifest within manifest-versions repo, relative to
				https://chrome-internal.googlesource.com/chromeos/manifest-versions/+/HEAD/buildspecs/
				e.g. 85/13277.0.0.xml`))
			b.Flags.StringVar(&b.project, "project", "",
				"Name of the project/program to create the project-specific buildspec for")
			return b
		}}
}

func (b *projectBuildspec) validate() error {
	if b.buildspec == "" {
		return fmt.Errorf("--buildspec required")
	}

	if b.project == "" {
		return fmt.Errorf("--project required")
	}

	return nil
}

func (b *projectBuildspec) Run(a subcommands.Application, args []string, env subcommands.Env) int {
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

	if err := b.CreateProjectBuildspec(authedClient); err != nil {
		LogErr(err.Error())
		return 4
	}

	return 0
}

// CreateProjectBuildspec creates a project/program-specific buildspec as
// outlined in go/per-project-buildspecs.
func (b *projectBuildspec) CreateProjectBuildspec(authedClient *http.Client) error {
	return nil
}
