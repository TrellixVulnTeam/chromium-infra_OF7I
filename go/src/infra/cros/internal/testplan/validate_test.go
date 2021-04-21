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
										KernelVersions: &plan.SourceTestPlan_KernelVersions{},
									},
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
										KernelVersions: &plan.SourceTestPlan_KernelVersions{},
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
										KernelVersions: &plan.SourceTestPlan_KernelVersions{},
									},
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
										KernelVersions: &plan.SourceTestPlan_KernelVersions{},
									},
								},
							},
						},
					},
				},
			},
			"TEST_ENVIRONMENT_UNSPECIFIED cannot be used in enabled_test_environments",
		},
		{
			"no requirements specified",
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
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateMapping(test.mapping)
			assert.ErrorContains(t, err, test.errorSubstring)
		})
	}
}
