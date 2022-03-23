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

// EntryBuilder provides methods to build ingestion control records.
type EntryBuilder struct {
	record *Entry
}

// NewEntry starts building a new Entry.
func NewEntry(uniqifier int) *EntryBuilder {
	return &EntryBuilder{
		record: &Entry{
			BuildID:      fmt.Sprintf("buildbucket-host/%v", uniqifier),
			BuildProject: "build-project",
			BuildResult: &controlpb.BuildResult{
				Host:         "buildbucket-host",
				Id:           int64(uniqifier),
				CreationTime: timestamppb.New(time.Date(2025, time.December, 1, 1, 2, 3, uniqifier*1000, time.UTC)),
			},
			BuildJoinedTime:  time.Date(2020, time.December, 11, 1, 1, 1, uniqifier*1000, time.UTC),
			IsPresubmit:      true,
			PresubmitProject: "presubmit-project",
			PresubmitResult: &controlpb.PresubmitResult{
				PresubmitRunId: &pb.PresubmitRunId{
					System: "luci-cv",
					Id:     fmt.Sprintf("%s/123123-%v", "presubmit-project", uniqifier),
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
			PresubmitJoinedTime: time.Date(2020, time.December, 12, 1, 1, 1, uniqifier*1000, time.UTC),
			LastUpdated:         time.Date(2020, time.December, 13, 1, 1, 1, uniqifier*1000, time.UTC),
		},
	}
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

// WithBuildProject specifies the build project to use on the ingestion control record.
func (b *EntryBuilder) WithBuildProject(project string) *EntryBuilder {
	b.record.BuildProject = project
	return b
}

// WithBuildResult specifies the build result for the entry.
func (b *EntryBuilder) WithBuildResult(value *controlpb.BuildResult) *EntryBuilder {
	b.record.BuildResult = value
	return b
}

// WithBuildJoinedTime specifies the time the build result was populated.
func (b *EntryBuilder) WithBuildJoinedTime(value time.Time) *EntryBuilder {
	b.record.BuildJoinedTime = value
	return b
}

// WithPresubmitProject specifies the presubmit project to use on the ingestion control record.
func (b *EntryBuilder) WithPresubmitProject(project string) *EntryBuilder {
	b.record.PresubmitProject = project
	return b
}

// WithPresubmitResult specifies the build result for the entry.
func (b *EntryBuilder) WithPresubmitResult(value *controlpb.PresubmitResult) *EntryBuilder {
	b.record.PresubmitResult = value
	return b
}

// WithPresubmitJoinedTime specifies the time the presubmit result was populated.
func (b *EntryBuilder) WithPresubmitJoinedTime(lastUpdated time.Time) *EntryBuilder {
	b.record.PresubmitJoinedTime = lastUpdated
	return b
}

// Build constructs the entry.
func (b *EntryBuilder) Build() *Entry {
	return b.record
}

// SetEntriesForTesting replaces the set of stored entries to match the given set.
func SetEntriesForTesting(ctx context.Context, es []*Entry) (time.Time, error) {
	testutil.MustApply(ctx,
		spanner.Delete("Ingestions", spanner.AllKeys()))
	// Insert some Ingestion records.
	commitTime, err := span.ReadWriteTransaction(ctx, func(ctx context.Context) error {
		for _, r := range es {
			ms := spanutil.InsertMap("Ingestions", map[string]interface{}{
				"BuildId":             r.BuildID,
				"BuildProject":        r.BuildProject,
				"BuildResult":         r.BuildResult,
				"BuildJoinedTime":     r.BuildJoinedTime,
				"IsPresubmit":         r.IsPresubmit,
				"PresubmitProject":    r.PresubmitProject,
				"PresubmitResult":     r.PresubmitResult,
				"PresubmitJoinedTime": r.PresubmitJoinedTime,
				"LastUpdated":         r.LastUpdated,
			})
			span.BufferWrite(ctx, ms)
		}
		return nil
	})
	return commitTime.In(time.UTC), err
}
