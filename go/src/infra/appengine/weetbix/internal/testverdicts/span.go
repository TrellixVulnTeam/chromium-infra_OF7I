// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package testverdicts

import (
	"context"
	"time"

	"cloud.google.com/go/spanner"

	"go.chromium.org/luci/server/span"

	spanutil "infra/appengine/weetbix/internal/span"
	pb "infra/appengine/weetbix/proto/v1"
)

// IngestedInvocation represents a row in the IngestedInvocations table.
type IngestedInvocation struct {
	Project                      string
	IngestedInvocationID         string
	SubRealm                     string
	PartitionTime                time.Time
	HasUnsubmittedChanges        bool
	HasContributedToClSubmission bool
}

// ReadIngestedInvocations read ingested invocations from the
// IngestedInvocations table.
// Must be called in a spanner transactional context.
func ReadIngestedInvocations(ctx context.Context, keys spanner.KeySet, fn func(inv *IngestedInvocation) error) error {
	var b spanutil.Buffer
	fields := []string{"Project", "IngestedInvocationId", "SubRealm", "PartitionTime", "HasUnsubmittedChanges", "HasContributedToClSubmission"}
	return span.Read(ctx, "IngestedInvocations", keys, fields).Do(
		func(row *spanner.Row) error {
			inv := &IngestedInvocation{}
			err := b.FromSpanner(
				row,
				&inv.Project,
				&inv.IngestedInvocationID,
				&inv.SubRealm,
				&inv.PartitionTime,
				&inv.HasContributedToClSubmission,
				&inv.HasContributedToClSubmission,
			)
			if err != nil {
				return err
			}
			return fn(inv)
		})
}

// SaveUnverified saves the ingested invocation into the IngestedInvocations
// table without verifying it.
// Must be called in spanner RW transactional context.
func (inv *IngestedInvocation) SaveUnverified(ctx context.Context) {
	row := map[string]interface{}{
		"Project":                      inv.Project,
		"IngestedInvocationId":         inv.IngestedInvocationID,
		"SubRealm":                     inv.SubRealm,
		"PartitionTime":                inv.PartitionTime,
		"HasUnsubmittedChanges":        inv.HasUnsubmittedChanges,
		"HasContributedToClSubmission": inv.HasContributedToClSubmission,
	}
	span.BufferWrite(ctx, spanner.InsertOrUpdateMap("IngestedInvocations", spanutil.ToSpannerMap(row)))
}

// TestVerdict represents a row in the TestVerdicts table.
type TestVerdict struct {
	Project                      string
	TestID                       string
	PartitionTime                time.Time
	VariantHash                  string
	IngestedInvocationID         string
	SubRealm                     string
	ExpectedCount                int64
	UnexpectedCount              int64
	SkippedCount                 int64
	IsExonerated                 bool
	PassedAvgDuration            *time.Duration
	HasUnsubmittedChanges        bool
	HasContributedToClSubmission bool
}

// ReadTestVerdicts read test verdicts from the TestVerdicts table.
// Must be called in a spanner transactional context.
func ReadTestVerdicts(ctx context.Context, keys spanner.KeySet, fn func(tv *TestVerdict) error) error {
	var b spanutil.Buffer
	fields := []string{
		"Project", "TestId", "PartitionTime", "VariantHash", "IngestedInvocationId", "SubRealm", "ExpectedCount",
		"UnexpectedCount", "SkippedCount", "IsExonerated", "PassedAvgDurationUsec", "HasUnsubmittedChanges",
		"HasContributedToClSubmission",
	}
	return span.Read(ctx, "TestVerdicts", keys, fields).Do(
		func(row *spanner.Row) error {
			tv := &TestVerdict{}
			var passedAvgDurationUsec spanner.NullInt64
			err := b.FromSpanner(
				row,
				&tv.Project,
				&tv.TestID,
				&tv.PartitionTime,
				&tv.VariantHash,
				&tv.IngestedInvocationID,
				&tv.SubRealm,
				&tv.ExpectedCount,
				&tv.UnexpectedCount,
				&tv.SkippedCount,
				&tv.IsExonerated,
				&passedAvgDurationUsec,
				&tv.HasUnsubmittedChanges,
				&tv.HasContributedToClSubmission,
			)
			if err != nil {
				return err
			}
			if passedAvgDurationUsec.Valid {
				passedAvgDuration := time.Microsecond * time.Duration(passedAvgDurationUsec.Int64)
				tv.PassedAvgDuration = &passedAvgDuration
			}
			return fn(tv)
		})
}

