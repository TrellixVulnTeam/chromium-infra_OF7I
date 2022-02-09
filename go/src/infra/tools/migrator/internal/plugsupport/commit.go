// Copyright 2021 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package plugsupport

import (
	"context"
	"io/ioutil"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"

	"infra/tools/migrator"
)

// ExecuteCommit implements "commit" subcommand.
func ExecuteCommit(ctx context.Context, projectDir ProjectDir) (*migrator.ReportDump, error) {
	blob, err := ioutil.ReadFile(projectDir.CommitMessageFile())
	if err != nil {
		return nil, errors.Annotate(err, "failed to read the commit message").Err()
	}
	message := string(blob)

	return visitReposInParallel(ctx, projectDir, projectDir.CommitReportPath(), func(ctx context.Context, r *repo) {
		git := r.git(ctx)
		defer func() {
			if git.err != nil {
				logging.Errorf(ctx, "%s", git.err)
				r.report(ctx, "GIT_ERROR", git.err.Error())
			}
		}()

		// Skip completely untouched checkouts.
		uncommittedDiff := git.read("diff", "HEAD", "--name-only")
		localCommit := git.read("rev-list", "--count", "@{u}..HEAD") != "0"
		if uncommittedDiff == "" && !localCommit {
			return
		}

		// Prepare the local commit or amend the existing one (if any).
		commitCmd := []string{
			"commit", "--quiet", "--all", "--no-edit", "--message", message,
		}
		if localCommit {
			commitCmd = append(commitCmd, "--amend")
		}
		git.run(commitCmd...)
		if git.err != nil {
			return
		}

		if localCommit {
			r.report(ctx, "AMENDED", "Amended the local commit")
		} else {
			r.report(ctx, "COMMITTED", "Created the local commit")
		}
	})
}
