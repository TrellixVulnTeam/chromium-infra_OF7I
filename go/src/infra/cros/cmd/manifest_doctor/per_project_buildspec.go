// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package main

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/data/text"
	"go.chromium.org/luci/common/errors"
	luciflag "go.chromium.org/luci/common/flag"
	lgs "go.chromium.org/luci/common/gcloud/gs"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"infra/cros/internal/branch"
	"infra/cros/internal/gerrit"
	"infra/cros/internal/gs"
	"infra/cros/internal/manifestutil"
	"infra/cros/internal/repo"
)

const (
	chromeExternalHost              = "chromium.googlesource.com"
	chromeInternalHost              = "chrome-internal.googlesource.com"
	manifestInternalProject         = "chromeos/manifest-internal"
	externalManifestVersionsProject = "chromiumos/manifest-versions"
	chromeosProgramPrefix           = "chromeos/program/"
	chromeosProjectPrefix           = "chromeos/project/"
)

var (
	projectRegexp = regexp.MustCompile(`(?P<program>[a-z0-9-]+)/(?P<project>[a-z0-9-]+)$`)
)

type projectBuildspec struct {
	subcommands.CommandRunBase
	authFlags    authcli.Flags
	buildspec    string
	watchPaths   []string
	minMilestone int
	projects     []string
	force        bool
}

func cmdProjectBuildspec(authOpts auth.Options) *subcommands.Command {
	return &subcommands.Command{
		UsageLine: "project-buildspec --buildspec=full/buildspecs/94/14144.0.0-rc2.xml --program=galaxy --project=milkyway",
		ShortDesc: "Create project-specific buildspec(s) for specific project(s)/program(s) and version(s)",
		CommandRun: func() subcommands.CommandRun {
			b := &projectBuildspec{}
			b.authFlags = authcli.Flags{}
			b.authFlags.Register(b.GetFlags(), authOpts)
			b.Flags.StringVar(&b.buildspec, "buildspec", "",
				text.Doc(`
				Path to manifest within manifest-versions repo, e.g.
				full/buildspecs/94/14144.0.0-rc2.xml`))
			b.Flags.Var(luciflag.CommaList(&b.watchPaths), "paths",
				"Comma-separated list of subdirs to watch within the external manifest-versions repo, e.g."+
					"full/buildspecs/,buildspecs/. Each specified subdir must have milestone numbers under "+
					"them, e.g. 95/ and 96/.")
			b.Flags.IntVar(&b.minMilestone, "min_milestone", -1,
				"Minimum milestone of branches to consider within watch paths.")
			b.Flags.BoolVar(&b.force, "force", false,
				"Existing buildspecs will not be regenerated unless --force is set")
			b.Flags.Var(luciflag.CommaList(&b.projects), "projects",
				"Name of the project(s) (e.g. galaxy/milkyway) to create buildspecs for."+
					" Supports wildcards, e.g. galaxy/* or galaxy/milk*")
			return b
		}}
}