// SaveUnverified saves the test verdict into the TestVerdicts table without
// verifying it.
// Must be called in spanner RW transactional context.
func (tvr *TestVerdict) SaveUnverified(ctx context.Context) {
	var passedAvgDuration spanner.NullInt64
	if tvr.PassedAvgDuration != nil {
		passedAvgDuration.Int64 = tvr.PassedAvgDuration.Microseconds()
		passedAvgDuration.Valid = true
	}

	row := map[string]interface{}{
		"Project":                      tvr.Project,
		"TestId":                       tvr.TestID,
		"PartitionTime":                tvr.PartitionTime,
		"VariantHash":                  tvr.VariantHash,
		"IngestedInvocationId":         tvr.IngestedInvocationID,
		"SubRealm":                     tvr.SubRealm,
		"ExpectedCount":                tvr.ExpectedCount,
		"UnexpectedCount":              tvr.UnexpectedCount,
		"SkippedCount":                 tvr.SkippedCount,
		"IsExonerated":                 tvr.IsExonerated,
		"PassedAvgDurationUsec":        passedAvgDuration,
		"HasUnsubmittedChanges":        tvr.HasUnsubmittedChanges,
		"HasContributedToClSubmission": tvr.HasContributedToClSubmission,
	}
	span.BufferWrite(ctx, spanner.InsertOrUpdateMap("TestVerdicts", spanutil.ToSpannerMap(row)))
}

// TestVariantRealm represents a row in the TestVariantRealm table.
type TestVariantRealm struct {
	Project           string
	TestID            string
	VariantHash       string
	SubRealm          string
	Variant           *pb.Variant
	LastIngestionTime time.Time
}

// ReadTestVariantRealms read test variant realms from the TestVariantRealms
// table.
// Must be called in a spanner transactional context.
func ReadTestVariantRealms(ctx context.Context, keys spanner.KeySet, fn func(tvr *TestVariantRealm) error) error {
	var b spanutil.Buffer
	fields := []string{"Project", "TestId", "VariantHash", "SubRealm", "Variant", "LastIngestionTime"}
	return span.Read(ctx, "TestVariantRealms", keys, fields).Do(
		func(row *spanner.Row) error {
			tvr := &TestVariantRealm{}
			err := b.FromSpanner(
				row,
				&tvr.Project,
				&tvr.TestID,
				&tvr.VariantHash,
				&tvr.SubRealm,
				&tvr.Variant,
				&tvr.LastIngestionTime,
			)
			if err != nil {
				return err
			}
			return fn(tvr)
		})
}

// SaveUnverified saves the test variant realm into the TestVariantRealms table
// without verifying it.
// Must be called in spanner RW transactional context.
func (tvr *TestVariantRealm) SaveUnverified(ctx context.Context) {
	row := map[string]interface{}{
		"Project":           tvr.Project,
		"TestId":            tvr.TestID,
		"VariantHash":       tvr.VariantHash,
		"SubRealm":          tvr.SubRealm,
		"Variant":           tvr.Variant,
		"LastIngestionTime": tvr.LastIngestionTime,
	}
	span.BufferWrite(ctx, spanner.InsertOrUpdateMap("TestVariantRealms", spanutil.ToSpannerMap(row)))
}
