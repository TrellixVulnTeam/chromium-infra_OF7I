package testplan

import (
	"context"
	"infra/tools/dirmd"
	dirmdpb "infra/tools/dirmd/proto"
	"infra/tools/dirmd/proto/chromeos"
	"testing"

	"github.com/google/go-cmp/cmp"
	"go.chromium.org/chromiumos/config/go/test/plan"
	"google.golang.org/protobuf/testing/protocmp"
)

// buildMapping is a convenience to reduce boilerplate building Mappings.
func buildMapping(dirToSourceTestPlans map[string][]*plan.SourceTestPlan) *dirmd.Mapping {
	mapping := dirmd.NewMapping(len(dirToSourceTestPlans))
	for dir, plans := range dirToSourceTestPlans {
		mapping.Dirs[dir] = &dirmdpb.Metadata{
			Chromeos: &chromeos.ChromeOS{
				Cq: &chromeos.ChromeOS_CQ{
					SourceTestPlans: plans,
				},
			},
		}
	}

	return mapping
}

func TestRelevantSourceTestPlans(t *testing.T) {
	ctx := context.Background()
	// Define example SourceTestPlans for use in test cases.
	// Make each a fn. so each SourceTestPlan in a mapping is unique.
	hwKernelPlan := func() *plan.SourceTestPlan {
		return &plan.SourceTestPlan{
			EnabledTestEnvironments: []plan.SourceTestPlan_TestEnvironment{
				plan.SourceTestPlan_HARDWARE,
			},
			Requirements: &plan.SourceTestPlan_Requirements{
				KernelVersions: &plan.SourceTestPlan_Requirements_KernelVersions{},
			},
		}
	}

	vmKernelPlan := func() *plan.SourceTestPlan {
		return &plan.SourceTestPlan{
			EnabledTestEnvironments: []plan.SourceTestPlan_TestEnvironment{
				plan.SourceTestPlan_VIRTUAL,
			},
			Requirements: &plan.SourceTestPlan_Requirements{
				KernelVersions: &plan.SourceTestPlan_Requirements_KernelVersions{},
			},
		}
	}

	vmSocPlan := func() *plan.SourceTestPlan {
		return &plan.SourceTestPlan{
			EnabledTestEnvironments: []plan.SourceTestPlan_TestEnvironment{
				plan.SourceTestPlan_VIRTUAL,
			},
			Requirements: &plan.SourceTestPlan_Requirements{
				SocFamilies: &plan.SourceTestPlan_Requirements_SocFamilies{},
			},
		}
	}

	onlyCPlan := func() *plan.SourceTestPlan {
		return &plan.SourceTestPlan{
			EnabledTestEnvironments: []plan.SourceTestPlan_TestEnvironment{
				plan.SourceTestPlan_VIRTUAL,
			},
			Requirements: &plan.SourceTestPlan_Requirements{
				SocFamilies: &plan.SourceTestPlan_Requirements_SocFamilies{},
			},
			PathRegexps: []string{`.*\.c`, `.*\.h`},
		}
	}

	onlyPyPlan := func() *plan.SourceTestPlan {
		return &plan.SourceTestPlan{
			EnabledTestEnvironments: []plan.SourceTestPlan_TestEnvironment{
				plan.SourceTestPlan_HARDWARE,
			},
			Requirements: &plan.SourceTestPlan_Requirements{
				SocFamilies: &plan.SourceTestPlan_Requirements_SocFamilies{},
			},
			PathRegexps: []string{`.*\.py`},
		}
	}

	noDocsPlan := func() *plan.SourceTestPlan {
		return &plan.SourceTestPlan{
			EnabledTestEnvironments: []plan.SourceTestPlan_TestEnvironment{
				plan.SourceTestPlan_VIRTUAL,
			},
			Requirements: &plan.SourceTestPlan_Requirements{
				SocFamilies: &plan.SourceTestPlan_Requirements_SocFamilies{},
			},
			PathRegexpExcludes: []string{`.*\.md`, `.*/README`},
		}
	}

	tests := []struct {
		name          string
		mapping       *dirmd.Mapping
		affectedFiles []string
		expected      []*plan.SourceTestPlan
	}{
		{
			name: "no path regexps",
			mapping: buildMapping(map[string][]*plan.SourceTestPlan{
				"otherdir/utils": {hwKernelPlan()},
				"a":              {vmKernelPlan()},
				"a/b":            {vmSocPlan()},
			}),
			affectedFiles: []string{
				// Both files under a/b match vmSocPlan. Plans are deduped
				// in the output.
				"a/b/test1.txt",
				"a/b/test2.txt",
				// Files under a/d match vmKernelPlan.
				"a/d/test.txt",
				// Files under c match no plans.
				"c/test.txt",
			},
			expected: []*plan.SourceTestPlan{vmSocPlan(), vmKernelPlan()},
		},
		{
			name: "path regexps",
			mapping: buildMapping(map[string][]*plan.SourceTestPlan{
				"a/b": {onlyCPlan(), onlyPyPlan()},
			}),
			affectedFiles: []string{
				// Files only match onlyCPlan.
				"a/b/test1.c", "a/b/test1.h",
			},
			expected: []*plan.SourceTestPlan{onlyCPlan()},
		},
		{
			name: "path regexps exclude",

			mapping: buildMapping(map[string][]*plan.SourceTestPlan{
				"a/b": {vmSocPlan(), noDocsPlan()},
			}),
			affectedFiles: []string{
				// Files are excluded from noDocsPlan.
				"a/b/c/CONTRIBUTING.md", "a/b/README",
			},
			expected: []*plan.SourceTestPlan{vmSocPlan()},
		},
		{
			name: "root metadata",
			mapping: buildMapping(map[string][]*plan.SourceTestPlan{
				"a/b": {vmSocPlan()},
				".":   {hwKernelPlan()},
			}),
			affectedFiles: []string{
				// File falls back to the root metadata.
				"otherdir/test.txt",
			},
			expected: []*plan.SourceTestPlan{hwKernelPlan()},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			plans, err := relevantSourceTestPlans(ctx, test.mapping, test.affectedFiles)
			if err != nil {
				t.Fatalf("relevantSourceTestPlans(%v, %v) failed: %s", test.mapping, test.affectedFiles, err)
			}

			if len(plans) != len(test.expected) {
				t.Fatalf(
					"relevantSourceTestPlans(%v, %v) returned %d plans, expected %d",
					test.mapping, test.affectedFiles, len(plans), len(test.expected),
				)
			}

			for i, stp := range plans {
				if diff := cmp.Diff(test.expected[i], stp, protocmp.Transform()); diff != "" {
					t.Errorf(
						"relevantSourceTestPlans(%v, %v) returned unexpected diff on plan at index %d (-want +got):\n%s",
						test.mapping, test.affectedFiles, i, diff,
					)
				}
			}
		})
	}
}

func TestRelevantSourceTestPlansErrors(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name          string
		mapping       *dirmd.Mapping
		affectedFiles []string
	}{
		{
			name: "bad path regexps",
			mapping: buildMapping(map[string][]*plan.SourceTestPlan{
				"a/b": {
					{
						PathRegexps: []string{`okre`, `*badre`},
					},
				},
			}),
			affectedFiles: []string{"a/b/test.txt"},
		},
		{
			name: "bad path regexps exclude",
			mapping: buildMapping(map[string][]*plan.SourceTestPlan{
				"a/b": {
					{
						PathRegexpExcludes: []string{`okre`, `*badre`},
					},
				},
			}),
			affectedFiles: []string{"a/b/test.txt"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := relevantSourceTestPlans(ctx, test.mapping, test.affectedFiles); err == nil {
				t.Errorf("relevantSourceTestPlans(%v, %v) succeeded for bad input, want err", test.mapping, test.affectedFiles)
			}
		})
	}
}
