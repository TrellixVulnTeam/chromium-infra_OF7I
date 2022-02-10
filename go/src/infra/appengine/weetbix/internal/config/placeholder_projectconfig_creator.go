package config

import (
	"strings"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"

	configpb "infra/appengine/weetbix/internal/config/proto"
	pb "infra/appengine/weetbix/proto/v1"
)

// Creates a placeholder project with the given key as the project key.
func createPlaceholderProjectWithKey(key string) *configpb.ProjectMetadata {
	return &configpb.ProjectMetadata{
		DisplayName: strings.Title(key),
	}
}

// Creates a placeholder Monorail project with default values and a key.
func createPlaceholderMonorailProjectWithKey(key string) *configpb.MonorailProject {
	return &configpb.MonorailProject{
		Project:         key,
		PriorityFieldId: 10,
		Priorities: []*configpb.MonorailPriority{
			{
				Priority: "0",
				Threshold: &configpb.ImpactThreshold{
					TestResultsFailed: &configpb.MetricThreshold{
						OneDay: proto.Int64(1500),
					},
				},
			},
			{
				Priority: "1",
				Threshold: &configpb.ImpactThreshold{
					TestResultsFailed: &configpb.MetricThreshold{
						OneDay: proto.Int64(500),
					},
				},
			},
		},
	}
}

// Creates a placeholder impact threshold config
func createPlaceholderImpactThreshold() *configpb.ImpactThreshold {
	return &configpb.ImpactThreshold{
		TestResultsFailed: &configpb.MetricThreshold{
			OneDay: proto.Int64(1000),
		},
	}
}

// Creates a placeholder Clustering config with default values.
func createPlaceholderClustering() *configpb.Clustering {
	return &configpb.Clustering{
		TestNameRules: []*configpb.TestNameClusteringRule{
			{
				Name:         "Google Test (Value-parameterized)",
				Pattern:      `^ninja:(?P<target>[\w/]+:\w+)/` + `(\w+/)?(?P<suite>\w+)\.(?P<case>\w+)/\w+$`,
				LikeTemplate: `ninja:${target}/%${suite}.${case}%`,
			},
			{
				Name:         "Google Test (Type-parameterized)",
				Pattern:      `^ninja:(?P<target>[\w/]+:\w+)/` + `(\w+/)?(?P<suite>\w+)/\w+\.(?P<case>\w+)$`,
				LikeTemplate: `ninja:${target}/%${suite}/%.${case}`,
			},
		},
	}
}

// Creates a placeholder realms config slice with the specified key as the dataset key.
func createPlaceholderRealmsWithKey(key string) []*configpb.RealmConfig {
	return []*configpb.RealmConfig{
		{
			Name: "ci",
			TestVariantAnalysis: &configpb.TestVariantAnalysisConfig{
				UpdateTestVariantTask: &configpb.UpdateTestVariantTask{
					UpdateTestVariantTaskInterval:   durationpb.New(time.Hour),
					TestVariantStatusUpdateDuration: durationpb.New(6 * time.Hour),
				},
				BqExports: []*configpb.BigQueryExport{
					{
						Table: &configpb.BigQueryExport_BigQueryTable{
							CloudProject: "test-hrd",
							Dataset:      key,
							Table:        "flaky_test_variants",
						},
						Predicate: &pb.AnalyzedTestVariantPredicate{},
					},
				},
			},
		},
	}
}

// Creates a project config with the specified key.
func CreatePlaceholderConfigWithKey(key string) *configpb.ProjectConfig {
	return &configpb.ProjectConfig{
		ProjectMetadata:    createPlaceholderProjectWithKey(key),
		Monorail:           createPlaceholderMonorailProjectWithKey(key),
		BugFilingThreshold: createPlaceholderImpactThreshold(),
		Realms:             createPlaceholderRealmsWithKey(key),
		Clustering:         createPlaceholderClustering(),
	}
}

// Creates a placeholder project config with key "chromium".
func CreatePlaceholderConfig() *configpb.ProjectConfig {
	defaulyKey := "chromium"
	return &configpb.ProjectConfig{
		ProjectMetadata:    createPlaceholderProjectWithKey(defaulyKey),
		Monorail:           createPlaceholderMonorailProjectWithKey(defaulyKey),
		BugFilingThreshold: createPlaceholderImpactThreshold(),
		Realms:             createPlaceholderRealmsWithKey(defaulyKey),
		Clustering:         createPlaceholderClustering(),
	}
}
