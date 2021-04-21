package testplan

import (
	"fmt"
	"infra/tools/dirmd"
	"regexp"
	"strings"

	"github.com/golang/protobuf/proto"
	planpb "go.chromium.org/chromiumos/config/go/test/plan"
	"go.chromium.org/luci/common/errors"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// ValidateMapping validates ChromeOS test config in mapping.
func ValidateMapping(mapping *dirmd.Mapping) error {
	validationFns := []func(string, *planpb.SourceTestPlan) error{
		validateEnabledTestEnvironments,
		validateAtLeastOneRequirement,
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

func validateEnabledTestEnvironments(_ string, plan *planpb.SourceTestPlan) error {
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

func validateAtLeastOneRequirement(_ string, plan *planpb.SourceTestPlan) error {
	requirementFields := []string{
		"kernel_versions",
		"soc_families",
	}

	messageReflect := proto.MessageReflect(plan)

	for _, field := range requirementFields {
		fd := messageReflect.Descriptor().Fields().ByName(protoreflect.Name(field))
		if fd == nil {
			panic(fmt.Sprintf("Could not find field descriptor for %q", field))
		}

		// Requirements must be explicitly set, even if they are an empty
		// message. For singular message fields, it is possible to distinguish
		// between the default value (empty message) and whether the field was
		// explicitly populated with the default value. See documentation of
		// the Has method: https://pkg.go.dev/google.golang.org/protobuf/reflect/protoreflect#Message.
		//
		// Check that the requirement is a singular message field, and then call
		// Has. Note that this condition is only determined by the fields
		// in requirementFields and the SourceTestPlan schema, not by the actual
		// input proto; thus, panic if true, since it should be impossible for
		// code that passes unit tests.
		if !(fd.Cardinality() == protoreflect.Optional && fd.Kind() == protoreflect.MessageKind) {
			panic(fmt.Sprintf("Requirements must be singular message fields. Invalid requirement: %q", field))
		}

		if messageReflect.Has(fd) {
			return nil
		}
	}

	return fmt.Errorf("at least one requirement must be specified")
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
