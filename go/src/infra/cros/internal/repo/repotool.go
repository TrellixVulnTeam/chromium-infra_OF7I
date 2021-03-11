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
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

var (
	commandRunnerImpl commandRunner = realCommandRunner{}
)

type commandRunner interface {
	runCommand(ctx context.Context, stdoutBuf, stderrBuf *bytes.Buffer, name string, args ...string) error
}

type realCommandRunner struct{}

func (c realCommandRunner) runCommand(ctx context.Context, stdoutBuf, stderrBuf *bytes.Buffer, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = stdoutBuf
	cmd.Stderr = stderrBuf
	return cmd.Run()
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
	if err := commandRunnerImpl.runCommand(ctx, &stdoutBuf, &stderrBuf, repoToolPath, "list"); err != nil {
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
