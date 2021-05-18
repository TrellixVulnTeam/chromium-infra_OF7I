// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package main

import (
	"fmt"
	"path/filepath"

	"infra/cros/internal/branch"
	"infra/cros/internal/git"
	"infra/cros/internal/manifestutil"
	"infra/cros/internal/osutils"
	"infra/cros/internal/repo"

	"go.chromium.org/luci/common/errors"
)

const (
	// Default location of manifest-internal project.
	manifestInternalProject = "manifest-internal"
)

func pinLocalManifest(checkout, path, branch string, referenceManifest *repo.Manifest) error {
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

		fmt.Printf("no changes needed for project %s, branch %s\n", path, branch)
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
	if err := git.PushRef(projectPath, "HEAD", remoteRef); err != nil {
		return errors.Annotate(err, "failed to push/upload changes for project %s, branch %s", path, branch).Err()
	}
	fmt.Printf("committed changes for project %s, branch %s\n", path, branch)

	return nil
}

// BranchLocalManifests is responsible for doing the actual work of local manifest branching.
func BranchLocalManifests(checkout string, projects []string, minMilestone int) error {
	branches, err := branch.BranchesFromMilestone(checkout, minMilestone)
	if err != nil {
		return errors.Annotate(err, "BranchesFromMilestone failure").Err()
	}

	manifestInternalPath := filepath.Join(checkout, manifestInternalProject)
	if !osutils.PathExists(manifestInternalPath) {
		return fmt.Errorf("manifest-internal checkout not found at %s", manifestInternalPath)
	}

	errs := []error{}
	for _, branch := range branches {
		// Checkout appropriate branch in sentinel project.
		if err := git.Checkout(manifestInternalPath, branch); err != nil {
			errs = append(errs, errors.Annotate(err, "failed to checkout branch %s in %s", branch, manifestInternalProject).Err())
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
			if err := pinLocalManifest(checkout, path, branch, referenceManifest); err != nil {
				errs = append(errs, err)
			}
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return errors.NewMultiError(errs...)
}
