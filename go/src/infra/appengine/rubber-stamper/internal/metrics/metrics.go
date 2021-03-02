// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package metrics

import (
	"go.chromium.org/luci/common/tsmon/field"
	"go.chromium.org/luci/common/tsmon/metric"
)

var (
	// ReviewerAttemptCount is a counter of attempts of reviewing a CL.
	// We should have:
	// ReviewerAttemptsCount = ReviewerApprovedCount + ReviewerDeclinedCount + (Count of errors returned by func reviewer)
	// review_type can be decided by the getReviewType func in reviewer.go.
	ReviewerAttemptCount = metric.NewCounter(
		"rubber-stamper/reviewer/attempt",
		"Counter of attempts of reviewing a CL",
		nil,
		field.String("host"),
		field.String("repo"),
		field.String("review_type"), // benign_file | cherry_pick | revert
	)

	// ReviewerApprovedCount is a counter of approving a CL.
	ReviewerApprovedCount = metric.NewCounter(
		"rubber-stamper/reviewer/approved",
		"Counter of approving a CL",
		nil,
		field.String("host"),
		field.String("repo"),
		field.String("review_type"),
	)

	// ReviewerDeclinedCount is a counter of declining a CL.
	ReviewerDeclinedCount = metric.NewCounter(
		"rubber-stamper/reviewer/declined",
		"Counter of declining a CL",
		nil,
		field.String("host"),
		field.String("repo"),
		field.String("review_type"),
	)
)
