// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package main

import (
	"fmt"
	"os"

	"infra/cros/internal/branch"
	"infra/cros/internal/git"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth"
)

func getCmdDeleteBranch(opts auth.Options) *subcommands.Command {
	return &subcommands.Command{
		UsageLine: "delete <options> branchName",
		ShortDesc: "Delete a branch.",
		LongDesc:  "Delete a branch.",
		CommandRun: func() subcommands.CommandRun {
			c := &deleteBranchRun{}
			c.InitFlags(opts)
			return c
		},
	}
}

type deleteBranchRun struct {
	CommonFlags
	// Name of the branch to delete.
	branchName string
}

func (c *deleteBranchRun) validate(args []string) (bool, string) {
	if len(args) < 1 {
		return false, "missing required argument(s)."
	}
	c.branchName = args[0]
	return true, ""
}

// Getters so that functions using the branchCommand interface
// can access CommonFlags in the underlying struct.
func (c *deleteBranchRun) getRoot() string {
	return c.Root
}

func (c *deleteBranchRun) getManifestURL() string {
	return c.ManifestURL
}

func (c *deleteBranchRun) Run(a subcommands.Application, args []string,
	env subcommands.Env) int {
	ret := Run(c, a, args, env)
	if ret != 0 {
		return ret
	}
	if c.Push && !c.Force {
		branch.LogErr("Must set --force to delete remote branches.")
		return 1
	}
	if err := branch.InitWorkingManifest(c.getManifestURL(), "refs/heads/main"); err != nil {
		branch.LogErr("%s\n", err.Error())
		return 1
	}
	defer os.RemoveAll(branch.ManifestCheckout)

	// Need to do this for testing, sadly -- don't want to delete real branches.
	if c.ManifestURL != defaultManifestURL {
		branch.LogErr("Warning: --manifest-url should not be used for branch deletion.\n")
	}

	// Generate git branch names.
	branches := branch.ProjectBranches(c.branchName, "")

	// Delete branches on remote.
	// TODO(@owner): Consider parallelizing this. It's not super important
	// because delete is seldom used.
	retCode := 0
	for _, projectBranch := range branches {
		project := projectBranch.Project
		br := git.NormalizeRef(projectBranch.BranchName)
		remote := branch.WorkingManifest.GetRemoteByName(project.RemoteName)
		if remote == nil {
			// Try and delete as many of the branches as possible, even if some fail.
			branch.LogErr("Remote %s does not exist in working manifest.\n", project.RemoteName)
			retCode = 1
			continue
		}
		projectRemote := fmt.Sprintf("%s/%s", remote.Fetch, project.Name)
		cmd := []string{"push", projectRemote, "--delete", br}
		if !c.Push {
			cmd = append(cmd, "--dry-run")
		}

		_, err := git.RunGit(branch.ManifestCheckout, cmd)
		if err != nil {
			branch.LogErr("Failed to delete branch %s in project %s.\n", br, project.Name)
			// Try and delete as many of the branches as possible, even if some fail.
			retCode = 1
		}
	}

	return retCode
}
