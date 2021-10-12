package main

// import (
// 	"context"
// 	"infra/cros/internal/assert"
// 	"testing"

// 	testpb "go.chromium.org/chromiumos/config/go/test/api"
// 	test_api_v1 "go.chromium.org/chromiumos/config/go/test/api/v1"
// 	tpv2 "go.chromium.org/chromiumos/infra/proto/go/test_platform/v2"
// )

// func TestRunOrch(t *testing.T) {
// 	ctx := context.Background()
// 	request := &tpv2.Request{
// 		TestSpecs: []*tpv2.TestSpec{
// 			&tpv2.TestSpec{
// 				Spec: &tpv2.TestSpec_HwTestPlanV1{
// 					HwTestPlanV1: &test_api_v1.HWTestPlan{
// 						CoverageRules: []*testpb.CoverageRule{
// 							&testpb.CoverageRule{
// 								Name: "test_rule1",
// 								TestSuites: []*testpb.TestSuite{
// 									{
// 										Name: "test_suite1",
// 										Spec: &testpb.TestSuite_TestCaseTagCriteria_{
// 											TestCaseTagCriteria: &testpb.TestSuite_TestCaseTagCriteria{
// 												Tags: []string{"kernel"},
// 											},
// 										},
// 									},
// 									{
// 										Name: "test_suite2",
// 										Spec: &testpb.TestSuite_TestCaseIds{
// 											TestCaseIds: &testpb.TestCaseIdList{
// 												TestCaseIds: []*testpb.TestCase_Id{
// 													{
// 														Value: "suiteA",
// 													},
// 												},
// 											},
// 										},
// 									},
// 								},
// 								DutCriteria: []*testpb.DutCriterion{
// 									{
// 										AttributeId: &testpb.DutAttribute_Id{
// 											Value: "dutattr1",
// 										},
// 										Values: []string{"valA", "valB"},
// 									},
// 								},
// 							},
// 						},
// 					},
// 				},
// 			},
// 		},
// 	}

// 	err := RunOrch(ctx, request)
// 	assert.NilError(t, err)
// }

// func TestRunOrchErrors(t *testing.T) {
// 	ctx := context.Background()

// 	testCases := []struct {
// 		name         string
// 		request      *tpv2.Request
// 		errorMessage string
// 	}{
// 		{
// 			"empty request",
// 			&tpv2.Request{},
// 			"at least one TestSpec in request required",
// 		},
// 		{
// 			"empty CoverageRule",
// 			&tpv2.Request{
// 				TestSpecs: []*tpv2.TestSpec{
// 					{
// 						Spec: &tpv2.TestSpec_HwTestPlanV1{
// 							HwTestPlanV1: &test_api_v1.HWTestPlan{
// 								CoverageRules: []*testpb.CoverageRule{},
// 							},
// 						},
// 					},
// 				},
// 			},
// 			"at least one DutCriterion required in each CoverageRule",
// 		},
// 	}

// 	for _, tc := range testCases {
// 		t.Run(tc.name, func(t *testing.T) {
// 			err := RunOrch(ctx, tc.request)
// 			assert.ErrorContains(t, err, tc.errorMessage)
// 		})
// 	}
// }
