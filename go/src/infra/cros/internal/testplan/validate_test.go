package testplan

import (
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
			if err := ValidateMapping(test.mapping); err != nil {
				t.Errorf("ValidateMapping(%v) failed: %s", test.mapping, err)
			}
		})
	}
}

func TestValidateMappingErrors(t *testing.T) {
	tests := []struct {
		name    string
		mapping *dirmd.Mapping
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
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := ValidateMapping(test.mapping); err == nil {
				t.Errorf("ValidateMapping(%v) succeeded with bad Mapping, want err", test.mapping)
			}
		})
	}
}
