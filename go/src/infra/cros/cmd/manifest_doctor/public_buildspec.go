// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package main

import (
	"context"
	"fmt"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/errors"
	luciflag "go.chromium.org/luci/common/flag"
	lgs "go.chromium.org/luci/common/gcloud/gs"

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

	if err := b.CreatePublicBuildspecs(ctx, gsClient); err != nil {
		LogErr(err.Error())
		return 5
	}

	return 0
}

// CreateProjectBuildspec creates a public buildspec as outlined in go/per-project-buildspecs.
func (b *publicBuildspec) createPublicBuildspec(buildspecPath string, gsClient gs.Client) error {
	internalPath := lgs.MakePath(internalBuildspecsGSBucket, buildspecPath)
	buildspecData, err := gsClient.Read(internalPath)
	if err != nil {
		return err
	}
	buildspec, err := repo.ParseManifest(buildspecData)
	if err != nil {
		return err
	}

	// Look at remotes, filter out non public projects.
	publicRemote := make(map[string]bool, len(buildspec.Remotes))
	var publicRemotes []repo.Remote
	for _, remote := range buildspec.Remotes {
		public, ok := remote.GetAnnotation("public")
		publicRemote[remote.Name] = ok && (public == "true")
		if publicRemote[remote.Name] {
			publicRemotes = append(publicRemotes, remote)
		}
	}

	// Verify that the default is not a private remote.
	defaultRemote := buildspec.Default.RemoteName
	if public, ok := publicRemote[defaultRemote]; !(ok && public) {
		return fmt.Errorf("default remote is private")
	}

	var publicProjects []repo.Project
	for _, project := range buildspec.Projects {
		// Check for the (implicit) default remote or a known public remote.
		if public, ok := publicRemote[project.RemoteName]; project.RemoteName == "" || (ok && public) {
			publicProjects = append(publicProjects, project)
		}
	}
	buildspec.Remotes = publicRemotes
	buildspec.Projects = publicProjects
	externalBuildspecData, err := buildspec.WriteToBytes()
	if err != nil {
		return err
	}

	// Upload to external buildspec dir.
	uploadPath := lgs.MakePath(externalBuildspecsGSBucket, buildspecPath)
	if !b.push {
		LogOut("Dry run, not uploading buildspec to %s...", string(uploadPath))
		return nil
	}
	if err := gsClient.WriteFileToGS(uploadPath, externalBuildspecData); err != nil {
		return err
	}
	LogOut("Uploaded buildspec to %s", string(uploadPath))
	return nil
}

// CreateProjectBuildspec creates a public buildspec as
// outlined in go/per-project-buildspecs.
func (b *publicBuildspec) CreatePublicBuildspecs(ctx context.Context, gsClient gs.Client) error {
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

			// Create and upload external buildspec.
			LogOut("Attempting to create external buildspec for %s...", internalBuildspec)
			if err := b.createPublicBuildspec(internalBuildspec, gsClient); err != nil {
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
