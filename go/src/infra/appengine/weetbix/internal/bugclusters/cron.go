// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package bugclusters

import (
	"context"
	"infra/appengine/weetbix/internal/bugs/monorail"
	"infra/appengine/weetbix/internal/clustering"
	"infra/appengine/weetbix/internal/config"
)

// UpdateBugs updates monorail bugs to reflect the latest analysis.
// Simulate, if true, avoids any changes being applied to monorail and logs
// the changes which would be made instead. This must be set when running
// on developer computers as developer-initiated monorail changes appear
// on monorail as the developer themselves rather than the Weetbix service.
// This leads to bugs errounously being detected as having manual priority
// changes.
func UpdateBugs(ctx context.Context, monorailHost, projectID string, thresholds clustering.ImpactThresholds, simulate bool) error {
	mc, err := monorail.NewClient(ctx, monorailHost)
	if err != nil {
		return err
	}
	cc, err := clustering.NewClient(ctx, projectID)
	if err != nil {
		return err
	}
	projectCfg, err := config.Projects(ctx)
	if err != nil {
		return err
	}
	monorailCfg := make(map[string]*config.MonorailProject)
	for project, cfg := range projectCfg {
		monorailCfg[project] = cfg.Monorail
	}
	mgrs := make(map[string]BugManager)
	mbm := monorail.NewBugManager(mc, monorailCfg)
	mbm.Simulate = simulate
	mgrs[monorail.ManagerName] = mbm

	bu := NewBugUpdater(mgrs, cc, thresholds)
	if err := bu.Run(ctx); err != nil {
		return err
	}
	return nil
}
