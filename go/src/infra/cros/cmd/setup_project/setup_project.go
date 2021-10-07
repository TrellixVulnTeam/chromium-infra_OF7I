// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package main

import (
	"context"
	gerrs "errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	gitiles "infra/cros/internal/gerrit"
	"infra/cros/internal/gs"
	"infra/cros/internal/osutils"
	"infra/cros/internal/shared"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/common/data/text"
	"go.chromium.org/luci/common/errors"
	luciflag "go.chromium.org/luci/common/flag"
	lgs "go.chromium.org/luci/common/gcloud/gs"
)

const (
	chromeExternalHost       = "https://chromium.googlesource.com"
	chromeInternalHost       = "https://chrome-internal.googlesource.com"
	chromeInternalReviewHost = "https://chrome-internal-review.googlesource.com"
	googlePrivacyPolicy      = "https://policies.google.com/privacy"
	googleTermsOfService     = "https://policies.google.com/terms"
)

var (
	// StdoutLog contains the stdout logger for this package.
	StdoutLog *log.Logger
	// StderrLog contains the stderr logger for this package.
	StderrLog *log.Logger
)

// LogOut logs to stdout.
func LogOut(format string, a ...interface{}) {
	if StdoutLog != nil {
		StdoutLog.Printf(format, a...)
	}
}

// LogErr logs to stderr.
func LogErr(format string, a ...interface{}) {
	if StderrLog != nil {
		StderrLog.Printf(format, a...)
	}
}

type setupProject struct {
	subcommands.CommandRunBase
	gitCookiesPath       string
	chromeosCheckoutPath string
	// Settings for which local manifests to use.
	program     string
	project     string
	allProjects bool
	chipset     string
	otherRepos  []string
	// Modifiers on where to get the local manifests.
	localManifestBranch string
	buildspec           string
}

func cmdSetupProject() *subcommands.Command {
	return &subcommands.Command{
		UsageLine: "setup-project --checkout=/usr/.../chromiumos " +
			"--program=galaxy {--project=milkyway|--all_projects}",
		ShortDesc: "Syncs a ChromiumOS checkout using local_manifests from the specified project.\n" +
			"Google Privacy Policy: " + googlePrivacyPolicy +
			"\nGoogle Terms of Service: " + googleTermsOfService,
		CommandRun: func() subcommands.CommandRun {
			b := &setupProject{}
			b.Flags.StringVar(&b.chromeosCheckoutPath, "checkout", "",
				"Path to a ChromeOS checkout.")
			b.Flags.StringVar(&b.program, "program", "",
				"Program the project belongs to.")
			b.Flags.StringVar(&b.project, "project", "",
				"Project to sync to.")
			b.Flags.BoolVar(&b.allProjects, "all_projects", false,
				"If specified, will include all projects under the specified program.")
			b.Flags.StringVar(&b.chipset, "chipset", "",
				"Name of the chipset overlay to sync a local manifest from.")
			b.Flags.Var(luciflag.CommaList(&b.otherRepos), "other-repos",
				"Other chrome-internal repos to sync a local manifest from, e.g."+
					"chromeos/vendor/qti/camx.")
			b.Flags.StringVar(&b.localManifestBranch, "branch", "main",
				"Sync the project from the local manifest at the given branch.")
			b.Flags.StringVar(&b.buildspec, "buildspec", "",
				text.Doc(`Specific buildspec to sync to, e.g.
				full/buildspecs/94/14144.0.0-rc2.xml. Requires
				per-project buildspecs to have been created for the appropriate
				projects for the appropriate version, see go/per-project-buildspecs.
				If set, takes priority over --branch.`))
			b.Flags.StringVar(&b.gitCookiesPath, "gitcookies", "~/.gitcookies",
				"Path to your .gitcookies file, used to auth with gerrit.")
			return b
		}}
}

func (b *setupProject) validate() error {
	if b.chromeosCheckoutPath == "" {
		return fmt.Errorf("--checkout required")
	} else if _, err := os.Stat(b.chromeosCheckoutPath); gerrs.Is(err, os.ErrNotExist) {
		return fmt.Errorf("path %s does not exist", b.chromeosCheckoutPath)
	} else if err != nil {
		return fmt.Errorf("error validating --chromeos_checkout=%s", b.chromeosCheckoutPath)
	}

	if b.project == "" && !b.allProjects && len(b.otherRepos) == 0 {
		return fmt.Errorf("--project, --all_projects or --other-repos required")
	}
	if b.project != "" && b.allProjects {
		return fmt.Errorf("--project and --all_projects cannot both be set")
	}
	if b.program == "" && b.project != "" {
		return fmt.Errorf("--program must be used with --project")
	}
	if b.program == "" && b.allProjects {
		return fmt.Errorf("--program must be used with --all_projects")
	}

	if b.chipset != "" && b.buildspec != "" {
		return fmt.Errorf("using --buildspec with --chipset is not currently supported")
	}

	return nil
}

