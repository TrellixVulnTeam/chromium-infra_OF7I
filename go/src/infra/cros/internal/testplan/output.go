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
// the intersection of a.BuildTargets and b.BuildTargets. If this intersection
// is empty, no Output is added to the result.
//
// If curOutputs is empty, newOutputs is returned (this function is intended to
// be called multiple times to build up a result, curOutputs is empty in the
// first call).
func expandOutputs(curOutputs, newOutputs []*Output) []*Output {
	if len(curOutputs) == 0 {
		return newOutputs
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
				expandedOutputs = append(expandedOutputs, &Output{
					Name:            fmt.Sprintf("%s:%s", cur.Name, new.Name),
					BuildTargets:    buildTargetIntersection.ToSlice(),
					TestTags:        cur.TestTags,
					TestTagExcludes: cur.TestTagExcludes,
				})
			}
		}
	}

	return expandedOutputs
}

// kernelVersionOutputs returns a output for each kernel version.
func kernelVersionOutputs(
	sourceTestPlan *plan.SourceTestPlan, buildSummaryList *buildpb.SystemImage_BuildSummaryList,
) []*Output {
	kernelVersionToBuildTargets := make(map[string][]string)

	for _, value := range buildSummaryList.GetValues() {
		version := value.Kernel.Version
		if _, found := kernelVersionToBuildTargets[version]; !found {
			kernelVersionToBuildTargets[version] = []string{}
		}

		kernelVersionToBuildTargets[version] = append(
			kernelVersionToBuildTargets[version],
			value.GetBuildTarget().GetPortageBuildTarget().GetOverlayName(),
		)
	}

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
	socFamilyToBuildTargets := make(map[string][]string)

	for _, value := range buildSummaryList.GetValues() {
		socFamily := value.GetChipset().GetOverlay()
		if _, found := socFamilyToBuildTargets[socFamily]; !found {
			socFamilyToBuildTargets[socFamily] = []string{}
		}

		socFamilyToBuildTargets[socFamily] = append(
			socFamilyToBuildTargets[socFamily],
			value.GetBuildTarget().GetPortageBuildTarget().GetOverlayName(),
		)
	}

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

// generateOutputs computes a list of Outputs, based on sourceTestPlan and
// buildSummaryList.
func generateOutputs(
	sourceTestPlan *plan.SourceTestPlan, buildSummaryList *buildpb.SystemImage_BuildSummaryList,
) ([]*Output, error) {
	outputs := []*Output{}

	if sourceTestPlan.GetRequirements().GetKernelVersions() == nil &&
		sourceTestPlan.GetRequirements().GetSocFamilies() == nil {
		return nil, fmt.Errorf("at least one requirement must be set in SourceTestPlan: %v", sourceTestPlan)
	}

	if sourceTestPlan.GetRequirements().GetKernelVersions() != nil {
		outputs = expandOutputs(outputs, kernelVersionOutputs(sourceTestPlan, buildSummaryList))
	}

	if sourceTestPlan.GetRequirements().GetSocFamilies() != nil {
		outputs = expandOutputs(outputs, socFamilyOutputs(sourceTestPlan, buildSummaryList))
	}

	return outputs, nil
}
