package testplan

import (
	"fmt"

	"github.com/golang/protobuf/proto"
	configpb "go.chromium.org/chromiumos/config/go/api"
	buildpb "go.chromium.org/chromiumos/config/go/build/api"
	"go.chromium.org/chromiumos/config/go/payload"

	"go.chromium.org/chromiumos/config/go/test/plan"
	"go.chromium.org/luci/common/data/stringset"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// Output contains lists of build targets or DesignConfigIds and test tags to
// run, computed from SourceTestPlans. This is a placeholder, representing the
// information test platform would expect.
//
// TODO(b/182898188): Replace this with the actual interface to CTP v2 once it
// is defined.
type Output struct {
	Name            string   `json:"name"`
	BuildTargets    []string `json:"build_targets"`
	DesignConfigIds []string `json:"design_config_ids"`
	TestTags        []string `json:"test_tags"`
	TestTagExcludes []string `json:"test_tag_excludes"`
}

// expandOutputs joins newOutputs to curOutputs, by intersecting BuildTargets.
//
// For each combination of Outputs (a, b), where a is in curOutputs, and b is in
// newOutputs, a new Output is added to the result, with BuildTargets that are
// the intersection of a.BuildTargets and b.BuildTargets.
//
// All Outputs in either newOutputs or curOutputs that don't have any
// intersection of BuildTargets are added to the result as is.
//
// For example, if curOutputs is
// {
// 	  {
// 		  Name: "A", BuildTargets: []string{"1"},
// 	  },
// 	  {
// 		  Name: "B", BuildTargets: []string{"2"},
// 	  },
// }
//
// and newOutputs is
//
// {
// 	  {
// 		  Name: "C", BuildTargets: []string{"1", "3"},
// 	  },
// 	  {
// 		  Name: "D", BuildTargets: []string{"4"},
// 	  },
// }
//
// the result is
//
// {
// 	  {
// 		  Name: "A:C", BuildTargets: []string{"1"},
// 	  },
// 	  {
// 		  Name: "B", BuildTargets: []string{"2"},
// 	  },
// 	  {
// 		  Name: "D", BuildTargets: []string{"4"},
// 	  },
// }
//
// because "A" and "C" are joined, "B" and "D" are passed through as is.
//
// If curOutputs is empty, newOutputs is returned (this function is intended to
// be called multiple times to build up a result, curOutputs is empty in the
// first call).
func expandOutputs(curOutputs, newOutputs []*Output) []*Output {
	if len(curOutputs) == 0 {
		return newOutputs
	}

	// Make a map from name to Output for all outputs in curOutputs and
	// newOutputs. If an Output is involved in an intersection, it is removed
	// from unjoinedOutputs.
	unjoinedOutputs := make(map[string]*Output)
	for _, output := range append(curOutputs, newOutputs...) {
		unjoinedOutputs[output.Name] = output
	}

	expandedOutputs := make([]*Output, 0)

	for _, cur := range curOutputs {
		for _, new := range newOutputs {
			buildTargetIntersection := stringset.NewFromSlice(
				cur.BuildTargets...,
			).Intersect(
				stringset.NewFromSlice(new.BuildTargets...),
			)
			if len(buildTargetIntersection) > 0 {
				delete(unjoinedOutputs, cur.Name)
				delete(unjoinedOutputs, new.Name)

				expandedOutputs = append(expandedOutputs, &Output{
					Name:            fmt.Sprintf("%s:%s", cur.Name, new.Name),
					BuildTargets:    buildTargetIntersection.ToSlice(),
					TestTags:        cur.TestTags,
					TestTagExcludes: cur.TestTagExcludes,
				})
			}
		}
	}

	// Return all unjoined outputs as is.
	for _, output := range unjoinedOutputs {
		expandedOutputs = append(expandedOutputs, output)
	}

	return expandedOutputs
}

// partitionBuildTargets groups BuildTarget overlay names in buildSummaryList.
//
// For each BuildSummary in buildSummaryList, keyFn is called to get a string
// key. The result groups all overlay names that share the same string key. If
// keyFn returns the empty string, that BuildSummary is skipped.
func partitionBuildTargets(
	keyFn func(*buildpb.SystemImage_BuildSummary) string, buildSummaryList *buildpb.SystemImage_BuildSummaryList,
) map[string][]string {
	keyToBuildTargets := make(map[string][]string)

	for _, value := range buildSummaryList.GetValues() {
		key := keyFn(value)
		if key == "" {
			continue
		}

		if _, found := keyToBuildTargets[key]; !found {
			keyToBuildTargets[key] = []string{}
		}

		keyToBuildTargets[key] = append(
			keyToBuildTargets[key], value.GetBuildTarget().GetPortageBuildTarget().GetOverlayName(),
		)
	}

	return keyToBuildTargets
}

// kernelVersionOutputs returns a output for each kernel version.
func kernelVersionOutputs(
	sourceTestPlan *plan.SourceTestPlan, buildSummaryList *buildpb.SystemImage_BuildSummaryList,
) []*Output {
	kernelVersionToBuildTargets := partitionBuildTargets(
		func(buildSummary *buildpb.SystemImage_BuildSummary) string {
			return buildSummary.GetKernel().GetVersion()
		},
		buildSummaryList,
	)

	outputs := make([]*Output, 0, len(kernelVersionToBuildTargets))
	for version, buildTargets := range kernelVersionToBuildTargets {
		outputs = append(outputs, &Output{
			Name:            fmt.Sprintf("kernel-%s", version),
			BuildTargets:    buildTargets,
			TestTags:        sourceTestPlan.TestTags,
			TestTagExcludes: sourceTestPlan.TestTagExcludes,
		})
	}

	return outputs
}

// socFamilyOutputs returns an Output for each SoC family.
func socFamilyOutputs(
	sourceTestPlan *plan.SourceTestPlan, buildSummaryList *buildpb.SystemImage_BuildSummaryList,
) []*Output {
	socFamilyToBuildTargets := partitionBuildTargets(
		func(buildSummary *buildpb.SystemImage_BuildSummary) string {
			return buildSummary.GetChipset().GetOverlay()
		},
		buildSummaryList,
	)

	outputs := make([]*Output, 0, len(socFamilyToBuildTargets))
	for socFamily, buildTargets := range socFamilyToBuildTargets {
		outputs = append(outputs, &Output{
			Name:            fmt.Sprintf("soc-%s", socFamily),
			BuildTargets:    buildTargets,
			TestTags:        sourceTestPlan.TestTags,
			TestTagExcludes: sourceTestPlan.TestTagExcludes,
		})
	}

	return outputs
}

func arcVersionOutputs(
	sourceTestPlan *plan.SourceTestPlan, buildSummaryList *buildpb.SystemImage_BuildSummaryList,
) []*Output {
	arcVersionToBuildTargets := partitionBuildTargets(
		func(buildSummary *buildpb.SystemImage_BuildSummary) string {
			return buildSummary.GetArc().GetVersion()
		},
		buildSummaryList,
	)

	outputs := make([]*Output, 0, len(arcVersionToBuildTargets))
	for arcVersion, buildTargets := range arcVersionToBuildTargets {
		outputs = append(outputs, &Output{
			Name:            fmt.Sprintf("arc-%s", arcVersion),
			BuildTargets:    buildTargets,
			TestTags:        sourceTestPlan.TestTags,
			TestTagExcludes: sourceTestPlan.TestTagExcludes,
		})
	}

	return outputs
}

// fingerprintOutputs returns an Output containing all DesignConfigIds with a
// fingerprint sensor.
func fingerprintOutputs(
	sourceTestPlan *plan.SourceTestPlan, flatConfigList *payload.FlatConfigList,
) []*Output {
	var designConfigIds []string

	for _, value := range flatConfigList.Values {
		loc := value.GetHwDesignConfig().GetHardwareFeatures().GetFingerprint().GetLocation()
		if loc == configpb.HardwareFeatures_Fingerprint_LOCATION_UNKNOWN ||
			loc == configpb.HardwareFeatures_Fingerprint_NOT_PRESENT {
			continue
		}

		designConfigIds = append(
			designConfigIds,
			value.GetHwDesignConfig().GetId().GetValue(),
		)
	}

	return []*Output{
		{
			Name:            "fp-present",
			DesignConfigIds: designConfigIds,
			TestTags:        sourceTestPlan.TestTags,
			TestTagExcludes: sourceTestPlan.TestTagExcludes,
		},
	}
}

// typeName is a convenience function for the FullName of m.
func typeName(m proto.Message) protoreflect.FullName {
	return proto.MessageReflect(m).Descriptor().FullName()
}

// generateOutputs computes a list of Outputs, based on sourceTestPlan and
// buildSummaryList.
func generateOutputs(
	sourceTestPlan *plan.SourceTestPlan,
	buildSummaryList *buildpb.SystemImage_BuildSummaryList,
	flatConfigList *payload.FlatConfigList,
) ([]*Output, error) {
	outputs := []*Output{}

	// For each requirement set in sourceTestPlan, switch on the type of the
	// requirement and call the corresponding <requirement>Outputs function.
	//
	// Return an error if no requirements are set, or a requirement is
	// unimplemented.
	var unimplementedReq protoreflect.FieldDescriptor

	hasRequirement := false

	proto.MessageReflect(sourceTestPlan.Requirements).Range(
		func(fd protoreflect.FieldDescriptor, _ protoreflect.Value) bool {
			hasRequirement = true

			// It is not possible to do a type switch on the FieldDescriptor or
			// Value, so switch on the full type name.
			switch fd.Message().FullName() {
			case typeName(&plan.SourceTestPlan_Requirements_ArcVersions{}):
				outputs = expandOutputs(outputs, arcVersionOutputs(sourceTestPlan, buildSummaryList))

			case typeName(&plan.SourceTestPlan_Requirements_KernelVersions{}):
				outputs = expandOutputs(outputs, kernelVersionOutputs(sourceTestPlan, buildSummaryList))

			case typeName(&plan.SourceTestPlan_Requirements_SocFamilies{}):
				outputs = expandOutputs(outputs, socFamilyOutputs(sourceTestPlan, buildSummaryList))

			case typeName(&plan.SourceTestPlan_Requirements_Fingerprint{}):
				outputs = expandOutputs(outputs, fingerprintOutputs(sourceTestPlan, flatConfigList))

			default:
				unimplementedReq = fd
				return false
			}
			return true
		},
	)

	if !hasRequirement {
		return nil, fmt.Errorf("at least one requirement must be set in SourceTestPlan: %v", sourceTestPlan)
	}

	if unimplementedReq != nil {
		return nil, fmt.Errorf("unimplemented requirement %q", unimplementedReq.Name())
	}

	return outputs, nil
}
