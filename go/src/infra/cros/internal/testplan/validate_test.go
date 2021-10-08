package testplan

import (
	"testing"

	"infra/cros/internal/assert"
	"infra/tools/dirmd"
	dirmdpb "infra/tools/dirmd/proto"
	"infra/tools/dirmd/proto/chromeos"

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
			"single starlark file",
			&dirmd.Mapping{
				Dirs: map[string]*dirmdpb.Metadata{
					"a/b/c": {
						TeamEmail: "exampleteam@google.com",
						Chromeos: &chromeos.ChromeOS{
							Cq: &chromeos.ChromeOS_CQ{
								SourceTestPlans: []*plan.SourceTestPlan{
									{
										TestPlanStarlarkFiles: []*plan.SourceTestPlan_TestPlanStarlarkFile{
											{
												Repo: "test/repo",
												Path: "a/b/c/text.txt",
											},
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
			"multiple starlark files",
			&dirmd.Mapping{
				Dirs: map[string]*dirmdpb.Metadata{
					"a/b/c": {
						TeamEmail: "exampleteam@google.com",
						Chromeos: &chromeos.ChromeOS{
							Cq: &chromeos.ChromeOS_CQ{
								SourceTestPlans: []*plan.SourceTestPlan{
									{
										TestPlanStarlarkFiles: []*plan.SourceTestPlan_TestPlanStarlarkFile{
											{
												Repo: "test/repo1",
												Path: "a/b/c/test.star",
											},
											{
												Repo: "test/repo2",
												Path: "test2.star",
											},
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
			"valid regexps",
			&dirmd.Mapping{
				Dirs: map[string]*dirmdpb.Metadata{
					"a/b/c": {
						TeamEmail: "exampleteam@google.com",
						Chromeos: &chromeos.ChromeOS{
							Cq: &chromeos.ChromeOS_CQ{
								SourceTestPlans: []*plan.SourceTestPlan{
									{
										TestPlanStarlarkFiles: []*plan.SourceTestPlan_TestPlanStarlarkFile{
											{
												Repo: "test/repo",
												Path: "a/b/c/text.txt",
											},
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
										TestPlanStarlarkFiles: []*plan.SourceTestPlan_TestPlanStarlarkFile{
											{
												Repo: "test/repo",
												Path: "a/b/c/text.txt",
											},
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
			"starlark files empty",
			&dirmd.Mapping{
				Dirs: map[string]*dirmdpb.Metadata{
					"a/b/c": {
						TeamEmail: "exampleteam@google.com",
						Chromeos: &chromeos.ChromeOS{
							Cq: &chromeos.ChromeOS_CQ{
								SourceTestPlans: []*plan.SourceTestPlan{
									{
										PathRegexps: []string{"a/b/.*"},
									},
								},
							},
						},
					},
				},
			},
			"at least one TestPlanStarlarkFile must be specified",
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
										TestPlanStarlarkFiles: []*plan.SourceTestPlan_TestPlanStarlarkFile{
											{
												Repo: "testrepo",
												Path: "testfile",
											},
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
										TestPlanStarlarkFiles: []*plan.SourceTestPlan_TestPlanStarlarkFile{
											{
												Repo: "testrepo",
												Path: "testfile",
											},
										},
										PathRegexps: []string{`a/b/e/.*\.txt`},
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
