// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package updater

import (
	"context"

	"infra/appengine/weetbix/internal/analysis"
	"infra/appengine/weetbix/internal/bugs"
	"infra/appengine/weetbix/internal/bugs/monorail"
	"infra/appengine/weetbix/internal/config"

	"go.chromium.org/luci/common/errors"
)

// UpdateBugs updates monorail bugs to reflect the latest analysis.
// Simulate, if true, avoids any changes being applied to monorail and logs
// the changes which would be made instead. This must be set when running
// on developer computers as Weetbix-initiated monorail changes appear
// on monorail as the developer themselves rather than the Weetbix service.
// This leads to bugs errounously being detected as having manual priority
// changes.
func UpdateBugs(ctx context.Context, monorailHost, projectID string, simulate bool) error {
	mc, err := monorail.NewClient(ctx, monorailHost)
	if err != nil {
		return err
	}
	ac, err := analysis.NewClient(ctx, projectID)
	if err != nil {
		return err
	}
	projectCfg, err := config.Projects(ctx)
	if err != nil {
		return err
	}
	var errs []error
	for project, cfg := range projectCfg {
		monorailCfg := cfg.Monorail
		thresholds := cfg.BugFilingThreshold
		mgrs := make(map[string]BugManager)

		mbm := monorail.NewBugManager(mc, monorailCfg)
		mbm.Simulate = simulate
		mgrs[bugs.MonorailSystem] = mbm

		bu := NewBugUpdater(project, mgrs, ac, thresholds)
		if err := bu.Run(ctx); err != nil {
			// Isolate other projects from bug update errors
			// in one project.
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errors.NewMultiError(errs...)
	}
	return nil
}
