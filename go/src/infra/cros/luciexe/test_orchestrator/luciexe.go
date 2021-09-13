// The Test Orchestrator takes a request specifying criteria for tests to run,
// computes an optimal set of tests / HW to run, schedules the tests, and
// processes the results.
//
// See design doc at go/ctp2-dd.
//
// This program implements the luciexe protocol, and can be run locally or on
// Buildbucket. See https://pkg.go.dev/go.chromium.org/luci/luciexe.
package main

import (
	"context"
	"fmt"

	"infra/cros/internal/testorchestrator"

	tpv2 "go.chromium.org/chromiumos/infra/proto/go/test_platform/v2"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/luciexe/build"
)

func main() {
	request := &tpv2.RequestBeta{}
	build.Main(request, nil, nil, func(ctx context.Context, userArgs []string, state *build.State) error {
		return RunOrch(ctx, request)
	})
}

// RunOrch runs tests based on request.
func RunOrch(ctx context.Context, request *tpv2.RequestBeta) error {
	testSpecs := request.GetHwTestRequest().GetTestSpecs()
	if len(testSpecs) == 0 {
		return fmt.Errorf("at least one TestSpec in request required")
	}

	for _, spec := range testSpecs {
		swarmingDims, err := testorchestrator.GetRequestedDimensions(ctx, spec.GetRules().DutCriteria)
		if err != nil {
			return err
		}

		logging.Infof(ctx, "Computed RequestedDimensions: %s", swarmingDims)
	}

	return nil
}
