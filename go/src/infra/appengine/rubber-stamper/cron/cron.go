// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cron

import (
	"net/http"

	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/server/router"

	"infra/appengine/rubber-stamper/config"
)

// ScheduleReviews add tasks into Cloud Tasks queue, where each task handles
// one CL's review.
func ScheduleReviews(rc *router.Context) {
}

// UpdateConfig retrieves updated config from LUCI-config service.
func UpdateConfig(rc *router.Context) {
	ctx, resp := rc.Context, rc.Writer
	if err := config.Update(ctx); err != nil {
		logging.WithError(err).Errorf(ctx, "Failed to update config")
		http.Error(resp, err.Error(), 500)
	}
}
