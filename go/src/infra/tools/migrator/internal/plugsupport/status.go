// Copyright 2021 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package plugsupport

import (
	"context"

	"go.chromium.org/luci/common/logging"

	"infra/tools/migrator"
)

// ExecuteStatusCheck implements "status" subcommand.
func ExecuteStatusCheck(ctx context.Context, projectDir ProjectDir) (*migrator.ReportDump, error) {
	return visitReposInParallel(ctx, projectDir, projectDir.StatusReportPath(), func(ctx context.Context, r *repo) {
		git := r.git(ctx)

		// Check if we have any changes in the index or staging area.
		if uncommittedDiff := git.read("diff", "HEAD", "--name-only"); uncommittedDiff != "" {
			r.report(ctx, "UNCOMMITTED", "Has uncommitted changes")
		}

		// Check if we have any committed changes already (compared to the upstream).
		if count := git.read("rev-list", "--count", "@{u}..HEAD"); count != "0" {
			// Look if we have a CL associated with it already.
			if cl := git.gerritCL(); cl != "" {
				r.report(ctx, "UPLOADED", "Uploaded CL", migrator.MetadataOption("CL", cl))
			} else {
				r.report(ctx, "COMMITTED", "Pending commit")
			}
		}

		if git.err != nil {
			logging.Errorf(ctx, "%s", git.err)
			r.report(ctx, "GIT_ERROR", git.err.Error())
		}
	})
}
