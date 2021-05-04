package testplan

import (
	"fmt"

	buildpb "go.chromium.org/chromiumos/config/go/build/api"
	"go.chromium.org/chromiumos/config/go/test/plan"
	"go.chromium.org/luci/common/data/stringset"
)

// Output contains lists of build targets and test tags to run, computed from
// SourceTestPlans. This is a placeholder, representing the information test
// platform would expect.
//
// TODO(b/182898188): Replace this with the actual interface to CTP v2 once it
// is defined.
type Output struct {
	Name            string   `json:"name"`
	BuildTargets    []string `json:"build_targets"`
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

// generateOutputs computes a list of Outputs, based on sourceTestPlan and
// buildSummaryList.
func generateOutputs(
	sourceTestPlan *plan.SourceTestPlan, buildSummaryList *buildpb.SystemImage_BuildSummaryList,
) ([]*Output, error) {
	outputs := []*Output{}

	if sourceTestPlan.GetRequirements().GetKernelVersions() == nil &&
		sourceTestPlan.GetRequirements().GetSocFamilies() == nil &&
		sourceTestPlan.GetRequirements().GetArcVersions() == nil {
		return nil, fmt.Errorf("at least one requirement must be set in SourceTestPlan: %v", sourceTestPlan)
	}

	if sourceTestPlan.GetRequirements().GetKernelVersions() != nil {
		outputs = expandOutputs(outputs, kernelVersionOutputs(sourceTestPlan, buildSummaryList))
	}

	if sourceTestPlan.GetRequirements().GetSocFamilies() != nil {
		outputs = expandOutputs(outputs, socFamilyOutputs(sourceTestPlan, buildSummaryList))
	}

	if sourceTestPlan.GetRequirements().GetArcVersions() != nil {
		outputs = expandOutputs(outputs, arcVersionOutputs(sourceTestPlan, buildSummaryList))

	}

	return outputs, nil
}
