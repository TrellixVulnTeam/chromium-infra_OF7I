package testplan

import (
	"context"
	"errors"

	"infra/cros/internal/gerrit"
	"infra/cros/internal/testplan/computemapping"
	"infra/cros/internal/testplan/relevance"

	"go.chromium.org/chromiumos/config/go/test/plan"
	"go.chromium.org/luci/common/logging"
)

func FindRelevantPlans(
	ctx context.Context,
	changeRevs []*gerrit.ChangeRev,
	workdirFn computemapping.WorkdirCreation,
) ([]*plan.SourceTestPlan, error) {
	if len(changeRevs) == 0 {
		return nil, errors.New("changeRevs must be non-empty")
	}

	logging.Infof(ctx, "calculating relevant SourceTestPlans based on ChangeRevs")

	projectInfos, err := computemapping.ProjectInfos(ctx, changeRevs, workdirFn)
	if err != nil {
		return nil, err
	}

	var relevantSourceTestPlans []*plan.SourceTestPlan

	for _, pi := range projectInfos {
		stps, err := relevance.SourceTestPlans(ctx, pi.Mapping, pi.AffectedFiles)
		if err != nil {
			return nil, err
		}

		relevantSourceTestPlans = append(relevantSourceTestPlans, stps...)
	}

	logging.Infof(ctx, "found %d relevant SourceTestPlans", len(relevantSourceTestPlans))

	for _, plan := range relevantSourceTestPlans {
		logging.Debugf(ctx, "relevant SourceTestPlan: %q", plan)
	}

	return relevantSourceTestPlans, nil
}
