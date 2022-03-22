// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package main

import (
	"context"
	gerrs "errors"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/data/text"
	"go.chromium.org/luci/common/errors"
	luciflag "go.chromium.org/luci/common/flag"
	lgs "go.chromium.org/luci/common/gcloud/gs"

	"infra/cros/internal/branch"
	"infra/cros/internal/gerrit"
	"infra/cros/internal/gs"
	"infra/cros/internal/manifestutil"
	"infra/cros/internal/repo"
)

const (
	chromeosProgramPrefix = "chromeos/program/"
	chromeosProjectPrefix = "chromeos/project/"
)

var (
	projectRegexp = regexp.MustCompile(`(?P<program>[a-z0-9-]+)/(?P<project>[a-z0-9-]+)$`)
)

type MissingLocalManifestError struct {
	project string
	err     error
}

func (e MissingLocalManifestError) Error() string {
	return fmt.Sprintf("no local_manifest could be found for %s: %v", e.project, e.err)
}

func (e MissingLocalManifestError) Is(target error) bool {
	_, ok := target.(*MissingLocalManifestError)
	return ok
}

type projectBuildspec struct {
	subcommands.CommandRunBase
	authFlags                  authcli.Flags
	buildspec                  string
	watchPaths                 []string
	minMilestone               int
	projects                   []string
	otherRepos                 []string
	force                      bool
	ttl                        int
	push                       bool
	internalBuildspecsGSBucket string
	externalBuildspecsGSBucket string
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
			b.Flags.BoolVar(&b.push, "push", false,
				"Whether or not to push files to GS.")
			b.Flags.BoolVar(&b.force, "force", false,
				"Existing buildspecs will not be regenerated unless --force is set")
			b.Flags.Var(luciflag.CommaList(&b.projects), "projects",
				"Comma-separated list of projects (e.g. galaxy/milkyway) to create buildspecs for."+
					" Supports wildcards, e.g. galaxy/* or galaxy/milk*.")
			b.Flags.Var(luciflag.CommaList(&b.otherRepos), "other-repos",
				"Comma-separated list of repositories (e.g. chromeos/vendor/qti/camx) to create buildspecs for."+
					" Each repository must have a local_manifest.xml file at HEAD."+
					" Buildspecs will be uploaded to the corresponding gs bucket, e.g."+
					"gs://chromeos-vendor-qti-camx.")
			b.Flags.IntVar(&b.ttl, "ttl", -1,
				"TTL (in days) of newly generated buildspecs. If not set, no TTL will be set.")
			b.Flags.StringVar(&b.internalBuildspecsGSBucket, "internal-bucket",
				internalBuildspecsGSBucketDefault,
				fmt.Sprintf("Internal buildspec bucket. Defaults to %s.", internalBuildspecsGSBucketDefault))
			b.Flags.StringVar(&b.externalBuildspecsGSBucket, "external-bucket",
				externalBuildspecsGSBucketDefault,
				fmt.Sprintf("External buildspec bucket. Defaults to %s.", externalBuildspecsGSBucketDefault))
			return b
		}}
}

