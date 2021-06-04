// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package main

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/data/text"
	"go.chromium.org/luci/common/errors"
	lgs "go.chromium.org/luci/common/gcloud/gs"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"infra/cros/internal/branch"
	"infra/cros/internal/gerrit"
	"infra/cros/internal/gs"
	"infra/cros/internal/manifestutil"
)

const (
	chromeExternalHost      = "chromium.googlesource.com"
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

	gsClient, err := gs.NewProdClient(ctx, authedClient)
	if err != nil {
		LogErr(err.Error())
		return 4
	}

	if err := b.CreateProjectBuildspec(authedClient, gsClient); err != nil {
		LogErr(err.Error())
		return 5
	}

	return 0
}

// gsPath returns the appropriate GS path for the given project/version.
func gsPath(projectName, buildspec string) (lgs.Path, error) {
	if strings.HasPrefix(projectName, "chromeos/project") {
		tokens := strings.Split(projectName, "/")
		if len(tokens) >= 4 {
			bucket := fmt.Sprintf("chromeos-%s-%s", tokens[2], tokens[3])
			relPath := filepath.Join("buildspecs/", buildspec)
			return lgs.MakePath(bucket, relPath), nil
		}
	}
	// TODO(b:187795796): Add support for program repos, once GS buckets are
	// located/created.
	return "", fmt.Errorf("unsupported project name")
}

// CreateProjectBuildspec creates a project/program-specific buildspec as
// outlined in go/per-project-buildspecs.
func (b *projectBuildspec) CreateProjectBuildspec(authedClient *http.Client, gsClient gs.Client) error {
	buildspecInfo, err := branch.ParseBuildspec(b.buildspec)
	if err != nil {
		return err
	}
	ctx := context.Background()
	branches, err := gerrit.Branches(ctx, authedClient, chromeInternalHost, manifestInternalProject)
	if err != nil {
		return err
	}

	var releaseBranch string
	for branch := range branches {
		if strings.HasPrefix(branch, fmt.Sprintf("refs/heads/release-R%d-", buildspecInfo.ChromeBranch)) {
			releaseBranch = branch
			break
		}
	}
	if releaseBranch == "" {
		return fmt.Errorf("release branch for R%d was not found", buildspecInfo.ChromeBranch)
	}

	// Load the local manifest for the appropriate project/branch.
	localManifest, err := manifestutil.LoadManifestFromGitiles(ctx, authedClient, chromeInternalHost,
		b.project, releaseBranch, "local_manifest.xml")
	if err != nil {
		return errors.Annotate(err, "error loading tip-of-branch manifest").Err()
	}

	publicBuildspecPath := "full/buildspecs/" + b.buildspec
	_, err = gerrit.DownloadFileFromGitiles(ctx, authedClient, chromeExternalHost,
		"chromiumos/manifest-versions", "HEAD", publicBuildspecPath)
	if err != nil {
		errorCode, ok := status.FromError(err)
		if ok && errorCode.Code() == codes.NotFound {
			publicBuildspecURL := fmt.Sprintf("%s/chromiumos/manifest-versions/%s", chromeExternalHost, publicBuildspecPath)
			LogErr("Warning: A public buildspec does not exist at %s, so this "+
				"buildspec will not be all that useful to partners.", publicBuildspecURL)
		}
		// Otherwise, ignore the error, as this check isn't critical to the overall
		// success of the invocation.
	}

	// Load the internal buildspec.
	buildspecManifest, err := manifestutil.LoadManifestFromGitiles(ctx, authedClient, chromeInternalHost,
		"chromeos/manifest-versions", "HEAD", "buildspecs/"+b.buildspec)
	if err != nil {
		return errors.Annotate(err, "error loading buildspec manifest").Err()
	}

	// Create the project/program-specific buildspec.
	if err := manifestutil.PinManifestFromManifest(localManifest, buildspecManifest); err != nil {
		switch err.(type) {
		case manifestutil.MissingProjectsError:
			LogOut("missing projects in reference manifest, leaving unpinned: %s", err.(manifestutil.MissingProjectsError).MissingProjects)
		default:
			return err
		}
	}

	localManifestRaw, err := localManifest.ToBytes()
	if err != nil {
		return err
	}

	// Upload project buildspec to appropriate GS bucket.
	uploadPath, err := gsPath(b.project, b.buildspec)
	if err != nil {
		return err
	}

	if err := gsClient.WriteFileToGS(uploadPath, localManifestRaw); err != nil {
		return err
	}
	LogOut("wrote buildspec to %s\n", string(uploadPath))

	return nil
}