func (b *projectBuildspec) validate() error {
	if b.buildspec == "" && len(b.watchPaths) == 0 {
		return fmt.Errorf("--buildspec or --paths required")
	}
	if b.buildspec != "" && len(b.watchPaths) != 0 {
		return fmt.Errorf("--buildspec and --paths cannot be used together")
	}
	if len(b.watchPaths) > 0 && b.minMilestone == -1 {
		return fmt.Errorf("--min_milestone required for --paths")
	}

	if len(b.projects) == 0 {
		return fmt.Errorf("must specify at least one project with --projects")
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

	gerritClient, err := gerrit.NewClient(authedClient)
	if err != nil {
		LogErr(err.Error())
		return 5
	}

	if err := b.CreateBuildspecs(gsClient, gerritClient); err != nil {
		LogErr(err.Error())
		return 6
	}

	return 0
}

// gsProjectPath returns the appropriate GS path for the given project/version.
func gsProjectPath(program, project, buildspec string) lgs.Path {
	relPath := filepath.Join("buildspecs/", buildspec)
	return lgs.MakePath(fmt.Sprintf("chromeos-%s-%s", program, project), relPath)
}

// gsProgramPath returns the appropriate GS path for the given program/version.
func gsProgramPath(program, buildspec string) lgs.Path {
	relPath := filepath.Join("buildspecs/", buildspec)
	return lgs.MakePath(fmt.Sprintf("chromeos-%s", program), relPath)
}

// parseProject takes a project string of the form galaxy/milkway and returns
// the program and project separately.
func parseProject(project string) (string, string, error) {
	toks := projectRegexp.FindStringSubmatch(project)
	if toks == nil {
		return "", "", fmt.Errorf("%s is not a valid project", project)
	}
	return toks[1], toks[2], nil
}

// getProjects filters allProjects for only the projects associated with the
// specified program.
func getProjects(ctx context.Context, gerritClient *gerrit.Client, projectPatterns []string) ([]string, error) {
	// Only fetch all projects if we need to, i.e. if one or more pattern
	// contains a wildcard.
	hasWildcard := false
	for _, pattern := range projectPatterns {
		if strings.Contains(pattern, "*") {
			hasWildcard = true
			break
		}
	}
	if !hasWildcard {
		return projectPatterns, nil
	}

	allProjects, err := gerritClient.Projects(ctx, chromeInternalHost)
	if err != nil {
		return nil, errors.Annotate(err, "failed to fetch all projects").Err()
	}

	patterns := []*regexp.Regexp{}
	for _, pattern := range projectPatterns {
		pattern = strings.ReplaceAll(pattern, "*", ".*") + "$"
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, errors.Annotate(err, "invalid pattern specified in --projects").Err()
		}
		patterns = append(patterns, re)
	}

	projects := []string{}
	for _, project := range allProjects {
		project = strings.TrimPrefix(project, chromeosProjectPrefix)
		for _, pattern := range patterns {
			if pattern.MatchString(project) {
				projects = append(projects, project)
			}
		}
	}
	return projects, nil
}

func (b *projectBuildspec) findBuildspecs(ctx context.Context, gerritClient *gerrit.Client) ([]string, error) {
	// Find buildspecs.
	if len(b.watchPaths) == 0 {
		return []string{b.buildspec}, nil
	}
	var buildspecs []string
	for _, watchPath := range b.watchPaths {
		dirs, err := gerritClient.ListFiles(ctx, chromeExternalHost,
			externalManifestVersionsProject, "HEAD", watchPath)
		if err != nil {
			return nil, err
		}
		for _, dir := range dirs {
			mstone, err := strconv.Atoi(dir)
			if err != nil {
				LogErr("dir %s in %s is not a mstone, skipping...", dir, watchPath)
				continue
			}
			if mstone >= b.minMilestone {
				mstoneDir := filepath.Join(watchPath, dir)
				contents, err := gerritClient.ListFiles(ctx, chromeExternalHost,
					externalManifestVersionsProject, "HEAD", mstoneDir)
				if err != nil {
					return nil, err
				}
				for _, file := range contents {
					buildspecs = append(buildspecs, filepath.Join(mstoneDir, file))
				}
			}
		}
	}
	return buildspecs, nil
}