func (b *projectBuildspec) validate() error {
	if b.buildspec == "" && len(b.watchPaths) == 0 {
		return gerrs.New("--buildspec or --paths required")
	}
	if b.buildspec != "" && len(b.watchPaths) != 0 {
		return gerrs.New("--buildspec and --paths cannot be used together")
	}
	if len(b.watchPaths) > 0 && b.minMilestone == -1 {
		return gerrs.New("--min_milestone required for --paths")
	}

	if len(b.projects) == 0 && len(b.otherRepos) == 0 {
		return gerrs.New("must specify at least one project with --projects or --other-repos")
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

// gsProjectPath returns the appropriate GS path for the given project.
func gsProjectPath(program, project string) lgs.Path {
	return lgs.MakePath(fmt.Sprintf("chromeos-%s-%s", program, project), "buildspecs/")
}

// gsProgramPath returns the appropriate GS path for the given program.
func gsProgramPath(program string) lgs.Path {
	return lgs.MakePath(fmt.Sprintf("chromeos-%s", program), "buildspecs/")
}

// gsBuildspecPath returns the appropriate GS path for the given CrOS repo.
func gsBuildspecPath(name string) lgs.Path {
	bucket := strings.Join(strings.Split(name, "/"), "-")
	return lgs.MakePath(bucket, "buildspecs/")
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
// specified program. It also returns a bool specifying whether or not any of
// the patterns included a wildcard, which is used later on in handling errors.
func getProjects(ctx context.Context, gerritClient gerrit.Client, projectPatterns []string) ([]string, bool, error) {
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
		return projectPatterns, hasWildcard, nil
	}

	allProjects, err := gerritClient.Projects(ctx, chromeInternalHost)
	if err != nil {
		return nil, hasWildcard, errors.Annotate(err, "failed to fetch all projects").Err()
	}

	patterns := []*regexp.Regexp{}
	for _, pattern := range projectPatterns {
		pattern = strings.ReplaceAll(pattern, "*", ".*") + "$"
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, hasWildcard, errors.Annotate(err, "invalid pattern specified in --projects").Err()
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
	return projects, hasWildcard, nil
}

func (b *projectBuildspec) findBuildspecs(ctx context.Context, gsClient gs.Client) ([]string, error) {
	// Find buildspecs.
	if len(b.watchPaths) == 0 {
		return []string{b.buildspec}, nil
	}
	var errs []error
	var buildspecs []string
	for _, watchPath := range b.watchPaths {
		files, err := gsClient.List(ctx, b.externalBuildspecsGSBucket, watchPath)
		if err != nil {
			fullPath := fmt.Sprintf("gs://%s/%s", b.externalBuildspecsGSBucket, watchPath)
			LogErr("failed to list %s, skipping...", fullPath)
			errs = append(errs, errors.Annotate(err, "failed to list %s", fullPath).Err())
			continue
		}
		for _, file := range files {
			dir := filepath.Dir(strings.TrimPrefix(file, watchPath))
			mstone, err := strconv.Atoi(dir)
			if err != nil {
				LogErr("%s in %s is not a mstone, skipping...", dir, watchPath)
				continue
			}
			if mstone >= b.minMilestone && filepath.Base(file) != "" {
				buildspecs = append(buildspecs, file)
			}
		}
	}
	if len(errs) == 0 {
		return buildspecs, nil
	}
	return buildspecs, errors.NewMultiError(errs...)
}

func (b *projectBuildspec) CreateBuildspecs(gsClient gs.Client, gerritClient gerrit.Client) error {
	ctx := context.Background()
	buildspecs, err := b.findBuildspecs(ctx, gsClient)
	if err != nil {
		return err
	}
	if len(buildspecs) == 0 {
		return fmt.Errorf("no buildspecs were found for watch paths %s", strings.Join(b.watchPaths, ","))
	}

	// Resolve projects.
	projects, hasWildcard, err := getProjects(ctx, gerritClient, b.projects)
	if err != nil {
		return errors.Annotate(err, "failed to resolve projects").Err()
	}
	var errs []error
	if len(b.projects) > 0 && len(projects) == 0 {
		errs = append(errs, fmt.Errorf("no projects were found for patterns %s", strings.Join(b.projects, ",")))
	}

	// Iterate through all projects/programs and create buildspecs.
	projectConfig := map[string]projectBuildspecConfig{}
	for _, proj := range projects {
		program, project, err := parseProject(proj)
		if err != nil {
			return errors.Annotate(err, "invalid project %s", proj).Err()
		}

		programProject := chromeosProgramPrefix + program
		projectProject := chromeosProjectPrefix + program + "/" + project
		projectConfig[programProject] = projectBuildspecConfig{
			uploadPath: gsProgramPath(program),
			optional:   true,
			logPrefix:  program + ": ",
		}
		projectConfig[projectProject] = projectBuildspecConfig{
			uploadPath: gsProjectPath(program, project),
			optional:   false,
			logPrefix:  fmt.Sprintf("%s/%s: ", program, project),
		}
	}

	if err := b.CreateProjectBuildspecs(projectConfig, buildspecs, b.push, b.force, b.ttl, gsClient, gerritClient); err != nil {
		// If the projects were not all explicitly specified (i.e. some
		// projects were selected with a wildcard) we shouldn't fail if
		// some project does not have a local manifest.
		if errors.Contains(err, &MissingLocalManifestError{}) && hasWildcard {
			LogErr(err.Error())
		} else {
			errs = append(errs, err)
		}
	}

	// Process --other-repos.
	otherProjects := map[string]projectBuildspecConfig{}
	for _, project := range b.otherRepos {
		otherProjects[project] = projectBuildspecConfig{
			uploadPath: gsBuildspecPath(project),
			optional:   false,
		}
	}
	if err := b.CreateProjectBuildspecs(otherProjects, buildspecs, b.push, b.force, b.ttl, gsClient, gerritClient); err != nil {
		errs = append(errs, err)
	}
	if len(errs) == 0 {
		return nil
	}
	return errors.NewMultiError(errs...)
}

type projectBuildspecConfig struct {
	uploadPath lgs.Path
	optional   bool
	logPrefix  string
}

// CreateProjectBuildspec creates local-manifest-specific buildspecs as
// outlined in go/per-project-buildspecs.
// Projects is a map between project name and config, e.g.
// chromeos/project/galaxy/milkyway : {gs://chromeos-galaxy-milkyway/, true}
func (b *projectBuildspec) CreateProjectBuildspecs(projects map[string]projectBuildspecConfig, buildspecs []string, push, force bool, ttl int, gsClient gs.Client, gerritClient gerrit.Client) error {
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

	// Process the projects in alphabetical order for consistent logging.
	projectNames := make([]string, 0, len(projects))
	for k := range projects {
		projectNames = append(projectNames, k)
	}
	sort.Strings(projectNames)

	var errs []error
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

		for _, project := range projectNames {
			config := projects[project]
			// Load the local manifest for the appropriate project/branch.
			localManifest, err := manifestutil.LoadManifestFromGitiles(ctx, gerritClient, chromeInternalHost,
				project, releaseBranch, "local_manifest.xml")
			if err != nil {
				if config.optional {
					LogErr("%scouldn't load local_manifest.xml for %s, marked as optional so skipping...", config.logPrefix, project)
					continue
				}
				err = MissingLocalManifestError{
					project: project,
					err:     err,
				}
				errs = append(errs, errors.Annotate(err, "%serror loading tip-of-branch manifest", config.logPrefix).Err())
			} else {
				localManifests[project] = localManifest
			}
		}

		for _, buildspec := range buildspecs {
			publicBuildspecPath := lgs.MakePath(b.externalBuildspecsGSBucket, buildspec)

			_, err = gsClient.Read(publicBuildspecPath)
			if err != nil {
				if gerrs.Is(err, storage.ErrObjectNotExist) {
					LogErr("Warning: A public buildspec does not exist at %s, so this "+
						"buildspec will not be all that useful to partners.", string(publicBuildspecPath))
				}
				// Otherwise, ignore the error, as this check isn't critical to the overall
				// success of the invocation.
			}

			// Load the internal buildspec.
			privateBuildspecPath := lgs.MakePath(b.internalBuildspecsGSBucket, buildspec)
			buildspecManifest, err := manifestutil.LoadManifestFromGS(ctx, gsClient, privateBuildspecPath)
			if err != nil {
				return errors.Annotate(err, "error loading buildspec manifest").Err()
			}

			for project, localManifest := range localManifests {
				config := projects[project]
				uploadPath := config.uploadPath.Concat(buildspec)
				files, err := gsClient.List(ctx, uploadPath.Bucket(), uploadPath.Filename())
				if !force && err == nil && len(files) > 0 {
					// This is an optimization check so don't really care if there's an error.
					LogOut("%s%s already exists, will not regenerate unless --force is set",
						config.logPrefix, buildspec)
					continue
				}

				// Create the project/program-specific buildspec.
				if err := manifestutil.PinManifestFromManifest(localManifest, buildspecManifest); err != nil {
					switch err.(type) {
					case manifestutil.MissingProjectsError:
						LogOut("%smissing projects in reference manifest, leaving unpinned: %s", config.logPrefix,
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
				if push {
					if err := gsClient.WriteFileToGS(uploadPath, localManifestRaw); err != nil {
						return err
					}
					LogOut("%swrote buildspec to %s\n", config.logPrefix, string(uploadPath))
					// Set TTL if appropriate.
					if ttl > 0 {
						if err := gsClient.SetTTL(ctx, uploadPath, time.Duration(ttl*24*int(time.Hour))); err != nil {
							return errors.Annotate(err, "error setting ttl").Err()
						}
					}
				} else {
					LogOut("%sdry run, would have written buildspec to %s\n", config.logPrefix, string(uploadPath))
				}
			}
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return errors.NewMultiError(errs...)
}
