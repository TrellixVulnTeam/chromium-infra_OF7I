// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package resultingester

import (
	"context"
	"fmt"
	"sort"
	"time"

	"cloud.google.com/go/spanner"
	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/logging"
	rdbpb "go.chromium.org/luci/resultdb/proto/v1"
	"go.chromium.org/luci/server/span"
	"google.golang.org/protobuf/proto"

	"infra/appengine/weetbix/internal/analyzedtestvariants"
	"infra/appengine/weetbix/internal/config"
	configpb "infra/appengine/weetbix/internal/config/proto"
	"infra/appengine/weetbix/internal/services/testvariantupdator"
	spanutil "infra/appengine/weetbix/internal/span"
	"infra/appengine/weetbix/internal/tasks/taskspb"
	"infra/appengine/weetbix/pbutil"
	pb "infra/appengine/weetbix/proto/v1"
)

type testVariantKey struct {
	TestId      string
	VariantHash string
}

// tagKeys are the keys for tags that should be saved with analyzed test
// variants.
var tagKeys = map[string]struct{}{
	"monorail_component": {},
	"os":                 {},
	"team_email":         {},
	"test_name":          {},
	"target_platform":    {},
}

// shouldIngestForTestVariants returns whether the test results specified
// by the given IngestTestResults task, with a realm with the given RealmConfig.
func shouldIngestForTestVariants(realmcfg *configpb.RealmConfig, task *taskspb.IngestTestResults) bool {
	if realmcfg.GetTestVariantAnalysis().GetUpdateTestVariantTask().GetUpdateTestVariantTaskInterval() == nil {
		// Test Variant analysis not configured for realm. Skip ingestion.
		return false
	}
	// Ingest results from CI.
	return task.PresubmitRun == nil ||
		// And presubmit results, where the presubmit run succeeded
		// and the run was a FULL_RUN.
		(task.PresubmitRun != nil && task.PresubmitRun.PresubmitRunSucceeded &&
			task.PresubmitRun.Mode == "FULL_RUN")
}

// createOrUpdateAnalyzedTestVariants looks for new analyzed test variants or
// the ones to be updated, and save them in Spanner.
func createOrUpdateAnalyzedTestVariants(ctx context.Context, realm, builder string, tvs []*rdbpb.TestVariant) error {
	if len(tvs) == 0 {
		return nil
	}

	rc, err := config.Realm(ctx, realm)
	switch {
	case err != nil:
		return err
	case rc.GetTestVariantAnalysis().GetUpdateTestVariantTask().GetUpdateTestVariantTaskInterval() == nil:
		// This should never occur, as shouldIngestForTestVariants() should be called
		// before this method.
		return fmt.Errorf("no UpdateTestVariantTask config found for realm %s", realm)
	}

	ks := testVariantKeySet(realm, tvs)
	_, err = span.ReadWriteTransaction(ctx, func(ctx context.Context) error {
		found := make(map[testVariantKey]*pb.AnalyzedTestVariant)
		err := analyzedtestvariants.ReadStatusAndTags(ctx, ks, func(atv *pb.AnalyzedTestVariant) error {
			k := testVariantKey{atv.TestId, atv.VariantHash}
			found[k] = atv
			return nil
		})
		if err != nil {
			return err
		}

		ms := make([]*spanner.Mutation, 0)
		// A map of test variants to the enqueue time of their first UpdateTestVariant
		// task.
		tvToEnQTime := make(map[testVariantKey]time.Time)
		for _, tv := range tvs {
			tvStr := fmt.Sprintf("%s-%s-%s", realm, tv.TestId, tv.VariantHash)
			if shouldSkipTestVariant(tv) {
				continue
			}

			k := testVariantKey{tv.TestId, tv.VariantHash}
			atv, ok := found[k]
			if !ok {
				m, enqueueTime, err := insertRow(ctx, realm, builder, tv)
				if err != nil {
					logging.Errorf(ctx, "Insert test variant %s: %s", tvStr, err)
					continue
				}
				ms = append(ms, m)
				tvToEnQTime[k] = enqueueTime
			} else {
				nts := updatedTags(extractRequiredTags(tv), atv.Tags)
				ds, err := derivedStatus(tv.Status)
				if err != nil {
					logging.Errorf(ctx, "Update test variant %s: %s", tvStr, err)
					continue
				}
				ns, err := updatedStatus(ds, atv.Status)
				if err != nil {
					logging.Errorf(ctx, "Update test variant %s: %s", tvStr, err)
					continue
				}
				if ns == atv.Status && len(nts) == 0 {
					continue
				}

				vals := map[string]interface{}{
					"Realm":       atv.Realm,
					"TestId":      atv.TestId,
					"VariantHash": atv.VariantHash,
				}
				if len(nts) > 0 {
					vals["Tags"] = nts
				}
				if ns != atv.Status {
					vals["Status"] = int64(ns)
					if atv.Status == pb.AnalyzedTestVariantStatus_CONSISTENTLY_EXPECTED || atv.Status == pb.AnalyzedTestVariantStatus_NO_NEW_RESULTS {
						// The test variant starts to have unexpected failures again, need
						// to start updating its status.
						now := clock.Now(ctx)
						vals["NextUpdateTaskEnqueueTime"] = now
						tvToEnQTime[k] = now
					}
				}
				ms = append(ms, spanutil.UpdateMap("AnalyzedTestVariants", vals))
			}
		}
		span.BufferWrite(ctx, ms...)
		for tvKey, enQTime := range tvToEnQTime {
			testvariantupdator.Schedule(ctx, realm, tvKey.TestId, tvKey.VariantHash, rc.TestVariantAnalysis.UpdateTestVariantTask.UpdateTestVariantTaskInterval, enQTime)
		}
		return nil
	})
	return err
}

