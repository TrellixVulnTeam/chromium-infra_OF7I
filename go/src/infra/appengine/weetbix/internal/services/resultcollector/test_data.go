// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package resultcollector

import (
	rdbpb "go.chromium.org/luci/resultdb/proto/v1"
)

func mockedBatchGetTestVariantsResponse() *rdbpb.BatchGetTestVariantsResponse {
	return &rdbpb.BatchGetTestVariantsResponse{
		TestVariants: []*rdbpb.TestVariant{
			{
				TestId:      "ninja://test_known_flake",
				VariantHash: "variant_hash",
				Status:      rdbpb.TestVariantStatus_FLAKY,
				Results: []*rdbpb.TestResultBundle{
					{
						Result: &rdbpb.TestResult{
							Status: rdbpb.TestStatus_SKIP,
						},
					},
					{
						Result: &rdbpb.TestResult{
							Status: rdbpb.TestStatus_FAIL,
						},
					},
					{
						Result: &rdbpb.TestResult{
							Status: rdbpb.TestStatus_PASS,
						},
					},
				},
			},
			{
				TestId:      "ninja://test_consistent_failure",
				VariantHash: "variant_hash",
				Status:      rdbpb.TestVariantStatus_EXONERATED,
				Results: []*rdbpb.TestResultBundle{
					{
						Result: &rdbpb.TestResult{
							Status: rdbpb.TestStatus_FAIL,
						},
					},
				},
			},
			{
				TestId:      "ninja://test_has_unexpected",
				VariantHash: "variant_hash",
				Status:      rdbpb.TestVariantStatus_EXPECTED,
				Results: []*rdbpb.TestResultBundle{
					{
						Result: &rdbpb.TestResult{
							Status: rdbpb.TestStatus_PASS,
						},
					},
				},
			},
		},
	}
}
