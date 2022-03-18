// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package handlers

import (
	"context"

	"go.chromium.org/luci/common/errors"

	"infra/appengine/weetbix/internal/bugs/updater"
	"infra/appengine/weetbix/internal/config"
)

// UpdateAnalysisAndBugs handles the update-analysis-and-bugs cron job.
func (h *Handlers) UpdateAnalysisAndBugs(ctx context.Context) error {
	cfg, err := config.Get(ctx)
	if err != nil {
		return errors.Annotate(err, "get config").Err()
	}
	simulate := !h.prod
	enabled := cfg.BugUpdatesEnabled
	err = updater.UpdateAnalysisAndBugs(ctx, cfg.MonorailHostname, h.cloudProject, simulate, enabled)
	if err != nil {
		return errors.Annotate(err, "update bugs").Err()
	}
	return nil
}
