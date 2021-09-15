// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package main

import (
	"context"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/errors"
	luciflag "go.chromium.org/luci/common/flag"
	lgs "go.chromium.org/luci/common/gcloud/gs"

	"infra/cros/internal/gerrit"
	"infra/cros/internal/gs"
	"infra/cros/internal/repo"
)

const (
	internalBuildspecsGSBucket = "buildspecs-internal"
	externalBuildspecsGSBucket = "buildspecs-external"
)

type publicBuildspec struct {
	subcommands.CommandRunBase
	authFlags  authcli.Flags
	push       bool
	watchPaths []string
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
			b.Flags.Var(luciflag.CommaList(&b.watchPaths), "paths",
				"Comma-separated list of subdirs to watch within the internal "+
					"buildspecs bucket.")
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

	gerritClient, err := gerrit.NewClient(authedClient)
	if err != nil {
		LogErr(err.Error())
		return 5
	}

	if err := b.CreatePublicBuildspecs(ctx, gsClient, gerritClient); err != nil {
		LogErr(err.Error())
		return 6
	}

	return 0
}

// CreateProjectBuildspec creates a public buildspec as
// outlined in go/per-project-buildspecs.
func (b *publicBuildspec) CreatePublicBuildspecs(ctx context.Context, gsClient gs.Client, gerritClient *gerrit.Client) error {
	errs := []error{}
	for _, watchPath := range b.watchPaths {
		LogOut("Looking at gs://%s/%s...\n", internalBuildspecsGSBucket, watchPath)
		internalBuildspecs, err := gsClient.List(ctx, internalBuildspecsGSBucket, watchPath)
		if err != nil {
			LogErr(errors.Annotate(err, "failed to list internal buildspecs for dir %s, skipping...", watchPath).Err().Error())
			errs = append(errs, err)
			continue
		}

		externalBuildspecs, err := gsClient.List(ctx, externalBuildspecsGSBucket, watchPath)
		if err != nil {
			LogErr(errors.Annotate(err, "failed to list external buildspecs for dir %s, skipping...", watchPath).Err().Error())
			errs = append(errs, err)
		}
		externalBuildspecMap := make(map[string]bool, len(externalBuildspecs))
		for _, externalBuildspec := range externalBuildspecs {
			externalBuildspecMap[externalBuildspec] = true
		}

		for _, internalBuildspec := range internalBuildspecs {
			// Skip buildspecs that are already present.
			if _, ok := externalBuildspecMap[internalBuildspec]; ok {
				continue
			}

			// Load internal buildspec.
			internalPath := lgs.MakePath(internalBuildspecsGSBucket, internalBuildspec)
			buildspecData, err := gsClient.Read(internalPath)
			if err != nil {
				LogErr(errors.Annotate(err, "failed to read internal buildspec %s", internalBuildspec).Err().Error())
				errs = append(errs, err)
				continue
			}
			buildspec, err := repo.ParseManifest(buildspecData)
			if err != nil {
				LogErr(errors.Annotate(err, "failed to load internal buildspec %s", internalBuildspec).Err().Error())
				errs = append(errs, err)
				continue
			}

			// Create and upload external buildspec.
			LogOut("Attempting to create external buildspec for %s...", internalBuildspec)
			uploadPath := lgs.MakePath(externalBuildspecsGSBucket, internalBuildspec)
			if err := createPublicBuildspec(gsClient, gerritClient, buildspec, uploadPath, b.push); err != nil {
				LogErr(errors.Annotate(err, "failed to create external buildspec %s", internalBuildspec).Err().Error())
				errs = append(errs, err)
			}
		}
	}

	if len(errs) == 0 {
		return nil
	}
	return errors.NewMultiError(errs...)
}
