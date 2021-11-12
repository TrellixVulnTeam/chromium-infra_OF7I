// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package runs

import (
	"context"
	"time"

	"cloud.google.com/go/spanner"

	"infra/appengine/weetbix/internal/testutil"

	"go.chromium.org/luci/server/span"
)

const testProject = "myproject"

// RunBuilder provides methods to build a reclustering run
// for testing.
type RunBuilder struct {
	run *ReclusteringRun
}

// NewRun starts building a new Run, for testing.
func NewRun(uniqifier int) *RunBuilder {
	run := &ReclusteringRun{
		Project:           testProject,
		AttemptTimestamp:  time.Date(2010, time.January, 1, 1, 0, 0, uniqifier, time.UTC),
		AlgorithmsVersion: int64(uniqifier + 1),
		RulesVersion:      time.Date(2012, time.January, 1, 1, 0, 0, uniqifier, time.UTC),
		ShardCount:        int64(uniqifier + 1),
		ShardsReported:    int64(uniqifier / 2),
		Progress:          int64(uniqifier) * 500,
	}
	return &RunBuilder{run}
}

// WithProject specifies the project to use on the run.
func (b *RunBuilder) WithProject(project string) *RunBuilder {
	b.run.Project = project
	return b
}

// WithAttemptTimestamp specifies the attempt timestamp to use on the run.
func (b *RunBuilder) WithAttemptTimestamp(attemptTimestamp time.Time) *RunBuilder {
	b.run.AttemptTimestamp = attemptTimestamp
	return b
}

func (b *RunBuilder) WithReportedProgress() *RunBuilder {
	b.run.ShardsReported = b.run.ShardCount
	return b
}

func (b *RunBuilder) WithCompletedProgress() *RunBuilder {
	b.run.ShardsReported = b.run.ShardCount
	b.run.Progress = b.run.ShardCount * 1000
	return b
}

func (b *RunBuilder) Build() *ReclusteringRun {
	return b.run
}

// SetRunsForTesting replaces the set of stored runs to match the given set.
func SetRunsForTesting(ctx context.Context, rs []*ReclusteringRun) error {
	testutil.MustApply(ctx,
		spanner.Delete("ReclusteringRuns", spanner.AllKeys()))
	// Insert some ReclusteringRuns.
	_, err := span.ReadWriteTransaction(ctx, func(ctx context.Context) error {
		for _, r := range rs {
			if err := Create(ctx, r); err != nil {
				return err
			}
		}
		return nil
	})
	return err
}
