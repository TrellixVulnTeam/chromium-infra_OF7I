// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package control

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/spanner"
	"go.chromium.org/luci/server/span"
	"google.golang.org/protobuf/types/known/timestamppb"

	controlpb "infra/appengine/weetbix/internal/ingestion/control/proto"
	spanutil "infra/appengine/weetbix/internal/span"
	"infra/appengine/weetbix/internal/testutil"
	pb "infra/appengine/weetbix/proto/v1"
)

const testProject = "myproject"

// EntryBuilder provides methods to build ingestion control records.
type EntryBuilder struct {
	record *Entry
}

// NewEntry starts building a new Entry.
func NewEntry(uniqifier int) *EntryBuilder {
	return &EntryBuilder{
		record: &Entry{
			Project: testProject,
			BuildID: fmt.Sprintf("buildbucket-host/%v", uniqifier),
			BuildResult: &controlpb.BuildResult{
				Host:         "buildbucket-host",
				Id:           int64(uniqifier),
				CreationTime: timestamppb.New(time.Date(2025, time.December, 1, 1, 2, 3, uniqifier*1000, time.UTC)),
			},
			IsPresubmit: true,
			PresubmitResult: &controlpb.PresubmitResult{
				PresubmitRunId: &pb.PresubmitRunId{
					System: "luci-cv",
					Id:     fmt.Sprintf("%s/123123-%v", testProject, uniqifier),
				},
				PresubmitRunSucceeded: true,
				Owner:                 "automation",
				Cls: []*pb.Changelist{
					{
						Host:     "chromium-review.googlesource.com",
						Change:   12345,
						Patchset: 1,
					},
				},
				CreationTime: timestamppb.New(time.Date(2026, time.December, 1, 1, 2, 3, uniqifier*1000, time.UTC)),
			},
			LastUpdated: time.Date(2020, time.December, 12, 1, 1, 1, 0, time.UTC),
		},
	}
}

// WithProject specifies the project to use on the ingestion control record.
func (b *EntryBuilder) WithProject(project string) *EntryBuilder {
	b.record.Project = project
	return b
}

// WithBuildID specifies the build ID to use on the ingestion control record.
func (b *EntryBuilder) WithBuildID(id string) *EntryBuilder {
	b.record.BuildID = id
	return b
}

// WithIsPresubmit specifies whether the ingestion relates to a presubmit run.
func (b *EntryBuilder) WithIsPresubmit(isPresubmit bool) *EntryBuilder {
	b.record.IsPresubmit = isPresubmit
	return b
}

// WithBuildResult specifies the build result for the entry.
func (b *EntryBuilder) WithBuildResult(value *controlpb.BuildResult) *EntryBuilder {
	b.record.BuildResult = value
	return b
}

// WithPresubmitResult specifies the build result for the entry.
func (b *EntryBuilder) WithPresubmitResult(value *controlpb.PresubmitResult) *EntryBuilder {
	b.record.PresubmitResult = value
	return b
}

// WithLastUpdated specifies the last updated time for the entry.
func (b *EntryBuilder) WithLastUpdated(lastUpdated time.Time) *EntryBuilder {
	b.record.LastUpdated = lastUpdated
	return b
}

// Build constructs the entry.
func (b *EntryBuilder) Build() *Entry {
	return b.record
}

// SetEntriesForTesting replaces the set of stored entries to match the given set.
func SetEntriesForTesting(ctx context.Context, es []*Entry) (time.Time, error) {
	testutil.MustApply(ctx,
		spanner.Delete("IngestionControl", spanner.AllKeys()))
	// Insert some IngestionControl records.
	commitTime, err := span.ReadWriteTransaction(ctx, func(ctx context.Context) error {
		for _, r := range es {
			ms := spanutil.InsertMap("IngestionControl", map[string]interface{}{
				"Project":         r.Project,
				"BuildId":         r.BuildID,
				"BuildResult":     r.BuildResult,
				"IsPresubmit":     r.IsPresubmit,
				"PresubmitResult": r.PresubmitResult,
				"LastUpdated":     r.LastUpdated,
			})
			span.BufferWrite(ctx, ms)
		}
		return nil
	})
	return commitTime.In(time.UTC), err
}
