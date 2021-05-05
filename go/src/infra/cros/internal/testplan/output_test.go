package testplan

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	configpb "go.chromium.org/chromiumos/config/go/api"
	"go.chromium.org/chromiumos/config/go/api/software"
	buildpb "go.chromium.org/chromiumos/config/go/build/api"
	"go.chromium.org/chromiumos/config/go/payload"
	"go.chromium.org/chromiumos/config/go/test/plan"
)

// buildSummary is a convenience to reduce boilerplate when creating
// SystemImage_BuildSummary in test cases.
func buildSummary(overlay, kernelVersion, chipsetOverlay, arcVersion string) *buildpb.SystemImage_BuildSummary {
	return &buildpb.SystemImage_BuildSummary{
		BuildTarget: &buildpb.SystemImage_BuildTarget{
			PortageBuildTarget: &buildpb.Portage_BuildTarget{
				OverlayName: overlay,
			},
		},
		Kernel: &buildpb.SystemImage_BuildSummary_Kernel{
			Version: kernelVersion,
		},
		Chipset: &buildpb.SystemImage_BuildSummary_Chipset{
			Overlay: chipsetOverlay,
		},
		Arc: &buildpb.SystemImage_BuildSummary_Arc{
			Version: arcVersion,
		},
	}
}

// flatConfig is a convenience to reduce boilerplate when creating FlatConfig in
// test cases.
func flatConfig(
	designConfigID,
	buildTarget string,
	fingerprintLoc configpb.HardwareFeatures_Fingerprint_Location,
) *payload.FlatConfig {
	return &payload.FlatConfig{
		HwDesignConfig: &configpb.Design_Config{
			Id: &configpb.DesignConfigId{
				Value: designConfigID,
			},
			HardwareFeatures: &configpb.HardwareFeatures{
				Fingerprint: &configpb.HardwareFeatures_Fingerprint{
					Location: fingerprintLoc,
				},
			},
		},
		SwConfig: &software.SoftwareConfig{
			SystemBuildTarget: &buildpb.SystemImage_BuildTarget{
				PortageBuildTarget: &buildpb.Portage_BuildTarget{
					OverlayName: buildTarget,
				},
			},
		},
	}
}

var buildSummaryList = &buildpb.SystemImage_BuildSummaryList{
	Values: []*buildpb.SystemImage_BuildSummary{
		buildSummary("project1", "4.14", "chipsetA", ""),
		buildSummary("project2", "4.14", "chipsetB", ""),
		buildSummary("project3", "5.4", "chipsetA", ""),
		buildSummary("project4", "3.18", "chipsetC", "R"),
		buildSummary("project5", "4.14", "chipsetA", ""),
		buildSummary("project6", "4.14", "chipsetB", "P"),
	},
}

var flatConfigList = &payload.FlatConfigList{
	Values: []*payload.FlatConfig{
		flatConfig("config1", "project1", configpb.HardwareFeatures_Fingerprint_KEYBOARD_BOTTOM_LEFT),
		flatConfig("config2", "project1", configpb.HardwareFeatures_Fingerprint_NOT_PRESENT),
		flatConfig("config3", "project3", configpb.HardwareFeatures_Fingerprint_LOCATION_UNKNOWN),
		flatConfig("config4", "project4", configpb.HardwareFeatures_Fingerprint_LEFT_SIDE),
	},
}

