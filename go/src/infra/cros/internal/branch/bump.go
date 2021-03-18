// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package branch

import (
	"fmt"
	"os"

	mv "infra/cros/internal/chromeosversion"
	"infra/cros/internal/git"

	"go.chromium.org/luci/common/errors"
)

func bumpVersion(
	component mv.VersionComponent,
	br, commitMsg string,
	dryRun bool) error {
	// Branch won't exist if running tool with --dry-run.
	if dryRun {
		return nil
	}
	if component == mv.Unspecified {
		return fmt.Errorf("component was unspecified")
	}

	// Get checkout of versionProjectPath, which has mv.sh.
	opts := &CheckoutOptions{
		Depth: 1,
		Ref:   br,
	}
	versionProjectCheckout, err := GetProjectCheckout(VersionFileProjectPath, opts)
	defer os.RemoveAll(versionProjectCheckout)
	if err != nil {
		return errors.Annotate(err, "bumpVersion: local checkout of version project failed").Err()
	}

	version, err := mv.GetVersionInfoFromRepo(versionProjectCheckout)
	if err != nil {
		return errors.Annotate(err, "failed to read version file").Err()
	}

	version.IncrementVersion(component)
	// We are cloning from a remote, so the remote name will be origin.
	remoteRef := git.RemoteRef{
		Remote: "origin",
		Ref:    git.NormalizeRef(br),
	}

	if err := version.UpdateVersionFile(); err != nil {
		return errors.Annotate(err, "failed to update version file").Err()
	}

	_, err = git.CommitAll(versionProjectCheckout, commitMsg)
	gitOpts := git.Opts{
		DryRun: dryRun,
		Force:  false,
	}
	errs := []error{
		err,
		git.PushRef(versionProjectCheckout, "HEAD", remoteRef, gitOpts),
	}
	for _, err := range errs {
		if err != nil {
			return errors.Annotate(err, "failed to push version changes to remote").Err()
		}
	}
	return nil
}

// BumpForCreate bumps the version in mv.sh, as needed, in the
// source branch for a branch creation command.
func BumpForCreate(componentToBump mv.VersionComponent, release, push bool, branchName, sourceUpstream string) error {
	commitMsg := fmt.Sprintf("Bump %s number after creating branch %s", componentToBump, branchName)
	LogOut(commitMsg)
	if err := bumpVersion(componentToBump, branchName, commitMsg, !push); err != nil {
		return err
	}

	if release {
		// Bump milestone after creating release branch.
		commitMsg = fmt.Sprintf("Bump milestone after creating release branch %s", branchName)
		LogOut(commitMsg)
		if err := bumpVersion(mv.ChromeBranch, sourceUpstream, commitMsg, !push); err != nil {
			return err
		}
		// Also need to bump the build number, otherwise two release will have conflicting versions.
		// See crbug.com/213075.
		commitMsg = fmt.Sprintf("Bump build number after creating release branch %s", branchName)
		LogOut(commitMsg)
		if err := bumpVersion(mv.Build, sourceUpstream, commitMsg, !push); err != nil {
			return err
		}
	} else {
		// For non-release branches, we also have to bump some component of the source branch.
		// This is so that subsequent branches created from the source branch do not conflict
		// with the branch we just created.
		// Example:
		// Say we just branched off of our source branch (version 1.2.0). The newly-created branch
		// has version 1.2.1. If later on somebody tries to branch off of the source branch again,
		// a second branch will be created with version 1.2.0. This is problematic.
		// To avoid this, we bump the source branch. So in this case, we would bump 1.2.0 --> 1.3.0.
		// See crbug.com/965164 for context.
		var sourceComponentToBump mv.VersionComponent
		if componentToBump == mv.Patch {
			sourceComponentToBump = mv.Branch
		} else {
			sourceComponentToBump = mv.Build
		}
		commitMsg = fmt.Sprintf("Bump %s number for source branch %s after creating branch %s",
			sourceComponentToBump, sourceUpstream, branchName)
		LogOut(commitMsg)
		if err := bumpVersion(sourceComponentToBump, sourceUpstream, commitMsg, !push); err != nil {
			return err
		}
	}
	return nil
}