func testVariantKeySet(realm string, tvs []*rdbpb.TestVariant) spanner.KeySet {
	ks := spanner.KeySets()
	for _, tv := range tvs {
		if tv.Status == rdbpb.TestVariantStatus_UNEXPECTEDLY_SKIPPED {
			continue
		}
		ks = spanner.KeySets(spanner.Key{realm, tv.TestId, tv.VariantHash}, ks)
	}
	return ks
}

func shouldSkipTestVariant(tv *rdbpb.TestVariant) bool {
	if tv.Status == rdbpb.TestVariantStatus_UNEXPECTEDLY_SKIPPED {
		return true
	}

	for _, trb := range tv.Results {
		tr := trb.Result
		if !tr.Expected && tr.Status != rdbpb.TestStatus_PASS && tr.Status != rdbpb.TestStatus_SKIP {
			// If any result is an unexpected failure, Weetbix should save this test variant.
			return false
		}
	}
	return true
}

func insertRow(ctx context.Context, realm, builder string, tv *rdbpb.TestVariant) (mu *spanner.Mutation, enqueueTime time.Time, err error) {
	status, err := derivedStatus(tv.Status)
	if err != nil {
		return nil, time.Time{}, err
	}

	now := clock.Now(ctx)
	row := map[string]interface{}{
		"Realm":                     realm,
		"TestId":                    tv.TestId,
		"VariantHash":               tv.VariantHash,
		"Variant":                   pbutil.VariantFromResultDB(tv.Variant),
		"Status":                    int64(status),
		"CreateTime":                spanner.CommitTimestamp,
		"StatusUpdateTime":          spanner.CommitTimestamp,
		"Builder":                   builder,
		"Tags":                      extractRequiredTags(tv),
		"NextUpdateTaskEnqueueTime": now,
	}
	if tv.TestMetadata != nil {
		tmd, err := proto.Marshal(pbutil.TestMetadataFromResultDB(tv.TestMetadata))
		if err != nil {
			panic(fmt.Sprintf("failed to marshal TestMetadata to bytes: %q", err))
		}
		row["TestMetadata"] = spanutil.Compressed(tmd)
	}

	return spanutil.InsertMap("AnalyzedTestVariants", row), now, nil
}

func derivedStatus(tvStatus rdbpb.TestVariantStatus) (pb.AnalyzedTestVariantStatus, error) {
	switch {
	case tvStatus == rdbpb.TestVariantStatus_FLAKY:
		// The new test variant has flaky results in a build, the analyzed test
		// variant becomes flaky.
		// Note that this is only true if Weetbix knows the the ingested test
		// results are from builds contribute to CL submissions. Which is true for
		// Chromium, the only project Weetbix supports now.
		return pb.AnalyzedTestVariantStatus_FLAKY, nil
	case tvStatus == rdbpb.TestVariantStatus_UNEXPECTED || tvStatus == rdbpb.TestVariantStatus_EXONERATED:
		return pb.AnalyzedTestVariantStatus_HAS_UNEXPECTED_RESULTS, nil
	default:
		return pb.AnalyzedTestVariantStatus_STATUS_UNSPECIFIED, fmt.Errorf("unsupported test variant status: %s", tvStatus.String())
	}
}

