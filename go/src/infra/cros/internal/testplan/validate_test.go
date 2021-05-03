package testplan

import (
	"infra/cros/internal/assert"
	"infra/tools/dirmd"
	dirmdpb "infra/tools/dirmd/proto"
	"infra/tools/dirmd/proto/chromeos"
	"testing"

	"go.chromium.org/chromiumos/config/go/test/plan"
)

func TestValidateMapping(t *testing.T) {
	tests := []struct {
		name    string
		mapping *dirmd.Mapping
	}{
		{
			"no ChromeOS metadata",
			&dirmd.Mapping{
				Dirs: map[string]*dirmdpb.Metadata{
					"a/b/c": {
						TeamEmail: "exampleteam@google.com",
					},
				},
			},
		},
		{
			"single test environment",
			&dirmd.Mapping{
				Dirs: map[string]*dirmdpb.Metadata{
					"a/b/c": {
						TeamEmail: "exampleteam@google.com",
						Chromeos: &chromeos.ChromeOS{
							Cq: &chromeos.ChromeOS_CQ{
								SourceTestPlans: []*plan.SourceTestPlan{
									{
										EnabledTestEnvironments: []plan.SourceTestPlan_TestEnvironment{
											plan.SourceTestPlan_HARDWARE,
										},
										Requirements: &plan.SourceTestPlan_Requirements{
											KernelVersions: &plan.SourceTestPlan_Requirements_KernelVersions{},
										}},
								},
							},
						},
					},
				},
			},
		},
		{
			"multiple test environments",
			&dirmd.Mapping{
				Dirs: map[string]*dirmdpb.Metadata{
					"a/b/c": {
						TeamEmail: "exampleteam@google.com",
						Chromeos: &chromeos.ChromeOS{
							Cq: &chromeos.ChromeOS_CQ{
								SourceTestPlans: []*plan.SourceTestPlan{
									{
										EnabledTestEnvironments: []plan.SourceTestPlan_TestEnvironment{
											plan.SourceTestPlan_HARDWARE,
											plan.SourceTestPlan_VIRTUAL,
										},
										Requirements: &plan.SourceTestPlan_Requirements{
											KernelVersions: &plan.SourceTestPlan_Requirements_KernelVersions{},
										}},
								},
							},
						},
					},
				},
			},
		},
		{
			"valid regexps",
			&dirmd.Mapping{
				Dirs: map[string]*dirmdpb.Metadata{
					"a/b/c": {
						TeamEmail: "exampleteam@google.com",
						Chromeos: &chromeos.ChromeOS{
							Cq: &chromeos.ChromeOS_CQ{
								SourceTestPlans: []*plan.SourceTestPlan{
									{
										EnabledTestEnvironments: []plan.SourceTestPlan_TestEnvironment{
											plan.SourceTestPlan_HARDWARE,
										},
										Requirements: &plan.SourceTestPlan_Requirements{
											KernelVersions: &plan.SourceTestPlan_Requirements_KernelVersions{},
										},
										PathRegexps:        []string{"a/b/c/d/.*"},
										PathRegexpExcludes: []string{`a/b/c/.*\.md`},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			"root directory",
			&dirmd.Mapping{
				Dirs: map[string]*dirmdpb.Metadata{
					".": {
						TeamEmail: "exampleteam@google.com",
						Chromeos: &chromeos.ChromeOS{
							Cq: &chromeos.ChromeOS_CQ{
								SourceTestPlans: []*plan.SourceTestPlan{
									{
										EnabledTestEnvironments: []plan.SourceTestPlan_TestEnvironment{
											plan.SourceTestPlan_HARDWARE,
										},
										Requirements: &plan.SourceTestPlan_Requirements{
											KernelVersions: &plan.SourceTestPlan_Requirements_KernelVersions{},
										},
										PathRegexps:        []string{"a/b/c/d/.*"},
										PathRegexpExcludes: []string{`a/b/c/.*\.md`},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.NilError(t, ValidateMapping(test.mapping))
		})
	}
}

func TestValidateMappingErrors(t *testing.T) {
	tests := []struct {
		name           string
		mapping        *dirmd.Mapping
		errorSubstring string
	}{
		{
			"enabled_test_environments empty",
			&dirmd.Mapping{
				Dirs: map[string]*dirmdpb.Metadata{
					"a/b/c": {
						TeamEmail: "exampleteam@google.com",
						Chromeos: &chromeos.ChromeOS{
							Cq: &chromeos.ChromeOS_CQ{
								SourceTestPlans: []*plan.SourceTestPlan{
									{
										Requirements: &plan.SourceTestPlan_Requirements{
											KernelVersions: &plan.SourceTestPlan_Requirements_KernelVersions{},
										}},
								},
							},
						},
					},
				},
			},
			"enabled_test_environments must not be empty",
		},
		{
			"enabled_test_environments has TEST_ENVIRONMENT_UNSPECIFIED",
			&dirmd.Mapping{
				Dirs: map[string]*dirmdpb.Metadata{
					"a/b/c": {
						TeamEmail: "exampleteam@google.com",
						Chromeos: &chromeos.ChromeOS{
							Cq: &chromeos.ChromeOS_CQ{
								SourceTestPlans: []*plan.SourceTestPlan{
									{
										EnabledTestEnvironments: []plan.SourceTestPlan_TestEnvironment{
											plan.SourceTestPlan_HARDWARE,
											plan.SourceTestPlan_TEST_ENVIRONMENT_UNSPECIFIED,
										},
										Requirements: &plan.SourceTestPlan_Requirements{
											KernelVersions: &plan.SourceTestPlan_Requirements_KernelVersions{},
										}},
								},
							},
						},
					},
				},
			},
			"TEST_ENVIRONMENT_UNSPECIFIED cannot be used in enabled_test_environments",
		},
		{
			"no requirements message",
			&dirmd.Mapping{
				Dirs: map[string]*dirmdpb.Metadata{
					"a/b/c": {
						TeamEmail: "exampleteam@google.com",
						Chromeos: &chromeos.ChromeOS{
							Cq: &chromeos.ChromeOS_CQ{
								SourceTestPlans: []*plan.SourceTestPlan{
									{
										EnabledTestEnvironments: []plan.SourceTestPlan_TestEnvironment{
											plan.SourceTestPlan_HARDWARE,
										},
									},
								},
							},
						},
					},
				},
			},
			"at least one requirement must be specified",
		},
		{
			"empty requirements message",
			&dirmd.Mapping{
				Dirs: map[string]*dirmdpb.Metadata{
					"a/b/c": {
						TeamEmail: "exampleteam@google.com",
						Chromeos: &chromeos.ChromeOS{
							Cq: &chromeos.ChromeOS_CQ{
								SourceTestPlans: []*plan.SourceTestPlan{
									{
										EnabledTestEnvironments: []plan.SourceTestPlan_TestEnvironment{
											plan.SourceTestPlan_HARDWARE,
										},
										Requirements: &plan.SourceTestPlan_Requirements{},
									},
								},
							},
						},
					},
				},
			},
			"at least one requirement must be specified",
		},
		{
			"invalid regexp",
			&dirmd.Mapping{
				Dirs: map[string]*dirmdpb.Metadata{
					"a/b/c": {
						TeamEmail: "exampleteam@google.com",
						Chromeos: &chromeos.ChromeOS{
							Cq: &chromeos.ChromeOS_CQ{
								SourceTestPlans: []*plan.SourceTestPlan{
									{
										EnabledTestEnvironments: []plan.SourceTestPlan_TestEnvironment{
											plan.SourceTestPlan_HARDWARE,
										},
										Requirements: &plan.SourceTestPlan_Requirements{
											KernelVersions: &plan.SourceTestPlan_Requirements_KernelVersions{},
										},
										PathRegexps: []string{"a/b/c/d/["},
									},
								},
							},
						},
					},
				},
			},
			"failed to compile path regexp",
		},
		{
			"invalid regexp prefix",
			&dirmd.Mapping{
				Dirs: map[string]*dirmdpb.Metadata{
					"a/b/c": {
						TeamEmail: "exampleteam@google.com",
						Chromeos: &chromeos.ChromeOS{
							Cq: &chromeos.ChromeOS_CQ{
								SourceTestPlans: []*plan.SourceTestPlan{
									{
										EnabledTestEnvironments: []plan.SourceTestPlan_TestEnvironment{
											plan.SourceTestPlan_HARDWARE,
										},
										Requirements: &plan.SourceTestPlan_Requirements{
											KernelVersions: &plan.SourceTestPlan_Requirements_KernelVersions{},
										}, PathRegexps: []string{`a/b/e/.*\.txt`},
									},
								},
							},
						},
					},
				},
			},
			"path_regexp(_exclude)s defined in a directory that is not the root of the repo must have the sub-directory as a prefix",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateMapping(test.mapping)
			assert.ErrorContains(t, err, test.errorSubstring)
		})
	}
}
