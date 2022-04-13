// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package testverdicts

import (
	"time"
)

// RuleBuilder provides methods to build a test verdict for testing.
type TestVerdictBuilder struct {
	verdict TestVerdict
}

func NewTestVerdict() *TestVerdictBuilder {
	verdict := TestVerdict{
		Project:              "proj",
		TestID:               "test_id",
		VariantHash:          "hash",
		IngestedInvocationID: "inv-id",
		SubRealm:             "realm",
	}
	return &TestVerdictBuilder{
		verdict: verdict,
	}
}

func (b *TestVerdictBuilder) WithProject(project string) *TestVerdictBuilder {
	b.verdict.Project = project
	return b
}

func (b *TestVerdictBuilder) WithTestID(testID string) *TestVerdictBuilder {
	b.verdict.TestID = testID
	return b
}

func (b *TestVerdictBuilder) WithPartitionTime(partitionTime time.Time) *TestVerdictBuilder {
	b.verdict.PartitionTime = partitionTime
	return b
}

func (b *TestVerdictBuilder) WithVariantHash(variantHash string) *TestVerdictBuilder {
	b.verdict.VariantHash = variantHash
	return b
}

func (b *TestVerdictBuilder) WithIngestedInvocationID(invID string) *TestVerdictBuilder {
	b.verdict.IngestedInvocationID = invID
	return b
}

func (b *TestVerdictBuilder) WithSubRealm(subRealm string) *TestVerdictBuilder {
	b.verdict.SubRealm = subRealm
	return b
}

func (b *TestVerdictBuilder) WithExpectedCount(count int64) *TestVerdictBuilder {
	b.verdict.ExpectedCount = count
	return b
}

func (b *TestVerdictBuilder) WithUnexpectedCount(count int64) *TestVerdictBuilder {
	b.verdict.UnexpectedCount = count
	return b
}

func (b *TestVerdictBuilder) WithSkippedCount(count int64) *TestVerdictBuilder {
	b.verdict.SkippedCount = count
	return b
}

func (b *TestVerdictBuilder) WithIsExonerated(isExonerated bool) *TestVerdictBuilder {
	b.verdict.IsExonerated = isExonerated
	return b
}

func (b *TestVerdictBuilder) WithPassedAvgDuration(duration time.Duration) *TestVerdictBuilder {
	b.verdict.PassedAvgDuration = &duration
	return b
}

func (b *TestVerdictBuilder) WithoutPassedAvgDuration() *TestVerdictBuilder {
	b.verdict.PassedAvgDuration = nil
	return b
}

func (b *TestVerdictBuilder) WithHasUnsubmittedChanges(hasUnsubmittedChanges bool) *TestVerdictBuilder {
	b.verdict.HasUnsubmittedChanges = hasUnsubmittedChanges
	return b
}

func (b *TestVerdictBuilder) WithHasContributedToClSubmission(hasContributedToClSubmission bool) *TestVerdictBuilder {
	b.verdict.HasContributedToClSubmission = hasContributedToClSubmission
	return b
}

func (b *TestVerdictBuilder) Build() *TestVerdict {
	// Copy the result, so that calling further methods on the builder does
	// not change the returned test verdict.
	result := new(TestVerdict)
	*result = b.verdict
	return result
}
