package testplan

import (
	"fmt"
	"infra/tools/dirmd"

	planpb "go.chromium.org/chromiumos/config/go/test/plan"
	"go.chromium.org/luci/common/errors"
)

// ValidateMapping validates ChromeOS test config in mapping.
func ValidateMapping(mapping *dirmd.Mapping) error {
	for dir, metadata := range mapping.Dirs {
		for _, sourceTestPlan := range metadata.GetChromeos().GetCq().GetSourceTestPlans() {
			if err := validateEnabledTestEnvironments(sourceTestPlan); err != nil {
				return errors.Annotate(err, "validation failed for %s", dir).Err()
			}
		}
	}
	return nil
}

func validateEnabledTestEnvironments(plan *planpb.SourceTestPlan) error {
	for _, env := range plan.GetEnabledTestEnvironments() {
		if env == planpb.SourceTestPlan_TEST_ENVIRONMENT_UNSPECIFIED {
			return fmt.Errorf("TEST_ENVIRONMENT_UNSPECIFIED cannot be used in enabled_test_environments")
		}
	}

	if len(plan.GetEnabledTestEnvironments()) == 0 {
		return fmt.Errorf("enabled_test_environments must not be empty")
	}

	return nil
}
