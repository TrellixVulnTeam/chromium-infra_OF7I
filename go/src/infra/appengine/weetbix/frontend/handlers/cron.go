// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package handlers

import (
	"context"

	"go.chromium.org/luci/common/errors"

	// Store auth sessions in the datastore.
	_ "go.chromium.org/luci/server/encryptedcookies/session/datastore"

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
	err = updater.UpdateAnalysisAndBugs(ctx, cfg.MonorailHostname, h.cloudProject, simulate)
	if err != nil {
		return errors.Annotate(err, "update bugs").Err()
	}
	return nil
}
