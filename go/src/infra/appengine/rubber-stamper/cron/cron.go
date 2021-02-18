// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cron

import (
	"net/http"

	"cloud.google.com/go/errorreporting"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/server/router"

	"infra/appengine/rubber-stamper/config"
	"infra/appengine/rubber-stamper/internal/scheduler"
	"infra/appengine/rubber-stamper/internal/util"
)

// ScheduleReviews add tasks into Cloud Tasks queue, where each task handles
// one CL's review.
func ScheduleReviews(rc *router.Context) {
	ctx, resp := rc.Context, rc.Writer
	if err := scheduler.ScheduleReviews(ctx); err != nil {
		logging.WithError(err).Errorf(ctx, "failed to schedule reviews")
		erc := util.GetErrorReportingClient(ctx)
		if erc != nil {
			erc.Report(errorreporting.Entry{
				Error: err,
			})
		}
		http.Error(resp, err.Error(), 500)
	}
}

// UpdateConfig retrieves updated config from LUCI-config service.
func UpdateConfig(rc *router.Context) {
	ctx, resp := rc.Context, rc.Writer
	if err := config.Update(ctx); err != nil {
		logging.WithError(err).Errorf(ctx, "failed to update config")
		erc := util.GetErrorReportingClient(ctx)
		if erc != nil {
			erc.Report(errorreporting.Entry{
				Error: err,
			})
		}
		http.Error(resp, err.Error(), 500)
	}
}
