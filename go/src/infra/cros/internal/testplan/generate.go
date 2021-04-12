package testplan

import (
	"infra/cros/internal/gerrit"

	buildpb "go.chromium.org/chromiumos/config/go/build/api"
	"go.chromium.org/chromiumos/config/go/test/plan"
)

func Generate(
	changeRevs []*gerrit.ChangeRev, buildSummaryList *buildpb.SystemImage_BuildSummaryList,
) ([]*Output, error) {
	projectMappingInfos, err := computeProjectMappingInfos(changeRevs)
	if err != nil {
		return nil, err
	}

	var sourceTestPlans []*plan.SourceTestPlan

	for _, pmi := range projectMappingInfos {
		relevantSTPs, err := relevantSourceTestPlans(pmi.Mapping, pmi.AffectedFiles)
		if err != nil {
			return nil, err
		}

		sourceTestPlans = append(sourceTestPlans, relevantSTPs...)
	}

	mergedSourceTestPlan := mergeSourceTestPlans(sourceTestPlans...)

	outputs, err := generateOutputs(mergedSourceTestPlan, buildSummaryList)
	if err != nil {
		return nil, err
	}

	return outputs, nil
}
