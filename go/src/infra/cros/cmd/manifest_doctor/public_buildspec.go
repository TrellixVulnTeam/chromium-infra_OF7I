// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/errors"
	luciflag "go.chromium.org/luci/common/flag"
	lgs "go.chromium.org/luci/common/gcloud/gs"

	"infra/cros/internal/gerrit"
	"infra/cros/internal/gs"
	"infra/cros/internal/manifestutil"
	"infra/cros/internal/repo"
)

type publicBuildspec struct {
	subcommands.CommandRunBase
	authFlags                  authcli.Flags
	push                       bool
	watchPaths                 []string
	readFromManifestVersions   bool
	internalBuildspecsGSBucket string
	externalBuildspecsGSBucket string
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
			b.Flags.BoolVar(&b.readFromManifestVersions, "legacy", false,
				"If set, will apply watch paths to the internal "+
					"manifest-versions repository instead of GS.")
			b.Flags.StringVar(&b.internalBuildspecsGSBucket, "internal-bucket",
				internalBuildspecsGSBucketDefault,
				fmt.Sprintf("Internal buildspec bucket. Defaults to %s.", internalBuildspecsGSBucketDefault))
			b.Flags.StringVar(&b.externalBuildspecsGSBucket, "external-bucket",
				externalBuildspecsGSBucketDefault,
				fmt.Sprintf("External buildspec bucket. Defaults to %s.", externalBuildspecsGSBucketDefault))
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

func (b *publicBuildspec) copyExternalBuildspec(ctx context.Context, gsClient gs.Client, gerritClient gerrit.Client, externalBuildspec string) error {
	data, err := gerritClient.DownloadFileFromGitiles(ctx, chromeExternalHost,
		externalManifestVersionsProject, "HEAD", externalBuildspec)
	if err != nil {
		return errors.Annotate(err, "failed to download %s from %s", externalBuildspec, externalManifestVersionsProject).Err()
	}
	if b.push {
		uploadPath := lgs.MakePath(b.externalBuildspecsGSBucket, externalBuildspec)
		if err := gsClient.WriteFileToGS(uploadPath, []byte(data)); err != nil {
			return errors.Annotate(err, "failed to write external buildspec to %s", string(uploadPath)).Err()
		}
	}
	return nil
}

// CreateProjectBuildspec creates a public buildspec as
// outlined in go/per-project-buildspecs.
func (b *publicBuildspec) CreatePublicBuildspecs(ctx context.Context, gsClient gs.Client, gerritClient gerrit.Client) error {
	errs := []error{}
	for _, watchPath := range b.watchPaths {
		var internalBuildspecs []string
		var err error
		if b.readFromManifestVersions {
			LogOut("Looking at https://%s/%s/+/HEAD/%s", chromeInternalHost, internalManifestVersionsProject, watchPath)
			internalBuildspecs, err = gerritClient.ListFiles(ctx, chromeInternalHost, internalManifestVersionsProject, "HEAD", watchPath)
			// Unlike GS, Gerrit lists relative paths so we need to reconstruct them here.
			for i := range internalBuildspecs {
				internalBuildspecs[i] = filepath.Join(watchPath, internalBuildspecs[i])
			}
		} else {
			LogOut("Looking at gs://%s/%s...\n", b.internalBuildspecsGSBucket, watchPath)
			internalBuildspecs, err = gsClient.List(ctx, b.internalBuildspecsGSBucket, watchPath)
		}
		if err != nil {
			LogErr(errors.Annotate(err, "failed to list internal buildspecs for dir %s, skipping...", watchPath).Err().Error())
			errs = append(errs, err)
			continue
		}

		legacyExternalBuildspecMap := make(map[string]bool)
		if b.readFromManifestVersions {
			// If legacy, also fetch list of existing external buildspecs.
			legacyExternalBuildspecs, err := gerritClient.ListFiles(ctx, chromeExternalHost, externalManifestVersionsProject, "HEAD", watchPath)
			if err != nil && !strings.Contains(err.Error(), "NotFound") {
				LogErr(errors.Annotate(err, "failed to list external buildspecs for dir %s in %s, skipping...", watchPath, externalManifestVersionsProject).Err().Error())
				errs = append(errs, err)
				continue
			}
			// Unlike GS, Gerrit lists relative paths so we need to reconstruct them here.
			for _, buildspec := range legacyExternalBuildspecs {
				legacyExternalBuildspecMap[filepath.Join(watchPath, buildspec)] = true
			}
		}

		externalPath := watchPath
		externalBuildspecs, err := gsClient.List(ctx, b.externalBuildspecsGSBucket, externalPath)
		if err != nil {
			LogErr(errors.Annotate(err, "failed to list external buildspecs for dir %s, skipping...", externalPath).Err().Error())
			errs = append(errs, err)
		}
		externalBuildspecMap := make(map[string]bool, len(externalBuildspecs))
		for _, externalBuildspec := range externalBuildspecs {
			externalBuildspecMap[externalBuildspec] = true
		}

		for _, internalBuildspec := range internalBuildspecs {
			externalBuildspec := internalBuildspec
			// Skip buildspecs that are already present.
			if _, ok := externalBuildspecMap[externalBuildspec]; ok {
				continue
			}

			// Load internal buildspec.
			var buildspec *repo.Manifest
			if b.readFromManifestVersions {
				buildspec, err = manifestutil.LoadManifestFromGitiles(ctx, gerritClient,
					chromeInternalHost, internalManifestVersionsProject, "HEAD", internalBuildspec)
			} else {
				internalPath := lgs.MakePath(b.internalBuildspecsGSBucket, internalBuildspec)
				buildspecData, err := gsClient.Read(internalPath)
				if err != nil {
					LogErr(errors.Annotate(err, "failed to read internal buildspec %s", internalBuildspec).Err().Error())
					errs = append(errs, err)
					continue
				}
				buildspec, err = repo.ParseManifest(buildspecData)
			}
			if err != nil {
				LogErr(errors.Annotate(err, "failed to load internal buildspec %s", internalBuildspec).Err().Error())
				errs = append(errs, err)
				continue
			}

			// If legacy, upload the internal buildspec to the internal bucket for internal consistency.
			if b.readFromManifestVersions && b.push {
				uploadPath := lgs.MakePath(b.internalBuildspecsGSBucket, internalBuildspec)
				if err := WriteManifestToGS(gsClient, uploadPath, buildspec); err != nil {
					LogErr(errors.Annotate(err, "failed to write internal buildspec to %s", string(uploadPath)).Err().Error())
					errs = append(errs, err)
					continue
				}
			}

			// If we're reading from legacy and an external buildspec already exists, we should
			// use that instead of creating a new one.
			if _, ok := legacyExternalBuildspecMap[externalBuildspec]; ok {
				if err := b.copyExternalBuildspec(ctx, gsClient, gerritClient, externalBuildspec); err != nil {
					LogErr(err.Error())
					errs = append(errs, err)
					continue
				}
			} else {
				// Create and upload external buildspec.
				LogOut("Attempting to create external buildspec %s for %s...", externalBuildspec, internalBuildspec)
				uploadPath := lgs.MakePath(b.externalBuildspecsGSBucket, externalBuildspec)
				if err := createPublicBuildspec(gsClient, gerritClient, buildspec, uploadPath, b.push); err != nil {
					LogErr(errors.Annotate(err, "failed to create external buildspec %s", externalBuildspec).Err().Error())
					errs = append(errs, err)
				}
			}
		}
	}

	if len(errs) == 0 {
		return nil
	}
	return errors.NewMultiError(errs...)
}
