package testplan

import (
	"fmt"
	"infra/tools/dirmd"

	"github.com/golang/protobuf/proto"
	planpb "go.chromium.org/chromiumos/config/go/test/plan"
	"go.chromium.org/luci/common/errors"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// ValidateMapping validates ChromeOS test config in mapping.
func ValidateMapping(mapping *dirmd.Mapping) error {
	validationFns := []func(*planpb.SourceTestPlan) error{
		validateEnabledTestEnvironments,
		validateAtLeastOneRequirement,
	}

	for dir, metadata := range mapping.Dirs {
		for _, sourceTestPlan := range metadata.GetChromeos().GetCq().GetSourceTestPlans() {
			for _, fn := range validationFns {
				if err := fn(sourceTestPlan); err != nil {
					return errors.Annotate(err, "validation failed for %s", dir).Err()
				}
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

func validateAtLeastOneRequirement(plan *planpb.SourceTestPlan) error {
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
