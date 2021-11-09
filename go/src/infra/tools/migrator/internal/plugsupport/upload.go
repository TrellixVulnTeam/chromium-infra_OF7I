// Copyright 2021 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package plugsupport

import (
	"context"
	"io/ioutil"

	"infra/tools/migrator"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
)

// UploadOptions are passed to ExecuteUpload.
type UploadOptions struct {
	Reviewers string // passed as --reviewers to "git cl upload"
	CC        string // passed as --cc to "git cl upload"
	Force     bool   // if true, call "git cl upload" even if nothing changed
}

// ExecuteUpload implements "upload" subcommand.
func ExecuteUpload(ctx context.Context, projectDir ProjectDir, opts UploadOptions) (*migrator.ReportDump, error) {
	blob, err := ioutil.ReadFile(projectDir.CommitMessageFile())
	if err != nil {
		return nil, errors.Annotate(err, "failed to read the commit message").Err()
	}
	message := string(blob)

	return visitReposInParallel(ctx, projectDir, projectDir.UploadReportPath(), func(ctx context.Context, r *repo) {
		git := r.git(ctx)

		// Check if we have any changes in the index or staging area.
		if !opts.Force {
			if uncommittedDiff := git.read("diff", "HEAD", "--name-only"); uncommittedDiff == "" {
				var clMD []migrator.ReportOption
				if cl := git.gerritCL(); cl != "" {
					clMD = append(clMD, migrator.MetadataOption("CL", cl))
				}
				r.report(ctx, "UNCHANGED", "No new changes", clMD...)
				return
			}
		}

		// Prepare the local commit or amend the existing one (if any).
		commitCmd := []string{
			"commit", "--quiet", "--all", "--no-edit", "--message", message,
		}
		amend := git.read("rev-list", "--count", "@{u}..HEAD") != "0"
		if amend {
			commitCmd = append(commitCmd, "--amend")
		}
		git.run(commitCmd...)

		// Upload it as a CL.
		uploadArgs := []string{
			"cl", "upload", "--force", "--bypass-hooks", "--message", message,
		}
		if opts.Reviewers != "" {
			uploadArgs = append(uploadArgs, "--reviewers", opts.Reviewers)
		} else {
			uploadArgs = append(uploadArgs, "--r-owners")
		}
		if opts.CC != "" {
			uploadArgs = append(uploadArgs, "--cc", opts.CC)
		}
		git.run(uploadArgs...)

		// We should have a CL link now.
		clMD := migrator.MetadataOption("CL", git.gerritCL())

		if amend {
			r.report(ctx, "UPDATED", "Updated the CL", clMD)
		} else {
			r.report(ctx, "UPLOADED", "Created the CL", clMD)
		}

		if git.err != nil {
			logging.Errorf(ctx, "%s", git.err)
			r.report(ctx, "GIT_ERROR", git.err.Error())
		}
	})
}