func (b *projectBuildspec) CreateBuildspecs(gsClient gs.Client, gerritClient *gerrit.Client) error {
	ctx := context.Background()
	buildspecs, err := b.findBuildspecs(ctx, gerritClient)
	if err != nil {
		return err
	}
	if len(buildspecs) == 0 {
		return fmt.Errorf("no buildspecs were found for watch paths %s", strings.Join(b.watchPaths, ","))
	}

	// Resolve projects.
	projects, err := getProjects(ctx, gerritClient, b.projects)
	if err != nil {
		return errors.Annotate(err, "failed to resolve projects").Err()
	}
	if len(projects) == 0 {
		return fmt.Errorf("no projects were found for patterns %s", strings.Join(b.projects, ","))
	}

	// Iterate through all projects/programs and create buildspecs.
	var errs []error
	for _, proj := range projects {
		program, project, err := parseProject(proj)
		if err != nil {
			return errors.Annotate(err, "invalid project %s", proj).Err()
		}
		if err := CreateProjectBuildspecs(program, project, buildspecs, b.force, gsClient, gerritClient); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return errors.NewMultiError(errs...)
}

// CreateProjectBuildspec creates a project/program-specific buildspec as
// outlined in go/per-project-buildspecs.
func CreateProjectBuildspecs(program, project string, buildspecs []string, force bool, gsClient gs.Client, gerritClient *gerrit.Client) error {
	logPrefix := fmt.Sprintf("%s/%s", program, project)

	// Aggregate buildspecs by milestone.
	buildspecsByMilestone := make(map[int][]string)
	for _, buildspec := range buildspecs {
		toks := strings.Split(buildspec, "/")
		baseBuildspec := toks[len(toks)-2] + "/" + toks[len(toks)-1]

		buildspecInfo, err := branch.ParseBuildspec(baseBuildspec)
		if err != nil {
			return err
		}
		if _, ok := buildspecsByMilestone[buildspecInfo.ChromeBranch]; !ok {
			buildspecsByMilestone[buildspecInfo.ChromeBranch] = []string{}
		}
		buildspecsByMilestone[buildspecInfo.ChromeBranch] = append(buildspecsByMilestone[buildspecInfo.ChromeBranch], buildspec)
	}
	ctx := context.Background()
	branches, err := gerritClient.Branches(ctx, chromeInternalHost, manifestInternalProject)
	if err != nil {
		return err
	}

	for milestone, buildspecs := range buildspecsByMilestone {
		var releaseBranch string
		hasPreviousMilestone := false
		for branch := range branches {
			if strings.HasPrefix(branch, fmt.Sprintf("refs/heads/release-R%d-", milestone)) {
				releaseBranch = branch
				break
			}
			// TODO(b/184153693): Remove 96-specific logic once 96 branches and
			// is no longer ToT.
			if strings.HasPrefix(branch, fmt.Sprintf("refs/heads/release-R%d-", milestone-1)) ||
				(milestone == 96 && strings.HasPrefix(branch, fmt.Sprintf("refs/heads/release-R94-"))) {
				hasPreviousMilestone = true
			}
		}
		if releaseBranch == "" {
			if !hasPreviousMilestone {
				return fmt.Errorf("release branch for R%d was not found", milestone)
			}
			// If the release branch for the previous milestone is present but this
			// one isn't, we can assume it hasn't been cut yet and select ToT.
			releaseBranch = "refs/heads/main"
		}

		localManifests := make(map[string]*repo.Manifest)

		programProject := chromeosProgramPrefix + program
		projectProject := chromeosProjectPrefix + program + "/" + project
		for _, project := range []string{projectProject, programProject} {
			// Load the local manifest for the appropriate project/branch.
			localManifests[project], err = manifestutil.LoadManifestFromGitiles(ctx, gerritClient, chromeInternalHost,
				project, releaseBranch, "local_manifest.xml")
			if err != nil {
				if project == programProject {
					LogErr("%s: couldn't load local_manifest.xml, it may not exist for the program so skipping...", logPrefix)
					continue
				}
				return errors.Annotate(err, "%s: error loading tip-of-branch manifest", logPrefix).Err()
			}
		}

		for _, buildspec := range buildspecs {
			publicBuildspecPath := buildspec
			_, err = gerritClient.DownloadFileFromGitiles(ctx, chromeExternalHost,
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
			buildspecManifest, err := manifestutil.LoadManifestFromGitiles(ctx, gerritClient, chromeInternalHost,
				"chromeos/manifest-versions", "HEAD", buildspec)
			if err != nil {
				return errors.Annotate(err, "error loading buildspec manifest").Err()
			}

			projects := map[string]lgs.Path{
				programProject: gsProgramPath(program, buildspec),
				projectProject: gsProjectPath(program, project, buildspec),
			}
			for project, uploadPath := range projects {
				files, err := gsClient.List(ctx, uploadPath.Bucket(), uploadPath.Filename())
				if !force && err == nil && len(files) > 0 {
					// This is an optimization check so don't really care if there's an error.
					LogOut("%s: %s already exists, will not regenerate unless --force is set",
						logPrefix, buildspec)
					continue
				}

				localManifest := localManifests[project]
				if localManifest == nil {
					continue
				}
				// Create the project/program-specific buildspec.
				if err := manifestutil.PinManifestFromManifest(localManifest, buildspecManifest); err != nil {
					switch err.(type) {
					case manifestutil.MissingProjectsError:
						LogOut("%s: missing projects in reference manifest, leaving unpinned: %s", logPrefix,
							err.(manifestutil.MissingProjectsError).MissingProjects)
					default:
						return err
					}
				}

				localManifestRaw, err := localManifest.ToBytes()
				if err != nil {
					return err
				}

				// Upload project buildspec to appropriate GS bucket.
				if err := gsClient.WriteFileToGS(uploadPath, localManifestRaw); err != nil {
					return err
				}
				LogOut("%s: wrote buildspec to %s\n", logPrefix, string(uploadPath))
			}
		}
	}

	return nil
}
