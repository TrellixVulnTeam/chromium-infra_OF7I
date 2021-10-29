// Copyright 2021 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package plugsupport

import (
	"context"
	"fmt"
	"os"

	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/sync/parallel"

	"infra/tools/migrator"
)

// ExecuteStatusCheck implements "status" subcommand.
func ExecuteStatusCheck(ctx context.Context, projectDir ProjectDir) (*migrator.ReportDump, error) {
	repos, err := discoverAllRepos(ctx, projectDir)
	if err != nil {
		return nil, err
	}

	ctx = InitReportSink(ctx)

	parallel.WorkPool(32, func(ch chan<- func() error) {
		for _, r := range repos {
			r := r
			ch <- func() error {
				executeStatusCheck(ctx, r)
				return nil
			}
		}
	})

	dump := DumpReports(ctx)

	// Write the reports out as CSV.
	scanOut, err := os.Create(projectDir.StatusReportPath())
	if err != nil {
		return nil, err
	}
	defer scanOut.Close()
	if err := dump.WriteToCSV(scanOut); err != nil {
		return nil, err
	}

	return dump, nil
}

// executeStatusCheck checks status of a single repo.
func executeStatusCheck(ctx context.Context, r *repo) {
	ctx = logging.SetField(ctx, "checkout", r.checkoutID)
	git := r.git(ctx)

	report := func(tag, description string, opts ...migrator.ReportOption) {
		getReportSink(ctx).add(r.reportID(), tag, description, opts...)
	}

	// Check if we have any changes in the index or staging area.
	if uncommittedDiff := git.read("diff", "HEAD", "--name-only"); uncommittedDiff != "" {
		report("UNCOMMITTED", "Has uncommitted changes")
	}

	// Check if we have any committed changes already (compared to the upstream).
	if count := git.read("rev-list", "--count", "@{u}..HEAD"); count != "0" {
		// Look if we have a CL associated with it already.
		var cl string
		if host := git.read("config", fmt.Sprintf("branch.%s.gerritserver", localBranch)); host != "" {
			issue := git.read("config", fmt.Sprintf("branch.%s.gerritissue", localBranch))
			if issue != "" {
				cl = fmt.Sprintf("%s/c/%s", host, issue)
			}
		}
		if cl == "" {
			report("COMMITTED", "Pending commit")
		} else {
			report("UPLOADED", "Uploaded CL", migrator.MetadataOption("CL", cl))
		}
	}

	if git.err != nil {
		logging.Errorf(ctx, "%s", git.err)
		report("GIT_ERROR", git.err.Error())
	}
}
