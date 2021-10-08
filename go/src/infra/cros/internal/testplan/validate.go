package testplan

import (
	"fmt"
	"regexp"
	"strings"

	"infra/tools/dirmd"

	planpb "go.chromium.org/chromiumos/config/go/test/plan"
	"go.chromium.org/luci/common/errors"
)

// ValidateMapping validates ChromeOS test config in mapping.
func ValidateMapping(mapping *dirmd.Mapping) error {
	validationFns := []func(string, *planpb.SourceTestPlan) error{
		validateAtLeastOneTestPlanStarlarkFile,
		validatePathRegexps,
	}

	for dir, metadata := range mapping.Dirs {
		for _, sourceTestPlan := range metadata.GetChromeos().GetCq().GetSourceTestPlans() {
			for _, fn := range validationFns {
				if err := fn(dir, sourceTestPlan); err != nil {
					return errors.Annotate(err, "validation failed for %s", dir).Err()
				}
			}
		}
	}

	return nil
}

func validateAtLeastOneTestPlanStarlarkFile(_ string, plan *planpb.SourceTestPlan) error {
	if len(plan.GetTestPlanStarlarkFiles()) == 0 {
		return fmt.Errorf("at least one TestPlanStarlarkFile must be specified")
	}

	return nil
}

func validatePathRegexps(dir string, plan *planpb.SourceTestPlan) error {
	for _, re := range append(plan.PathRegexps, plan.PathRegexpExcludes...) {
		if _, err := regexp.Compile(re); err != nil {
			return errors.Annotate(err, "failed to compile path regexp %q", re).Err()
		}

		if dir != "." && !strings.HasPrefix(re, dir) {
			return fmt.Errorf(
				"path_regexp(_exclude)s defined in a directory that is not "+
					"the root of the repo must have the sub-directory as a prefix. "+
					"Invalid regexp %q in directory %q",
				re, dir,
			)
		}
	}

	return nil
}