func (b *setupProject) resolveGitCookiesPath() (string, error) {
	gitCookiesPath, err := osutils.ResolveHomeRelPath(b.gitCookiesPath)
	if err != nil {
		return "", errors.Annotate(err, "error resolving gitcookies path").Err()
	}
	if !osutils.PathExists(gitCookiesPath) {
		return "", fmt.Errorf("%s does not exist", gitCookiesPath)
	}
	return gitCookiesPath, nil
}

func (b *setupProject) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	StdoutLog = a.(*setupProjectApplication).stdoutLog
	StderrLog = a.(*setupProjectApplication).stderrLog

	LogOut("Google Privacy Policy: %s", googlePrivacyPolicy)
	LogOut("Google Terms of Service: %s", googleTermsOfService)

	if err := b.validate(); err != nil {
		LogErr(err.Error())
		return 1
	}

	ctx := context.Background()
	gsClient, err := gs.NewProdClient(ctx, nil)
	if err != nil {
		LogErr(err.Error())
		return 2
	}

	var gitilesClient gitiles.APIClient
	if b.buildspec == "" {
		gitcookiesPath, err := b.resolveGitCookiesPath()
		if err != nil {
			LogErr(err.Error())
			return 3
		}
		gitilesClient, err = gitiles.NewProdAPIClient(ctx, chromeInternalReviewHost, gitcookiesPath)
		if err != nil {
			LogErr(err.Error())
			return 4
		}
	}

	if err := b.setupProject(ctx, gsClient, gitilesClient); err != nil {
		LogErr(err.Error())
		return 5
	}

	return 0
}

type localManifest struct {
	project string
	branch  string
	path    string
	// If set, file will be sourced from GS instead of from gerrit via the
	// gitiles API.
	gsPath     lgs.Path
	downloadTo string
}

func projectsInProgram(ctx context.Context, gitilesClient gitiles.APIClient, program string) ([]string, error) {
	projects, err := gitilesClient.Projects()
	if err != nil {
		return nil, err
	}

	programProjects := []string{}
	for _, project := range projects {
		prefix := fmt.Sprintf("chromeos/project/%s/", program)
		if strings.HasPrefix(project, prefix) {
			programProjects = append(programProjects, strings.TrimPrefix(project, prefix))
		}
	}
	return programProjects, nil
}

// gsProjectPath returns the appropriate GS path for the given project/version.
func gsProjectPath(program, project, buildspec string) lgs.Path {
	bucket := fmt.Sprintf("chromeos-%s-%s", program, project)
	relPath := filepath.Join("buildspecs/", buildspec)
	return lgs.MakePath(bucket, relPath)
}

// gsProgramPath returns the appropriate GS path for the given program/version.
func gsProgramPath(program, buildspec string) lgs.Path {
	relPath := filepath.Join("buildspecs/", buildspec)
	return lgs.MakePath(fmt.Sprintf("chromeos-%s", program), relPath)
}

func dasherizeProject(name string) string {
	return strings.Join(strings.Split(name, "/"), "-")
}

// gsBuildspecPath returns the appropriate GS path for the given CrOS repo.
func gsBuildspecPath(name, buildspec string) lgs.Path {
	relPath := filepath.Join("buildspecs/", buildspec)
	return lgs.MakePath(dasherizeProject(name), relPath)
}