// Get the updated AnalyzedTestVariant status based on the ResultDB test variant
// status.
func updatedStatus(derived, old pb.AnalyzedTestVariantStatus) (pb.AnalyzedTestVariantStatus, error) {
	switch {
	case old == derived:
		return old, nil
	case old == pb.AnalyzedTestVariantStatus_FLAKY:
		// If the AnalyzedTestVariant is already Flaky, its status does not change here.
		return old, nil
	case derived == pb.AnalyzedTestVariantStatus_FLAKY:
		// Any flaky occurrence will make an AnalyzedTestVariant become flaky.
		return derived, nil
	case old == pb.AnalyzedTestVariantStatus_CONSISTENTLY_UNEXPECTED:
		// All results of the ResultDB test variant are unexpected, so AnalyzedTestVariant
		// does need to change status.
		return old, nil
	case old == pb.AnalyzedTestVariantStatus_CONSISTENTLY_EXPECTED || old == pb.AnalyzedTestVariantStatus_NO_NEW_RESULTS:
		// New failures are found, AnalyzedTestVariant needs to change status.
		return derived, nil
	default:
		return pb.AnalyzedTestVariantStatus_STATUS_UNSPECIFIED, fmt.Errorf("unsupported updated Status")
	}
}

func extractRequiredTags(tv *rdbpb.TestVariant) []*pb.StringPair {
	tags := make([]*pb.StringPair, 0)
	knownKeys := make(map[string]struct{})
	for _, tr := range tv.Results {
		for _, t := range tr.Result.GetTags() {
			if _, ok := tagKeys[t.Key]; !ok {
				// We don't care about this tag.
				continue
			}
			if _, ok := knownKeys[t.Key]; ok {
				// We've got this tag.
				continue
			}
			knownKeys[t.Key] = struct{}{}
			tags = append(tags, &pb.StringPair{
				Key:   t.Key,
				Value: t.Value,
			})
		}
	}

	// Ensure determinism by leaving tags in sorted order.
	sortTags(tags)
	return tags
}

// tagsEqual compares two sets of tags.
func tagsEqual(newTags, oldTags []*pb.StringPair) bool {
	if len(newTags) != len(oldTags) {
		return false
	}
	ntStrings := pbutil.StringPairsToStrings(newTags...)
	sort.Strings(ntStrings)
	otStrings := pbutil.StringPairsToStrings(oldTags...)
	sort.Strings(otStrings)

	for i, t := range ntStrings {
		if t != otStrings[i] {
			return false
		}
	}
	return true
}

// updatedTags returns a merged slices of tags.
// * if the same key appears in both newTags and oldTags, use the value from
//   newTags;
// * if a key appears in only one of the slices, append the string pair as it
//   is;
// * if the merged slice is the same as oldTags, return nil to indicate there
//   is no need to update tags.
func updatedTags(newTags, oldTags []*pb.StringPair) []*pb.StringPair {
	switch {
	case len(newTags) == 0:
		return nil
	case len(oldTags) == 0:
		return newTags
	}

	if same := tagsEqual(newTags, oldTags); same {
		return nil
	}

	resultMap := make(map[string]*pb.StringPair)
	for _, t := range oldTags {
		resultMap[t.Key] = t
	}
	for _, t := range newTags {
		resultMap[t.Key] = t
	}

	result := make([]*pb.StringPair, 0, len(resultMap))
	for _, t := range resultMap {
		result = append(result, t)
	}

	// Ensure determinism by leaving tags in sorted order.
	// This is primarily to avoid flakes in unit tests.
	sortTags(result)
	return result
}

// sortTags performs an in-place sort of tags to be in
// ascending key order.
func sortTags(tags []*pb.StringPair) {
	sort.Slice(tags, func(i, j int) bool {
		return tags[i].Key < tags[j].Key
	})
}
