// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cron

import (
	"go.chromium.org/luci/server/router"
)

// ScheduleReviews add tasks into Cloud Tasks queue, where each task handles
// one CL's review.
func ScheduleReviews(rc *router.Context) {
}
