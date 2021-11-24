// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package runs

import (
	"context"
	"errors"
	"time"

	"go.chromium.org/luci/server/span"
)

// ProgressToken is used to report the progress reported for a shard. Only
// one progress token should ever be created in the lifetime of a shard,
// to avoid double-reporting progress.
type ProgressToken struct {
	project              string
	attemptTimestamp     time.Time
	reportedOnce         bool
	lastReportedProgress int
	invalid              bool
}

// NewProgressToken initialises a new progress token with the given LUCI
// project ID and attempt key.
func NewProgressToken(project string, attemptTimestamp time.Time) *ProgressToken {
	return &ProgressToken{
		project:          project,
		attemptTimestamp: attemptTimestamp,
	}
}

// ReportProgress reports the progress for the current shard. Progress ranges
// from 0 to 1000, with 1000 indicating the all work assigned to the shard
// is complete.
func (p *ProgressToken) ReportProgress(ctx context.Context, value int) error {
	if p.invalid {
		return errors.New("no more progress can be reported; token is invalid")
	}
	// Bound progress values to the allowed range.
	if value < 0 || value > 1000 {
		return errors.New("progress value must be between 0 and 1000")
	}
	if p.reportedOnce && p.lastReportedProgress == value {
		// Progress did not change, nothing to do.
		return nil
	}
	_, err := span.ReadWriteTransaction(ctx, func(ctx context.Context) error {
		deltaProgress := value - p.lastReportedProgress
		err := reportProgress(ctx, p.project, p.attemptTimestamp, !p.reportedOnce, deltaProgress)
		return err
	})
	if err != nil {
		// If we get an error back, we are not sure if the transaction
		// failed to commit, or if it did commit but our connection to
		// Spanner dropped. We should treat the token as invalid and
		// not report any more progress for this shard.
		p.invalid = true
		return err
	}
	p.reportedOnce = true
	p.lastReportedProgress = value
	return nil
}
