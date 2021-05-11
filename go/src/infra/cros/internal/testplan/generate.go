package testplan

import (
	"context"

	buildpb "go.chromium.org/chromiumos/config/go/build/api"
	testpb "go.chromium.org/chromiumos/config/go/test/api"
	"go.chromium.org/chromiumos/config/go/test/plan"
	"go.chromium.org/luci/common/logging"
	"infra/cros/internal/gerrit"
)

func Generate(
	ctx context.Context, changeRevs []*gerrit.ChangeRev,
	buildSummaryList *buildpb.SystemImage_BuildSummaryList,
) ([]*testpb.CoverageRule, error) {
	projectMappingInfos, err := computeProjectMappingInfos(ctx, changeRevs)
	if err != nil {
		return nil, err
	}

	var sourceTestPlans []*plan.SourceTestPlan

	for _, pmi := range projectMappingInfos {
		relevantSTPs, err := relevantSourceTestPlans(ctx, pmi.Mapping, pmi.AffectedFiles)
		if err != nil {
			return nil, err
		}

		sourceTestPlans = append(sourceTestPlans, relevantSTPs...)
	}

	logging.Infof(ctx, "found %d relevant SourceTestPlans", len(sourceTestPlans))

	for _, plan := range sourceTestPlans {
		logging.Debugf(ctx, "relevant SourceTestPlan: %q", plan)
	}

	mergedSourceTestPlan := mergeSourceTestPlans(sourceTestPlans...)

	logging.Infof(ctx, "generating outputs from merged SourceTestPlan")
	logging.Debugf(ctx, "merged SourceTestPlan: %q", mergedSourceTestPlan)

	outputs, err := generateOutputs(
		ctx, mergedSourceTestPlan, buildSummaryList,
	)
	if err != nil {
		return nil, err
	}

	return outputs, nil
}
