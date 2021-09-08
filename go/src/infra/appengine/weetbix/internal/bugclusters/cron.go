// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package bugclusters

import (
	"context"
	"infra/appengine/weetbix/internal/bugs"
	"infra/appengine/weetbix/internal/clustering"
)

// UpdateBugs updates monorail bugs to reflect the latest analysis.
func UpdateBugs(ctx context.Context, monorailHost, projectID, reporter string, thresholds clustering.ImpactThresholds) error {
	mc, err := bugs.NewMonorailClient(ctx, monorailHost)
	if err != nil {
		return err
	}
	cc, err := clustering.NewClient(ctx, projectID)
	if err != nil {
		return err
	}
	ig := bugs.NewIssueGenerator(reporter)
	bu := NewBugUpdater(mc, cc, ig, thresholds)
	if err := bu.Run(ctx); err != nil {
		return err
	}
	return nil
}
