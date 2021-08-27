// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package resultingester

import (
	"context"

	"go.chromium.org/luci/common/errors"

	"infra/appengine/weetbix/internal/tasks/taskspb"
)

func ingestTestResults(ctx context.Context, payload *taskspb.IngestTestResults) error {
	return errors.Reason("Not Implemented").Err()
}
