package testplan

import (
	"context"
	"errors"

	buildpb "go.chromium.org/chromiumos/config/go/build/api"
	testpb "go.chromium.org/chromiumos/config/go/test/api"
	"go.chromium.org/chromiumos/config/go/test/plan"
	"go.chromium.org/luci/common/logging"
	"infra/cros/internal/gerrit"
)

// Generate computes CoverageRules based on ChangeRevs or SourceTestPlans.
//
// Exactly one of changeRevs and sourceTestPlans must be non-empty.
// buildSummaryList and dutAttributeList must be non-nil.
func Generate(
	ctx context.Context, changeRevs []*gerrit.ChangeRev, sourceTestPlans []*plan.SourceTestPlan,
	buildSummaryList *buildpb.SystemImage_BuildSummaryList,
	dutAttributeList *testpb.DutAttributeList,
) ([]*testpb.CoverageRule, error) {
	if len(changeRevs) > 0 && len(sourceTestPlans) > 0 {
		return nil, errors.New("change revs and source test plans should not both be passed to generate")
	}

	if len(changeRevs) > 0 {
		logging.Infof(ctx, "calculating SourceTestPlans based on ChangeRevs")

		projectMappingInfos, err := computeProjectMappingInfos(ctx, changeRevs)
		if err != nil {
			return nil, err
		}

		for _, pmi := range projectMappingInfos {
			relevantSTPs, err := relevantSourceTestPlans(ctx, pmi.Mapping, pmi.AffectedFiles)
			if err != nil {
				return nil, err
			}

			sourceTestPlans = append(sourceTestPlans, relevantSTPs...)
		}

		logging.Infof(ctx, "found %d relevant SourceTestPlans", len(sourceTestPlans))
	} else {
		logging.Infof(ctx, "using given SourceTestPlans")
	}

	for _, plan := range sourceTestPlans {
		logging.Debugf(ctx, "relevant SourceTestPlan: %q", plan)
	}

	mergedSourceTestPlan := mergeSourceTestPlans(sourceTestPlans...)

	logging.Infof(ctx, "generating outputs from merged SourceTestPlan")
	logging.Debugf(ctx, "merged SourceTestPlan: %q", mergedSourceTestPlan)

	outputs, err := generateOutputs(
		ctx, mergedSourceTestPlan, buildSummaryList, dutAttributeList,
	)
	if err != nil {
		return nil, err
	}

	return outputs, nil
}
