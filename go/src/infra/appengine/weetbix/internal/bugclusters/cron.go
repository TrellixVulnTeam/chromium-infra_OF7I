// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package bugclusters

import (
	"context"
	"infra/appengine/weetbix/internal/bugs/monorail"
	"infra/appengine/weetbix/internal/clustering"
)

// UpdateBugs updates monorail bugs to reflect the latest analysis.
func UpdateBugs(ctx context.Context, monorailHost, projectID string, thresholds clustering.ImpactThresholds) error {
	mc, err := monorail.NewClient(ctx, monorailHost)
	if err != nil {
		return err
	}
	cc, err := clustering.NewClient(ctx, projectID)
	if err != nil {
		return err
	}
	mgrs := make(map[string]BugManager)
	mgrs[monorail.ManagerName] = monorail.NewBugManager(mc)

	bu := NewBugUpdater(mgrs, cc, thresholds)
	if err := bu.Run(ctx); err != nil {
		return err
	}
	return nil
}
