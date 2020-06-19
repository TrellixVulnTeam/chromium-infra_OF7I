// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package transform contains tools for transforming CTP build
// to test_platform/analytics/TestPlanRun proto.
package transform

import (
	"fmt"

	"go.chromium.org/chromiumos/infra/proto/go/test_platform/result_flow"
)

func inferExecutionURL(c *result_flow.BuildbucketConfig, bID int64) string {
	return fmt.Sprintf(
		"https://ci.chromium.org/p/%s/builders/%s/%s/b%d",
		c.GetProject(),
		c.GetBucket(),
		c.GetBuilder(),
		bID,
	)
}
