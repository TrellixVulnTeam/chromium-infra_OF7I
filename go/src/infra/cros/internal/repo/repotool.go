// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package repo contains functions for interacting with manifests and the
// repo tool.
package repo

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"infra/cros/internal/cmd"
	"infra/cros/internal/git"
)

const (
	depotToolsURL string = "https://chromium.googlesource.com/chromium/tools/depot_tools"
)

var (
	// CommandRunnerImpl exists for testing purposes.
	CommandRunnerImpl cmd.CommandRunner = cmd.RealCommandRunner{}
)

// Ensure that the repo tool is installed. Returns path of tool, cleanup and error.
func EnsureRepoTool() (string, func(), error) {
	path, err := exec.LookPath("repo")
	// Repo is already on the path.
	if err == nil {
		return path, func() {}, nil
	}

	dir, err := ioutil.TempDir("", "depot_tools")
	cleanup := func() {
		os.RemoveAll(dir)
	}
	if err != nil {
		return "", cleanup, err
	}

	if err := git.Clone(depotToolsURL, dir); err != nil {
		return "", cleanup, err
	}
	return filepath.Join(dir, "repo"), cleanup, nil
}

type InitArgs struct {
	// --manifest-url/-u value, if any.
	ManifestURL string
	// --manifest-branch/-b value, if any.
	ManifestBranch string
	// --manifest-name/-m value, if any.
	ManifestFile string
}

// Init runs `repo init`.
func Init(ctx context.Context, path, repoToolPath string, initArgs InitArgs) error {
	cmd := []string{"init"}
	if initArgs.ManifestURL != "" {
		cmd = append(cmd, []string{"-u", initArgs.ManifestURL}...)
	}
	if initArgs.ManifestBranch != "" {
		cmd = append(cmd, []string{"-b", initArgs.ManifestBranch}...)
	}
	if initArgs.ManifestFile != "" {
		cmd = append(cmd, []string{"-m", initArgs.ManifestFile}...)
	}
	var stdoutBuf, stderrBuf bytes.Buffer
	return CommandRunnerImpl.RunCommand(ctx, &stdoutBuf, &stderrBuf, path, repoToolPath, cmd...)
}

// Sync runs `repo sync`.
func Sync(ctx context.Context, path, repoToolPath string) error {
	var stdoutBuf, stderrBuf bytes.Buffer
	return CommandRunnerImpl.RunCommand(ctx, &stdoutBuf, &stderrBuf, path, repoToolPath, "sync")
}

// GetRepoToSourceRoot gets the mapping of Gerrit project to Chromium OS source tree path.
func GetRepoToSourceRoot(chromiumosCheckout, repoToolPath string) (map[string]string, error) {
	repoToSrcRoot := make(map[string]string)
	wd, err := os.Getwd()
	if err != nil {
		return repoToSrcRoot, fmt.Errorf("could not get working dir, %v", err)
	}
	if err := os.Chdir(chromiumosCheckout); err != nil {
		return repoToSrcRoot, fmt.Errorf("failed changing dir, %v", err)
	}
	defer func() {
		if err := os.Chdir(wd); err != nil {
			log.Fatalf("could not change working dir, %s", err)
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var stdoutBuf, stderrBuf bytes.Buffer
	if err := CommandRunnerImpl.RunCommand(ctx, &stdoutBuf, &stderrBuf, chromiumosCheckout, repoToolPath, "list"); err != nil {
		log.Printf("Error from repo.\nstdout =\n%s\n\nstderr=\n%s", stdoutBuf.String(), stderrBuf.String())
		return repoToSrcRoot, err
	}
	repos := strings.Split(stdoutBuf.String(), "\n")
	if len(repos) < 1 {
		return repoToSrcRoot, fmt.Errorf("expected to find at least one repo mappings. Instead, only found [%v]", repos)
	}
repoLoop:
	for _, r := range repos {
		if r == "" {
			break repoLoop
		}
		split := strings.Split(r, ":")
		if len(split) != 2 {
			return repoToSrcRoot, fmt.Errorf("unexpected line format [%s]", r)
		}
		repoName := strings.TrimSpace(split[1])
		srcRoot := strings.TrimSpace(split[0])
		repoToSrcRoot[repoName] = srcRoot
	}
	return repoToSrcRoot, nil
}