func TestGenerateOutputs(t *testing.T) {
	tests := []struct {
		name     string
		input    *plan.SourceTestPlan
		expected []*Output
	}{
		{
			name: "kernel versions",
			input: &plan.SourceTestPlan{
				Requirements: &plan.SourceTestPlan_Requirements{
					KernelVersions: &plan.SourceTestPlan_Requirements_KernelVersions{},
				},
			},
			expected: []*Output{
				{
					Name:         "kernel-3.18",
					BuildTargets: []string{"project4"},
				},
				{
					Name:         "kernel-4.14",
					BuildTargets: []string{"project1", "project2", "project5", "project6"},
				},
				{
					Name:         "kernel-5.4",
					BuildTargets: []string{"project3"},
				},
			},
		},
		{
			name: "soc families",
			input: &plan.SourceTestPlan{
				Requirements: &plan.SourceTestPlan_Requirements{
					SocFamilies: &plan.SourceTestPlan_Requirements_SocFamilies{},
				},
			},
			expected: []*Output{
				{
					Name:         "soc-chipsetA",
					BuildTargets: []string{"project1", "project3", "project5"},
				},
				{
					Name:         "soc-chipsetB",
					BuildTargets: []string{"project2", "project6"},
				},
				{
					Name:         "soc-chipsetC",
					BuildTargets: []string{"project4"},
				},
			},
		},
		{
			name: "build targets and designs",
			input: &plan.SourceTestPlan{
				Requirements: &plan.SourceTestPlan_Requirements{
					KernelVersions: &plan.SourceTestPlan_Requirements_KernelVersions{},
					Fingerprint:    &plan.SourceTestPlan_Requirements_Fingerprint{},
				},
			},
			// TODO(b/182898188): Intersect outputs with DesignConfigIds and
			// outputs with BuildTargets to reduce redundant testing (e.g.
			// config1 is project1).
			expected: []*Output{
				{
					Name:            "fp-present",
					DesignConfigIds: []string{"config1", "config4"},
				},
				{
					Name:         "kernel-3.18",
					BuildTargets: []string{"project4"},
				},
				{
					Name:         "kernel-4.14",
					BuildTargets: []string{"project1", "project2", "project5", "project6"},
				},
				{
					Name:         "kernel-5.4",
					BuildTargets: []string{"project3"},
				},
			},
		},
		{
			name: "multiple requirements",
			input: &plan.SourceTestPlan{
				Requirements: &plan.SourceTestPlan_Requirements{
					KernelVersions: &plan.SourceTestPlan_Requirements_KernelVersions{},
					SocFamilies:    &plan.SourceTestPlan_Requirements_SocFamilies{},
					ArcVersions:    &plan.SourceTestPlan_Requirements_ArcVersions{},
				},
			},
			expected: []*Output{
				{
					Name:         "kernel-4.14:soc-chipsetA",
					BuildTargets: []string{"project1", "project5"},
				},
				{
					Name:         "kernel-4.14:soc-chipsetB:arc-P",
					BuildTargets: []string{"project6"},
				},
				{
					Name:         "kernel-5.4:soc-chipsetA",
					BuildTargets: []string{"project3"},
				},
				{
					Name:         "kernel-3.18:soc-chipsetC:arc-R",
					BuildTargets: []string{"project4"},
				},
			},
		},
		{
			name: "with test tags",
			input: &plan.SourceTestPlan{
				Requirements: &plan.SourceTestPlan_Requirements{
					SocFamilies: &plan.SourceTestPlan_Requirements_SocFamilies{},
				},
				TestTags:        []string{"componentA"},
				TestTagExcludes: []string{"flaky"},
			},
			expected: []*Output{
				{
					Name:            "soc-chipsetA",
					BuildTargets:    []string{"project1", "project3", "project5"},
					TestTags:        []string{"componentA"},
					TestTagExcludes: []string{"flaky"},
				},
				{
					Name:            "soc-chipsetB",
					BuildTargets:    []string{"project2", "project6"},
					TestTags:        []string{"componentA"},
					TestTagExcludes: []string{"flaky"},
				},
				{
					Name:            "soc-chipsetC",
					BuildTargets:    []string{"project4"},
					TestTags:        []string{"componentA"},
					TestTagExcludes: []string{"flaky"},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			outputs, err := generateOutputs(test.input, buildSummaryList, flatConfigList)

			if err != nil {
				t.Fatalf("generateOutputs failed: %s", err)
			}

			if diff := cmp.Diff(
				test.expected,
				outputs,
				cmpopts.SortSlices(func(i, j *Output) bool {
					return i.Name < j.Name
				}),
				cmpopts.SortSlices(func(i, j string) bool {
					return i < j
				}),
			); diff != "" {
				t.Errorf("generateOutputs returned unexpected diff (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGenerateOutputsErrors(t *testing.T) {
	tests := []struct {
		name  string
		input *plan.SourceTestPlan
	}{
		{
			name: "no requirements ",
			input: &plan.SourceTestPlan{
				EnabledTestEnvironments: []plan.SourceTestPlan_TestEnvironment{
					plan.SourceTestPlan_HARDWARE,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := generateOutputs(test.input, buildSummaryList, flatConfigList); err == nil {
				t.Errorf("Expected error from generateOutputs")
			}
		})
	}
}
