package testplan

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"

	"go.chromium.org/chromiumos/config/go/test/plan"
)

func TestMergeSourceTestPlans(t *testing.T) {
	tests := []struct {
		name     string
		input    []*plan.SourceTestPlan
		expected *plan.SourceTestPlan
	}{
		{
			name: "basic",
			input: []*plan.SourceTestPlan{
				{
					EnabledTestEnvironments: []plan.SourceTestPlan_TestEnvironment{
						plan.SourceTestPlan_HARDWARE,
					},
					PathRegexps:        []string{`a/b/.*\.c`},
					PathRegexpExcludes: []string{`.*\.md`},
					Requirements: &plan.SourceTestPlan_Requirements{
						KernelVersions: &plan.SourceTestPlan_Requirements_KernelVersions{},
					}},
				{
					EnabledTestEnvironments: []plan.SourceTestPlan_TestEnvironment{
						plan.SourceTestPlan_HARDWARE,
					},
					TestTags:        []string{"componentA", "componentB"},
					TestTagExcludes: []string{"componentC", "flaky"},
					Requirements: &plan.SourceTestPlan_Requirements{
						KernelVersions: &plan.SourceTestPlan_Requirements_KernelVersions{},
					},
				},
				{
					EnabledTestEnvironments: []plan.SourceTestPlan_TestEnvironment{
						plan.SourceTestPlan_VIRTUAL,
					},
					TestTags:           []string{"componentC"},
					TestTagExcludes:    []string{"flaky"},
					PathRegexpExcludes: []string{`.*README`},
					Requirements: &plan.SourceTestPlan_Requirements{
						SocFamilies: &plan.SourceTestPlan_Requirements_SocFamilies{},
					},
				},
			},
			expected: &plan.SourceTestPlan{
				EnabledTestEnvironments: []plan.SourceTestPlan_TestEnvironment{
					plan.SourceTestPlan_HARDWARE,
					plan.SourceTestPlan_VIRTUAL,
				},
				TestTags:        []string{"componentA", "componentB", "componentC"},
				TestTagExcludes: []string{"flaky"},
				Requirements: &plan.SourceTestPlan_Requirements{
					KernelVersions: &plan.SourceTestPlan_Requirements_KernelVersions{},
					SocFamilies:    &plan.SourceTestPlan_Requirements_SocFamilies{},
				},
			},
		},
		{
			name: "single plan",
			input: []*plan.SourceTestPlan{
				{
					EnabledTestEnvironments: []plan.SourceTestPlan_TestEnvironment{
						plan.SourceTestPlan_VIRTUAL,
					},
					TestTags:           []string{"componentC"},
					TestTagExcludes:    []string{"flaky"},
					PathRegexpExcludes: []string{`.*README`},
					Requirements: &plan.SourceTestPlan_Requirements{
						SocFamilies: &plan.SourceTestPlan_Requirements_SocFamilies{},
					},
				},
			},
			expected: &plan.SourceTestPlan{
				EnabledTestEnvironments: []plan.SourceTestPlan_TestEnvironment{
					plan.SourceTestPlan_VIRTUAL,
				},
				TestTags:        []string{"componentC"},
				TestTagExcludes: []string{"flaky"},
				Requirements: &plan.SourceTestPlan_Requirements{
					SocFamilies: &plan.SourceTestPlan_Requirements_SocFamilies{},
				},
			},
		},
		{
			name:     "no plans",
			input:    []*plan.SourceTestPlan{},
			expected: &plan.SourceTestPlan{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			merged := mergeSourceTestPlans(test.input...)

			if diff := cmp.Diff(
				test.expected,
				merged,
				protocmp.Transform(),
				protocmp.SortRepeated(func(x, y string) bool {
					return x < y
				}),
			); diff != "" {
				t.Errorf(
					"mergeSourceTestPlans(%v) returned unexpected diff (-want +got):\n%s",
					test.input, diff,
				)
			}
		})
	}
}
