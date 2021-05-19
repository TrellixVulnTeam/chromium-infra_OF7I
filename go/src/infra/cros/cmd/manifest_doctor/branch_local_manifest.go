// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package main

import (
	gerrs "errors"
	"fmt"
	"os"
	"path/filepath"

	"infra/cros/internal/branch"
	"infra/cros/internal/git"
	"infra/cros/internal/manifestutil"
	"infra/cros/internal/osutils"
	"infra/cros/internal/repo"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/common/errors"
	luciflag "go.chromium.org/luci/common/flag"
)

const (
	// Default location of manifest-internal project.
	manifestInternalProjectPath = "manifest-internal"
)

type localManifestBrancher struct {
	subcommands.CommandRunBase
	chromeosCheckoutPath string
	minMilestone         int
	projectList          string
	projects             []string
	push                 bool
}

func cmdLocalManifestBrancher() *subcommands.Command {
	return &subcommands.Command{
		UsageLine: "branch-local-manifest --chromeos_checkout ~/chromiumos " +
			" --min_milestone 90 --projects chromeos/project/foo,chromeos/project/bar",
		ShortDesc: "Repair local_manifest.xml on specified non-ToT branches.",
		CommandRun: func() subcommands.CommandRun {
			b := &localManifestBrancher{}
			b.Flags.StringVar(&b.chromeosCheckoutPath, "chromeos_checkout", "",
				"Path to full ChromeOS checkout.")
			b.Flags.IntVar(&b.minMilestone, "min_milestone", -1,
				"Minimum milestone of branches to consider. Used directly "+
					"in selecting release branches and indirectly for others.")
			b.Flags.Var(luciflag.CommaList(&b.projects), "projects",
				"Comma-separated list of project paths to consider. "+
					"At least one project is required.")
			b.Flags.BoolVar(&b.push, "push", false,
				"Whether or not to push changes to the remote.")
			return b
		}}
}

func (b *localManifestBrancher) validate() error {
	if b.minMilestone == -1 {
		return fmt.Errorf("--min_milestone required")
	}

	if b.chromeosCheckoutPath == "" {
		return fmt.Errorf("--chromeos_checkout required")
	} else if _, err := os.Stat(b.chromeosCheckoutPath); gerrs.Is(err, os.ErrNotExist) {
		return fmt.Errorf("path %s does not exist", b.chromeosCheckoutPath)
	} else if err != nil {
		return fmt.Errorf("error validating --chromeos_checkout=%s", b.chromeosCheckoutPath)
	}

	if len(b.projects) == 0 {
		return fmt.Errorf("at least one project is required")
	}

	return nil
}

func (b *localManifestBrancher) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	// Common setup (argument validation, logging, etc.)
	ret := SetUp(b, a, args, env)
	if ret != 0 {
		return ret
	}

	if err := BranchLocalManifests(b.chromeosCheckoutPath, b.projects, b.minMilestone, !b.push); err != nil {
		LogErr(err.Error())
		return 2
	}

	return 0
}

func pinLocalManifest(checkout, path, branch string, referenceManifest *repo.Manifest, dryRun bool) error {
	// Checkout appropriate branch of project.
	projectPath := filepath.Join(checkout, path)
	if !osutils.PathExists(projectPath) {
		return fmt.Errorf("project path %s does not exist", projectPath)
	}

	if err := git.Checkout(projectPath, branch); err != nil {
		return errors.Annotate(err, "failed to checkout branch %s for project %s", branch, path).Err()
	}

	// Repair local manifest.
	localManifestPath := filepath.Join(projectPath, "local_manifest.xml")
	if _, err := os.Stat(localManifestPath); os.IsNotExist(err) {
		LogOut("local_manifest.xml does not exist for project %s, branch %s, skipping...", path, branch)
		return nil
	}

	localManifest, err := repo.LoadManifestFromFile(localManifestPath)
	if err != nil {
		return errors.Annotate(err, "failed to load local_manifest.xml from project %s, branch %s", path, branch).Err()
	}

	if err := manifestutil.PinManifestFromManifest(&localManifest, referenceManifest); err != nil {
		return errors.Annotate(err, "failed to pin local_manifest.xml from reference manifest for project %s, branch %s", path, branch).Err()
	}
	hasChanges, err := repo.UpdateManifestElementsInFile(localManifestPath, &localManifest)
	if err != nil {
		return errors.Annotate(err, "failed to write changes to local_manifest.xml for project %s, branch %s", path, branch).Err()
	}

	// If the manifest actually changed, commit and push those changes.
	if !hasChanges {
		LogOut("no changes needed for project %s, branch %s\n", path, branch)
		return nil
	}

	commitMsg := fmt.Sprintf("repair local_manifest.xml for branch %s", branch)
	if _, err := git.CommitAll(projectPath, commitMsg); err != nil {
		return errors.Annotate(err, "failed to commit changes for project %s, branch %s", path, branch).Err()
	}

	remotes, err := git.GetRemotes(projectPath)
	if err != nil {
		return errors.Annotate(err, "failed to get remotes for checkout of project %s", path).Err()
	}
	if len(remotes) > 1 {
		return fmt.Errorf("project %s has more than one remote, don't know which to push to", path)
	}
	if len(remotes) == 0 {
		return fmt.Errorf("project %s has no remotes", path)
	}

	remoteRef := git.RemoteRef{
		Remote: remotes[0],
		Ref:    fmt.Sprintf("refs/for/%s", branch) + "%submit",
	}
	if err := git.PushRef(projectPath, "HEAD", remoteRef, git.DryRunIf(dryRun)); err != nil {
		return errors.Annotate(err, "failed to push/upload changes for project %s, branch %s", path, branch).Err()
	}
	if !dryRun {
		LogOut("committed changes for project %s, branch %s\n", path, branch)
	} else {
		LogOut("would have committed changes (dry run) for project %s, branch %s\n", path, branch)
	}

	return nil
}

// BranchLocalManifests is responsible for doing the actual work of local manifest branching.
func BranchLocalManifests(checkout string, projects []string, minMilestone int, dryRun bool) error {
	branches, err := branch.BranchesFromMilestone(checkout, minMilestone)
	if err != nil {
		return errors.Annotate(err, "BranchesFromMilestone failure").Err()
	}

	manifestInternalPath := filepath.Join(checkout, manifestInternalProjectPath)
	if !osutils.PathExists(manifestInternalPath) {
		return fmt.Errorf("manifest-internal checkout not found at %s", manifestInternalPath)
	}

	errs := []error{}
	for _, branch := range branches {
		// Checkout appropriate branch in sentinel project.
		if err := git.Checkout(manifestInternalPath, branch); err != nil {
			errs = append(errs, errors.Annotate(err, "failed to checkout branch %s in %s", branch, manifestInternalProjectPath).Err())
			continue
		}

		// Read reference manifest.
		referencePath := filepath.Join(manifestInternalPath, "default.xml")
		referenceManifest, err := repo.LoadManifestFromFileWithIncludes(referencePath)
		if err != nil {
			errs = append(errs, errors.Annotate(err, "failed to load reference manifest for branch %s", branch).Err())
			continue
		}

		for _, path := range projects {
			if err := pinLocalManifest(checkout, path, branch, referenceManifest, dryRun); err != nil {
				errs = append(errs, err)
			}
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return errors.NewMultiError(errs...)
}
