// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package testutil contains various util functions for testing.
package testutil

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"regexp"

	"infra/cros/internal/cmd"
	"infra/cros/internal/git"
	"infra/cros/internal/util"

	"go.chromium.org/luci/common/errors"
)

var (
	// CommandRunnerImpl is the command runner impl currently being used by the
	// package. Exists for testing purposes.
	CommandRunnerImpl cmd.CommandRunner = cmd.RealCommandRunner{}
)

// AssertContentsEqual checks that there's no difference between two directories/files.
func AssertContentsEqual(pathA, pathB string) error {
	ai, err := os.Stat(pathA)
	if err != nil {
		return errors.Annotate(err, "failed to stat %s", pathA).Err()
	}
	bi, err := os.Stat(pathB)
	if err != nil {
		return errors.Annotate(err, "failed to stat %s", pathB).Err()
	}
	// diff -rq works for dirs and files, but only if both operands are the same.
	if ai.IsDir() && !bi.IsDir() {
		return fmt.Errorf("%s is a dir and %s is a file", pathA, pathB)
	}
	if !ai.IsDir() && bi.IsDir() {
		return fmt.Errorf("%s is a file and %s is a dir", pathA, pathB)
	}

	ctx := context.Background()
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd := []string{"-rq", pathA, pathB}
	if err = CommandRunnerImpl.RunCommand(ctx, &stdoutBuf, &stderrBuf, "", "diff", cmd...); err != nil {
		if err.Error() == "exit status 1" {
			return fmt.Errorf("files differ: %s", stdoutBuf.String())
		}
		return fmt.Errorf("error running diff")
	}
	return err
}

// AssertGitBranches asserts that the git repo has the given branches (it may have others, too).
func AssertGitBranches(gitRepo string, branches []string) error {
	actual, err := git.MatchBranchNameWithNamespace(gitRepo, regexp.MustCompile(".*"), regexp.MustCompile("refs/heads/"))
	if err != nil {
		return errors.Annotate(err, "error getting branches").Err()
	}
	if !util.UnorderedContains(actual, branches) {
		return fmt.Errorf("project branch mismatch. expected: %v got %v", branches, actual)
	}
	return nil
}

// AssertGitBranchesExact asserts that the git repo has only the correct branches.
func AssertGitBranchesExact(gitRepo string, branches []string) error {
	actual, err := git.MatchBranchNameWithNamespace(gitRepo, regexp.MustCompile(".*"), regexp.MustCompile("refs/heads/"))
	if err != nil {
		return errors.Annotate(err, "error getting branches").Err()
	}
	// Remove duplicates from branches. This is OK because branch names are unique identifiers
	// and so having a branch name twice in branches doesn't mean anything special.
	branchMap := make(map[string]bool)
	for _, branch := range branches {
		branchMap[branch] = true
	}
	branches = []string{}
	for branch := range branchMap {
		branches = append(branches, branch)
	}
	if !util.UnorderedEqual(actual, branches) {
		return fmt.Errorf("project branch mismatch. expected: %v got %v", branches, actual)
	}
	return nil
}
