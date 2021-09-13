package testorchestrator

import (
	"context"
	"fmt"

	testpb "go.chromium.org/chromiumos/config/go/test/api"
	bbpb "go.chromium.org/luci/buildbucket/proto"
	"go.chromium.org/luci/luciexe/build"
)

// GetRequestedDimensions gets RequestedDimensions for Swarming based on
// dutCriteria.
func GetRequestedDimensions(
	ctx context.Context, dutCriteria []*testpb.DutCriterion,
) (dims []*bbpb.RequestedDimension, err error) {
	step, _ := build.StartStep(ctx, "get requested dimensions")
	defer func() { step.End(err) }()

	if len(dutCriteria) == 0 {
		return nil, fmt.Errorf("at least one DutCriterion required in each CoverageRule")
	}

	dims = []*bbpb.RequestedDimension{}

	for _, criterion := range dutCriteria {
		key := criterion.GetAttributeId().GetValue()
		if key == "" {
			return nil, fmt.Errorf("criteria must have DutAttribute id set")
		}

		values := criterion.GetValues()
		if len(values) == 0 {
			return nil, fmt.Errorf("at least one value must be set on DutAttributes")
		}

		dims = append(dims, &bbpb.RequestedDimension{
			Key:   key,
			Value: values[0],
		})
	}

	step.SetSummaryMarkdown(fmt.Sprintf("Computed %d RequestedDimensions", len(dims)))

	return dims, nil
}
