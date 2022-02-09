// Copyright 2022 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package plugsupport

import (
	"context"
	"fmt"

	"infra/tools/migrator"
)

// ExecuteRebase implements "rebase" subcommand.
func ExecuteRebase(ctx context.Context, projectDir ProjectDir) (*migrator.ReportDump, error) {
	return visitReposInParallel(ctx, projectDir, projectDir.RebaseReportPath(), func(ctx context.Context, r *repo) {
		git := r.git(ctx)

		// Make sure we are on the expected local branch.
		branch := git.read("rev-parse", "--abbrev-ref", "HEAD")
		if branch != localBranch {
			r.report(ctx, "SKIPPED", fmt.Sprintf("Must be on branch %q", branch))
			return
		}

		// Checkout should be clean.
		uncommittedDiff := git.read("diff", "HEAD", "--name-only")
		if uncommittedDiff != "" {
			r.report(ctx, "UNCLEAN", "The checkout has uncommitted changes")
			return
		}

		if err := r.fetch(ctx); err != nil {
			r.report(ctx, "GIT_FETCH_ERROR", err.Error())
			return
		}

		before := git.read("rev-parse", "HEAD")
		git.run("rebase")
		if git.err != nil {
			r.report(ctx, "REBASE_ERROR", git.err.Error())
			return
		}
		after := git.read("rev-parse", "HEAD")

		if before != after {
			r.report(ctx, "REBASED", fmt.Sprintf("%s => %s", before[:6], after[:6]))
		} else {
			r.report(ctx, "UNCHANGED", "No changes")
		}
	})
}
