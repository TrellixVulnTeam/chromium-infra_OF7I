// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package resultingester

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/protobuf/proto"

	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/logging"
	cvv0 "go.chromium.org/luci/cv/api/v0"
	rdbpb "go.chromium.org/luci/resultdb/proto/v1"
	"go.chromium.org/luci/server/span"

	"infra/appengine/weetbix/internal/analyzedtestvariants"
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

// locBasedTagKeys are the keys for location based tags. Such tags should be
// saved with analyzed test variants.
var locBasedTagKeys = map[string]struct{}{
	"monorail_component": {},
	"os":                 {},
	"team_email":         {},
}

func shouldIngestForTestVariants(task *taskspb.IngestTestResults) bool {
	switch {
	case task.CvRun == nil:
		return true
	case task.CvRun.Status == cvv0.Run_SUCCEEDED:
		// Only ingest test variants for successful CV (full) runs.
		// Note that this is only true for Chromium, but since Weetbix test variant
		// analysis only supports Chromium at the moment so it's fine.
		// TODO(crbug.com/1259374): Update this after we have per project configs.
		return true
	default:
		return false
	}
}

// createOrUpdateAnalyzedTestVariants looks for new analyzed test variants or
// the ones to be updated, and save them in Spanner.
func createOrUpdateAnalyzedTestVariants(ctx context.Context, realm, builder string, tvs []*rdbpb.TestVariant) error {
	if len(tvs) == 0 {
		return nil
	}

	ks := testVariantKeySet(realm, tvs)
	_, err := span.ReadWriteTransaction(ctx, func(ctx context.Context) error {
		found := make(map[testVariantKey]*pb.AnalyzedTestVariant)
		err := analyzedtestvariants.ReadStatus(ctx, ks, func(atv *pb.AnalyzedTestVariant) error {
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
				if atv.Status == pb.AnalyzedTestVariantStatus_FLAKY {
					// The saved analyzed test variant is a known flake, any status of the new
					// test variant would not change its status.
					continue
				}
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

				if ns != atv.Status {
					vals := map[string]interface{}{
						"Realm":       atv.Realm,
						"TestId":      atv.TestId,
						"VariantHash": atv.VariantHash,
						"Status":      int64(ns),
					}
					if atv.Status == pb.AnalyzedTestVariantStatus_CONSISTENTLY_EXPECTED || atv.Status == pb.AnalyzedTestVariantStatus_NO_NEW_RESULTS {
						// The test variant starts to have unexpected failures again, need
						// to start updating its status.
						now := clock.Now(ctx)
						vals["NextUpdateTaskEnqueueTime"] = now
						tvToEnQTime[k] = now
					}

					ms = append(ms, spanutil.UpdateMap("AnalyzedTestVariants", vals))
				}
			}
		}
		span.BufferWrite(ctx, ms...)
		for tvKey, enQTime := range tvToEnQTime {
			testvariantupdator.Schedule(ctx, realm, tvKey.TestId, tvKey.VariantHash, enQTime)
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
		"Tags":                      extractLocationTags(tv),
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

func extractLocationTags(tv *rdbpb.TestVariant) []*pb.StringPair {
	tags := make([]*pb.StringPair, 0)
	knownKeys := make(map[string]struct{})
	for _, tr := range tv.Results {
		for _, t := range tr.Result.GetTags() {
			if _, ok := locBasedTagKeys[t.Key]; !ok {
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
	return tags
}