func (b *setupProject) setupProject(ctx context.Context, gsClient gs.Client, gitilesClient gitiles.APIClient) error {
	localManifestPath := filepath.Join(b.chromeosCheckoutPath, ".repo/local_manifests")
	// Create local_manifests dir if it does not already exist.
	if err := os.Mkdir(localManifestPath, os.ModePerm); err != nil && !gerrs.Is(err, os.ErrExist) {
		return err
	}

	files := []localManifest{}

	var projects []string
	if b.allProjects {
		var err error
		projects, err = projectsInProgram(ctx, gitilesClient, b.program)
		if err != nil {
			return errors.Annotate(err, "error getting all projects for program %s", b.program).Err()
		}
	} else if b.project != "" {
		projects = []string{b.project}
	}

	if len(projects) == 0 && len(b.otherRepos) == 0 {
		return fmt.Errorf("no projects found")
	}
	for _, project := range projects {
		var gspath lgs.Path
		if b.buildspec != "" {
			gspath = gsProjectPath(b.program, project, b.buildspec)
		}
		files = append(files, localManifest{
			project:    fmt.Sprintf("chromeos/project/%s/%s", b.program, project),
			branch:     b.localManifestBranch,
			path:       "local_manifest.xml",
			gsPath:     gspath,
			downloadTo: fmt.Sprintf("%s_project.xml", project),
		})
	}

	var programProject string
	if len(projects) != 0 {
		var gspath lgs.Path
		if b.buildspec != "" {
			gspath = gsProgramPath(b.program, b.buildspec)
		}
		programProject = fmt.Sprintf("chromeos/program/%s", b.program)
		files = append(files, localManifest{
			project:    programProject,
			branch:     b.localManifestBranch,
			path:       "local_manifest.xml",
			gsPath:     gspath,
			downloadTo: fmt.Sprintf("%s_program.xml", b.program),
		})
	}

	if b.chipset != "" && b.buildspec == "" {
		files = append(files,
			localManifest{
				project:    fmt.Sprintf("chromeos/overlays/chipset-%s-private", b.chipset),
				branch:     b.localManifestBranch,
				path:       "local_manifest.xml",
				downloadTo: fmt.Sprintf("%s_chipset.xml", b.chipset),
			},
		)
	}

	if len(b.otherRepos) > 0 {
		for _, project := range b.otherRepos {
			var gspath lgs.Path
			if b.buildspec != "" {
				gspath = gsBuildspecPath(project, b.buildspec)
			}
			files = append(files, localManifest{
				project:    project,
				branch:     b.localManifestBranch,
				path:       "local_manifest.xml",
				downloadTo: fmt.Sprintf("%s_repo.xml", dasherizeProject(project)),
				gsPath:     gspath,
			})
		}
	}

	cleanup := func(files []string) {
		for _, file := range files {
			os.Remove(file)
		}
	}

	writtenFiles := make([]string, 0, len(files))
	// Download each local_manifest.xml.
	for _, file := range files {
		downloadPath := filepath.Join(localManifestPath, file.downloadTo)
		var err error
		var errmsg string
		if string(file.gsPath) != "" {
			err = gsClient.DownloadWithGsutil(ctx, file.gsPath, downloadPath)
			errmsg = fmt.Sprintf("error downloading %s", file.gsPath)
		} else {
			err = gitilesClient.DownloadFileFromGitilesToPath(
				file.project, file.branch, file.path, downloadPath)
			errmsg = fmt.Sprintf("error downloading file %s/%s/%s from branch %s",
				chromeInternalHost, file.project, file.path, file.branch)
		}
		if err != nil && programProject != "" && file.project == programProject {
			// If the program-level buildspec doesn't exist, don't fail.
			// Unless something is very wrong, the project and program
			// buildspecs will be generated together.
			// If the project buildspec doesn't exist, this run will still fail.
			// If the project buildspec doesn't exist, it must just be that
			// this particular program doesn't have a local_manifest.xml.
			if gerrs.Is(err, shared.ErrObjectNotExist) {
				LogOut("program-level manifest doesn't exist, but this is sometimes expected behavior. continuing...")
				continue
			}
		}

		if err != nil {
			cleanup(writtenFiles)
			return errors.Annotate(err, errmsg).Err()
		}
		LogOut("installed %s", file.downloadTo)
		writtenFiles = append(writtenFiles, downloadPath)
	}

	if b.buildspec != "" {
		LogOut("You are syncing to a per-project/program buildspec, make sure " +
			"that you've run the equivalent of:" +
			"\n\nrepo init -u https://chromium.googlesource.com/chromiumos/manifest-versions -b main" +
			fmt.Sprintf(" -m %s", b.buildspec) + "\n\n")
		LogOut("Local manifest setup complete, sync new projects with:" +
			"\n\nrepo sync --force-sync -j48")
	} else {
		LogOut("Local manifest setup complete, sync new projects with:" +
			"\n\nrepo sync --force-sync -j48")
	}

	return nil
}

// GetApplication returns an instance of the application.
func GetApplication() *subcommands.DefaultApplication {
	return &subcommands.DefaultApplication{
		Name: "setup_project",
		Commands: []*subcommands.Command{
			cmdSetupProject(),
		},
	}
}

type setupProjectApplication struct {
	*subcommands.DefaultApplication
	stdoutLog *log.Logger
	stderrLog *log.Logger
}

func main() {
	s := &setupProjectApplication{
		GetApplication(),
		log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds),
		log.New(os.Stderr, "", log.LstdFlags|log.Lmicroseconds)}
	os.Exit(subcommands.Run(s, nil))
}
