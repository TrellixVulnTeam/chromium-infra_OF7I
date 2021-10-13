// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package verdicts

import (
	"context"
	"fmt"

	"infra/appengine/weetbix/internal/tasks/taskspb"
	pb "infra/appengine/weetbix/proto/v1"
)

// ComputeTestVariantStatusFromVerdicts computes the test variant's status based
// on its verdicts within a time range.
func ComputeTestVariantStatusFromVerdicts(ctx context.Context, tvKey *taskspb.TestVariantKey) (pb.AnalyzedTestVariantStatus, error) {
	return pb.AnalyzedTestVariantStatus_STATUS_UNSPECIFIED, fmt.Errorf("not implemented")
}
