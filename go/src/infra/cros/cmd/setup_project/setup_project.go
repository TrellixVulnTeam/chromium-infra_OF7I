// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package main

import (
	"context"
	gerrs "errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	gitiles "infra/cros/internal/gerrit"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/api/gerrit"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/hardcoded/chromeinfra"
)

const (
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
	program              string
	project              string
	allProjects          bool
	localManifestBranch  string
	chipset              string
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
			b.Flags.StringVar(&b.localManifestBranch, "branch", "main",
				"Sync the project from the local manifest at the given branch.")
			b.Flags.StringVar(&b.chipset, "chipset", "",
				"Name of the chipset overlay to sync a local manifest from.")
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
	if err := b.setupProject(ctx, authedClient); err != nil {
		LogErr(err.Error())
		return 4
	}

	return 0
}

type localManifest struct {
	project    string
	branch     string
	path       string
	downloadTo string
}

func projectsInProgram(ctx context.Context, authedClient *http.Client, program string) ([]string, error) {
	projects, err := gitiles.Projects(ctx, authedClient, chromeInternalHost)
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

func (b *setupProject) setupProject(ctx context.Context, authedClient *http.Client) error {
	localManifestPath := filepath.Join(b.chromeosCheckoutPath, ".repo/local_manifests")
	// Create local_manifests dir if it does not already exist.
	if err := os.Mkdir(localManifestPath, os.ModePerm); err != nil && !gerrs.Is(err, os.ErrExist) {
		return err
	}

	files := []localManifest{
		{
			project:    fmt.Sprintf("chromeos/program/%s", b.program),
			branch:     b.localManifestBranch,
			path:       "local_manifest.xml",
			downloadTo: fmt.Sprintf("%s_program.xml", b.program),
		},
	}

	var projects []string
	if b.allProjects {
		var err error
		projects, err = projectsInProgram(ctx, authedClient, b.program)
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
		files = append(files, localManifest{
			project:    fmt.Sprintf("chromeos/project/%s/%s", b.program, project),
			branch:     b.localManifestBranch,
			path:       "local_manifest.xml",
			downloadTo: fmt.Sprintf("%s_project.xml", project),
		})
	}

	if b.chipset != "" {
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
		err := gitiles.DownloadFileFromGitilesToPath(ctx, authedClient, chromeInternalHost,
			file.project, file.branch, file.path, downloadPath)

		if err != nil {
			cleanup(writtenFiles)
			errmsg := fmt.Sprintf("error downloading file %s/%s/%s from branch %s",
				chromeInternalHost, file.project, file.path, file.branch)
			return errors.Annotate(err, errmsg).Err()
		}
		writtenFiles = append(writtenFiles, downloadPath)
	}

	LogOut(`Local manifest setup complete, sync new projects with:

repo sync --force-sync -j48`)
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
	opts := chromeinfra.DefaultAuthOptions()
	opts.Scopes = []string{
		gerrit.OAuthScope,
		auth.OAuthScopeEmail,
	}
	s := &setupProjectApplication{
		GetApplication(opts),
		log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds),
		log.New(os.Stderr, "", log.LstdFlags|log.Lmicroseconds)}
	os.Exit(subcommands.Run(s, nil))
}
