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

	"cloud.google.com/go/storage"
	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/api/gerrit"
	"go.chromium.org/luci/common/data/text"
	"go.chromium.org/luci/common/errors"
	lgs "go.chromium.org/luci/common/gcloud/gs"
	"go.chromium.org/luci/hardcoded/chromeinfra"
)

const (
	chromeExternalHost = "chromium.googlesource.com"
	chromeInternalHost = "chrome-internal.googlesource.com"
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
	authFlags            authcli.Flags
	chromeosCheckoutPath string
	// Settings for which local manifests to use.
	program     string
	project     string
	allProjects bool
	chipset     string
	// Modifiers on where to get the local manifests.
	localManifestBranch string
	buildspec           string
}

func cmdSetupProject(authOpts auth.Options) *subcommands.Command {
	return &subcommands.Command{
		UsageLine: "setup-project --checkout=/usr/.../chromiumos " +
			"--program=galaxy {--project=milkyway|--all_projects}",
		ShortDesc: "Syncs a ChromiumOS checkout using local_manifests from the specified project.",
		CommandRun: func() subcommands.CommandRun {
			b := &setupProject{}
			b.authFlags = authcli.Flags{}
			b.authFlags.Register(b.GetFlags(), authOpts)
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
			b.Flags.StringVar(&b.localManifestBranch, "branch", "main",
				"Sync the project from the local manifest at the given branch.")
			b.Flags.StringVar(&b.buildspec, "buildspec", "",
				text.Doc(`Specific buildspec to sync to, e.g.
				full/buildspecs/94/14144.0.0-rc2.xml. Requires
				per-project buildspecs to have been created for the appropriate
				projects for the appropriate version, see go/per-project-buildspecs.
				If set, takes priority over --branch.`))
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

	if b.project == "" && !b.allProjects {
		return fmt.Errorf("--project or --all_projects required")
	}
	if b.program != "" && b.allProjects {
		return fmt.Errorf("--program and --all_projects cannot both be set")
	}

	if b.chipset != "" && b.buildspec != "" {
		return fmt.Errorf("using --buildspec with --chipset is not currently supported")
	}

	return nil
}

func (b *setupProject) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	StdoutLog = a.(*setupProjectApplication).stdoutLog
	StderrLog = a.(*setupProjectApplication).stderrLog

	if err := b.validate(); err != nil {
		LogErr(err.Error())
		return 1
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

	gitilesClient, err := gitiles.NewClient(authedClient)
	if err != nil {
		LogErr(err.Error())
		return 5
	}

	if err := b.setupProject(ctx, gsClient, gitilesClient); err != nil {
		LogErr(err.Error())
		return 6
	}

	return 0
}

type localManifest struct {
	// If blank, chromeInternalHost will be used.
	host    string
	project string
	branch  string
	path    string
	// If set, file will be sourced from GS instead of from gerrit via the
	// gitiles API.
	gsPath     lgs.Path
	downloadTo string
}

func projectsInProgram(ctx context.Context, gitilesClient *gitiles.Client, program string) ([]string, error) {
	projects, err := gitilesClient.Projects(ctx, chromeInternalHost)
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

func (b *setupProject) setupProject(ctx context.Context, gsClient gs.Client, gitilesClient *gitiles.Client) error {
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

	if len(projects) == 0 {
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

	var gspath lgs.Path
	if b.buildspec != "" {
		gspath = gsProgramPath(b.program, b.buildspec)
	}
	programProject := fmt.Sprintf("chromeos/program/%s", b.program)
	files = append(files, localManifest{
		project:    programProject,
		branch:     b.localManifestBranch,
		path:       "local_manifest.xml",
		gsPath:     gspath,
		downloadTo: fmt.Sprintf("%s_program.xml", b.program),
	})

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
			err = gsClient.Download(file.gsPath, downloadPath)
			if err != nil && file.project == programProject {
				// If the program-level buildspec doesn't exist, don't fail.
				// Unless something is very wrong, the project and program
				// buildspecs will be generated together.
				// If the project buildspec doesn't exist, this run will still fail.
				// If the project buildspec doesn't exist, it must just be that
				// this particular program doesn't have a local_manifest.xml.
				if gerrs.Is(err, storage.ErrObjectNotExist) {
					LogOut("program-level manifest doesn't exist, but this is sometimes expected behavior. continuing...")
					continue
				}
			}
			errmsg = fmt.Sprintf("error downloading %s", file.gsPath)
		} else {
			host := chromeInternalHost
			if file.host != "" {
				host = file.host
			}
			err = gitilesClient.DownloadFileFromGitilesToPath(ctx, host,
				file.project, file.branch, file.path, downloadPath)
			errmsg = fmt.Sprintf("error downloading file %s/%s/%s from branch %s",
				chromeInternalHost, file.project, file.path, file.branch)
		}

		if err != nil {
			cleanup(writtenFiles)
			return errors.Annotate(err, errmsg).Err()
		}
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
func GetApplication(authOpts auth.Options) *subcommands.DefaultApplication {
	return &subcommands.DefaultApplication{
		Name: "setup_project",
		Commands: []*subcommands.Command{
			authcli.SubcommandInfo(authOpts, "auth-info", false),
			authcli.SubcommandLogin(authOpts, "auth-login", false),
			authcli.SubcommandLogout(authOpts, "auth-logout", false),
			cmdSetupProject(authOpts),
		},
	}
}

type setupProjectApplication struct {
	*subcommands.DefaultApplication
	stdoutLog *log.Logger
	stderrLog *log.Logger
}

func main() {
	opts := auth.Options{
		TokenServerHost: chromeinfra.TokenServerHost,
		ClientID:        "300815340602-o77gh6n19i8796dqc9dsu99vcj23jf3o.apps.googleusercontent.com",
		ClientSecret:    "QshHhnal9bMGsLFcIENHz-QG",
		SecretsDir:      chromeinfra.SecretsDir(),
	}
	opts.Scopes = []string{
		gerrit.OAuthScope,
		auth.OAuthScopeEmail,
	}
	opts.Scopes = append(opts.Scopes, lgs.ReadOnlyScopes...)
	s := &setupProjectApplication{
		GetApplication(opts),
		log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds),
		log.New(os.Stderr, "", log.LstdFlags|log.Lmicroseconds)}
	os.Exit(subcommands.Run(s, nil))
}
