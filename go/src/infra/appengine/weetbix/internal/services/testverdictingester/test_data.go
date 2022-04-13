// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package testverdictingester

import (
	"time"

	bbpb "go.chromium.org/luci/buildbucket/proto"
	"go.chromium.org/luci/resultdb/pbutil"
	rdbpb "go.chromium.org/luci/resultdb/proto/v1"
	"google.golang.org/protobuf/types/known/durationpb"
)

func mockedGetBuildRsp(inv string) *bbpb.Build {
	return &bbpb.Build{
		Builder: &bbpb.BuilderID{
			Project: "chromium",
			Bucket:  "ci",
			Builder: "builder",
		},
		Infra: &bbpb.BuildInfra{
			Resultdb: &bbpb.BuildInfra_ResultDB{
				Hostname:   "results.api.cr.dev",
				Invocation: inv,
			},
		},
		Status: bbpb.Status_FAILURE,
	}
}

func mockedQueryTestVariantsRsp() *rdbpb.QueryTestVariantsResponse {
	return &rdbpb.QueryTestVariantsResponse{
		TestVariants: []*rdbpb.TestVariant{
			{
				TestId:      "test_id_1",
				VariantHash: "hash_1",
				Status:      rdbpb.TestVariantStatus_UNEXPECTED,
				Variant:     pbutil.Variant("k1", "v1"),
				Results: []*rdbpb.TestResultBundle{
					{
						Result: &rdbpb.TestResult{
							Status:   rdbpb.TestStatus_FAIL,
							Expected: false,
							Duration: durationpb.New(time.Second * 10),
						},
					},
				},
			},
			{
				TestId:      "test_id_1",
				VariantHash: "hash_2",
				Status:      rdbpb.TestVariantStatus_FLAKY,
				Variant:     pbutil.Variant("k1", "v2"),
				Results: []*rdbpb.TestResultBundle{
					{
						Result: &rdbpb.TestResult{
							Status:   rdbpb.TestStatus_FAIL,
							Expected: false,
							Duration: durationpb.New(time.Second * 10),
						},
					},
					{
						Result: &rdbpb.TestResult{
							Status:   rdbpb.TestStatus_PASS,
							Expected: true,
							Duration: durationpb.New(time.Second),
						},
					},
				},
			},
			{
				TestId:      "test_id_2",
				VariantHash: "hash_1",
				Status:      rdbpb.TestVariantStatus_FLAKY,
				Variant:     pbutil.Variant("k1", "v1"),
				Results: []*rdbpb.TestResultBundle{
					{
						Result: &rdbpb.TestResult{
							Status:   rdbpb.TestStatus_FAIL,
							Expected: false,
							Duration: durationpb.New(time.Second * 10),
						},
					},
					{
						Result: &rdbpb.TestResult{
							Status:   rdbpb.TestStatus_PASS,
							Expected: true,
							Duration: durationpb.New(time.Second),
						},
					},
					{
						Result: &rdbpb.TestResult{
							Status:   rdbpb.TestStatus_PASS,
							Expected: true,
							Duration: durationpb.New(time.Second * 3),
						},
					},
				},
			},
			{
				TestId:      "test_id_2",
				VariantHash: "hash_2",
				Status:      rdbpb.TestVariantStatus_EXONERATED,
				Variant:     pbutil.Variant("k1", "v2"),
				Results: []*rdbpb.TestResultBundle{
					{
						Result: &rdbpb.TestResult{
							Status:   rdbpb.TestStatus_FAIL,
							Expected: false,
							Duration: durationpb.New(time.Second * 10),
						},
					},
					{
						Result: &rdbpb.TestResult{
							Status:   rdbpb.TestStatus_PASS,
							Expected: true,
							Duration: durationpb.New(time.Second),
						},
					},
					{
						Result: &rdbpb.TestResult{
							Status:   rdbpb.TestStatus_PASS,
							Expected: true,
							Duration: durationpb.New(time.Second),
						},
					},
				},
			},
		},
	}
}
