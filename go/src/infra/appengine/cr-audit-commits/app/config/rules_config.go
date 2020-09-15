// Copyright 2017 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package config

import (
	"fmt"
	"time"

	"infra/appengine/cr-audit-commits/app/rules"
)

var (
	// StuckScannerDuration refers how many hours after a ref stops auditing,
	// a bug will be filed.
	StuckScannerDuration = 2 * time.Hour

	// MaxCommitsPerRefUpdate is the maximum commits that the Gitiles git.Log
	// API should return every time it is called.
	MaxCommitsPerRefUpdate = 6000
)

var (
	chromiumRobots = []string{
		"chromium-autoroll@skia-public.iam.gserviceaccount.com",
		"image-builder@chops-service-accounts.iam.gserviceaccount.com",
		"recipe-mega-autoroller@chops-service-accounts.iam.gserviceaccount.com",
	}

	chromeMergeRobots = []string{
		"chrome-release-bot@chromium.org",
		"chromium-release-autoroll@skia-public.iam.gserviceaccount.com",
	}

	chromeTPMs = []string{
		"adetaylor@chromium.org",
		"adetaylor@google.com",
		"benmason@chromium.org",
		"benmason@google.com",
		"bhthompson@chromium.org",
		"bhthompson@google.com",
		"bindusuvarna@chromium.org",
		"bindusuvarna@google.com",
		"cindyb@chromium.org",
		"cindyb@google.com",
		"dgagnon@chromium.org",
		"dgagnon@google.com",
		"djmm@chromium.org",
		"djmm@google.com",
		"geohsu@chromium.org",
		"geohsu@google.com",
		"gkihumba@chromium.org",
		"gkihumba@google.com",
		"govind@chromium.org",
		"govind@google.com",
		"josafat@chromium.org",
		"josafat@chromium.org",
		"kariahda@chromium.org",
		"kariahda@google.com",
		"kbleicher@chromium.org",
		"kbleicher@google.com",
		"ketakid@chromium.org",
		"ketakid@google.com",
		"mmoss@chromium.org",
		"mmoss@google.com",
		"pbommana@chromium.org",
		"pbommana@google.com",
		"shawnku@chromium.org",
		"shawnku@google.com",
		"sheriffbot@chromium.org",
		"srinivassista@chromium.org",
		"srinivassista@google.com",
	}

	fuchsiaRobots = []string{
		"docs-roller@fuchsia-infra.iam.gserviceaccount.com",
		"global-integration-roller@fuchsia-infra.iam.gserviceaccount.com",
	}

	// fileBugForTBRViolation is the notification function for manual-changes
	// rules.
	fileBugForTBRViolation = rules.CommentOrFileMonorailIssue{
		Components: []string{"Infra>Security>Audit"},
		Labels:     []string{"CommitLog-Audit-Violation", "TBR-Violation"},
	}
)

// skiaAsset returns the path to the named Skia asset version file.
func skiaAsset(asset string) string {
	return fmt.Sprintf("infra/bots/assets/%s/VERSION", asset)
}

// ruleMap maps each monitored repository to a list of account/rules structs.
var ruleMap = map[string]*rules.RefConfig{
	// Chromium

	"chromium-src-master": {
		BaseRepoURL: "https://chromium.googlesource.com/chromium/src.git",
		GerritURL:   "https://chromium-review.googlesource.com",
		BranchName:  "master",
		// No special meaning, ToT as of the time this line was added.
		StartingCommit:  "294151f22f1d8516abc4fb34c3d8e7e40972c60a",
		MonorailAPIURL:  "https://monorail-prod.appspot.com/_ah/api/monorail/v1",
		MonorailProject: "chromium",
		NotifierEmail:   "notifier@cr-audit-commits.appspotmail.com",
		Rules: map[string]rules.AccountRules{
			"autoroll-rules-chromium": rules.AutoRollRules(
				"chromium-autoroll@skia-public.iam.gserviceaccount.com",
				[]string{
					"chrome/android/profiles/newest.txt",
					"DEPS",
					"third_party/freetype/README.chromium",
					"third_party/freetype/include/freetype-custom-config/ftconfig.h",
					"third_party/freetype/include/freetype-custom-config/ftoption.h",
					"build/fuchsia/linux.sdk.sha1",
					"build/fuchsia/mac.sdk.sha1",
					"tools/perf/core/perfetto_binary_roller/binary_deps.json",
					"chrome/build/mac.pgo.txt",
					"chrome/build/win32.pgo.txt",
					"chrome/build/win64.pgo.txt",
				},
				[]string{
					"chromeos/profiles",
				}),
			"autoroll-rules-chromium-internal": rules.AutoRollRules("chromium-internal-autoroll@skia-corp.google.com.iam.gserviceaccount.com", []string{"DEPS"}, nil),
			"autoroll-rules-wpt":               rules.AutoRollRules("wpt-autoroller@chops-service-accounts.iam.gserviceaccount.com", nil, []string{"third_party/blink/web_tests"}),
			"findit-rules": {
				Account: "findit-for-me@appspot.gserviceaccount.com",
				Rules: []rules.Rule{
					rules.AutoCommitsPerDay{},
					rules.AutoRevertsPerDay{},
					rules.CulpritAge{},
					rules.CulpritInBuild{},
					rules.FailedBuildIsAppropriateFailure{},
					rules.RevertOfCulprit{},
					rules.OnlyCommitsOwnChange{},
				},
				Notification: rules.CommentOrFileMonorailIssue{
					Components: []string{"Tools>Test>Findit>Autorevert"},
					Labels:     []string{"CommitLog-Audit-Violation"},
				},
			},
			"release-bot-rules": {
				Account: "chrome-release-bot@chromium.org",
				Rules: []rules.Rule{
					rules.OnlyModifiesFilesAndDirsRule{
						Name: "OnlyModifiesReleaseFiles",
						Files: []string{
							"chrome/MAJOR_BRANCH_DATE",
							"chrome/VERSION",
						},
					},
				},
				Notification: rules.CommentOrFileMonorailIssue{
					Components: []string{"Infra>Client>Chrome>Release"},
					Labels:     []string{"CommitLog-Audit-Violation"},
				},
			},
		},
		OverwriteLastKnownCommit: "3abe288b37baa9aaa68bc6aba6fce7169cf9251a",
	},
	"chromium-infra": {
		BaseRepoURL: "https://chromium.googlesource.com/infra/infra",
		GerritURL:   "https://chromium-review.googlesource.com",
		BranchName:  "master",
		// No special meaning, ToT as of the time this line was added.
		StartingCommit:  "5c5cd4c06f35cd650c0ce8dc769b9c2286428aaf",
		MonorailAPIURL:  "https://monorail-prod.appspot.com/_ah/api/monorail/v1",
		MonorailProject: "chromium",
		NotifierEmail:   "notifier@cr-audit-commits.appspotmail.com",
		Rules: map[string]rules.AccountRules{
			"manual-changes": {
				Account: "*",
				Rules: []rules.Rule{
					rules.ChangeReviewed{Robots: chromiumRobots},
				},
				Notification: fileBugForTBRViolation,
			},
			"images-pins-roller": rules.AutoRollRules(
				"images-pins-roller@chops-service-accounts.iam.gserviceaccount.com",
				[]string{"build/images/pins.yaml"},
				nil,
			),
		},
		OverwriteLastKnownCommit: "ee5143202092142870b37f7950f9adb3255dc3c2",
	},
	"chromium-infra-luci-go": {
		BaseRepoURL: "https://chromium.googlesource.com/infra/luci/luci-go",
		GerritURL:   "https://chromium-review.googlesource.com",
		BranchName:  "master",
		// No special meaning, ToT as of the time this line was added.
		StartingCommit:  "48eb0a6f8f6a455b101e4e0e64ef5c8cbf21cbac",
		MonorailAPIURL:  "https://monorail-prod.appspot.com/_ah/api/monorail/v1",
		MonorailProject: "chromium",
		NotifierEmail:   "notifier@cr-audit-commits.appspotmail.com",
		Rules: map[string]rules.AccountRules{
			"manual-changes": {
				Account: "*",
				Rules: []rules.Rule{
					rules.ChangeReviewed{Robots: chromiumRobots},
				},
				Notification: fileBugForTBRViolation,
			},
		},
		OverwriteLastKnownCommit: "a0bb160410724106d9a5799af81d181568da7e1e",
	},
	"chromium-infra-config": {
		BaseRepoURL: "https://chrome-internal.googlesource.com/infradata/config.git",
		GerritURL:   "https://chrome-internal-review.googlesource.com",
		BranchName:  "master",
		// No special meaning, ToT as of the time this line was added.
		StartingCommit:  "174a9e06ba831b3dca2bedb57c5a67fea7ec7995",
		MonorailAPIURL:  "https://monorail-prod.appspot.com/_ah/api/monorail/v1",
		MonorailProject: "chromium",
		NotifierEmail:   "notifier@cr-audit-commits.appspotmail.com",
		Rules: map[string]rules.AccountRules{
			"manual-changes": {
				Account: "*",
				Rules: []rules.Rule{
					rules.ChangeReviewed{Robots: chromiumRobots},
				},
				Notification: fileBugForTBRViolation,
			},
			"image-autoroller": rules.AutoRollRules(
				"image-builder@chops-service-accounts.iam.gserviceaccount.com",
				[]string{
					"configs/gce-provider/vms.cfg",
					"dev-configs/gce-provider-dev/vms.cfg",
				},
				[]string{"images"},
			),
		},
		OverwriteLastKnownCommit: "3f78c0ad316c448178d41ff39bd52aa8b91e9631",
	},
	"chromium-infra-internal": {
		BaseRepoURL: "https://chrome-internal.googlesource.com/infra/infra_internal.git",
		GerritURL:   "https://chrome-internal-review.googlesource.com",
		BranchName:  "master",
		// No special meaning, ToT as of the time this line was added.
		StartingCommit:  "a4beb2be3d337aa260602e4a990101cb8d9b5930",
		MonorailAPIURL:  "https://monorail-prod.appspot.com/_ah/api/monorail/v1",
		MonorailProject: "chromium",
		NotifierEmail:   "notifier@cr-audit-commits.appspotmail.com",
		Rules: map[string]rules.AccountRules{
			"manual-changes": {
				Account: "*",
				Rules: []rules.Rule{
					rules.ChangeReviewed{Robots: chromiumRobots},
				},
				Notification: fileBugForTBRViolation,
			},
		},
		OverwriteLastKnownCommit: "a4beb2be3d337aa260602e4a990101cb8d9b5930",
	},
	"chromium-src-release-branches": {
		BaseRepoURL:     "https://chromium.googlesource.com/chromium/src.git",
		GerritURL:       "https://chromium-review.googlesource.com",
		MonorailAPIURL:  "https://monorail-prod.appspot.com/_ah/api/monorail/v1",
		MonorailProject: "chromium",
		NotifierEmail:   "notifier@cr-audit-commits.appspotmail.com",
		Rules: map[string]rules.AccountRules{
			"merge-approval-rules": {
				Account: "*",
				Rules: []rules.Rule{
					rules.OnlyMergeApprovedChange{
						AllowedRobots: chromeMergeRobots,
						AllowedUsers:  chromeTPMs,
					},
				},
				Notification: rules.FileBugForMergeApprovalViolation{
					Components: []string{"Programs>PMO>Browser>Release"},
					Labels:     []string{"CommitLog-Audit-Violation", "Merge-Without-Approval"},
				},
			},
			"merge-ack-rules": {
				Account: "*",
				Rules: []rules.Rule{
					rules.AcknowledgeMerge{},
				},
				Notification: rules.CommentOnBugToAcknowledgeMerge{},
			},
		},
		DynamicRefFunction: rules.ReleaseConfig,
	},

	// Fuchsia

	"fuchsia-infra-infra-master": {
		BaseRepoURL: "https://fuchsia.googlesource.com/infra/infra.git",
		GerritURL:   "https://fuchsia-review.googlesource.com",
		BranchName:  "refs/heads/master",
		// No special meaning, ToT as of the time this line was added.
		StartingCommit:  "b96a63a0d469c1d240e16be85e0c086a5d61e11e",
		MonorailAPIURL:  "https://monorail-prod.appspot.com/_ah/api/monorail/v1",
		MonorailProject: "fuchsia",
		NotifierEmail:   "notifier@cr-audit-commits.appspotmail.com",
		Rules: map[string]rules.AccountRules{
			"manual-changes": {
				Account: "*",
				Rules: []rules.Rule{
					rules.ChangeReviewed{Robots: fuchsiaRobots},
				},
				Notification: fileBugForTBRViolation,
			},
		},
	},
	"fuchsia-infra-prebuilt-master": {
		BaseRepoURL: "https://fuchsia.googlesource.com/infra/prebuilt.git",
		GerritURL:   "https://fuchsia-review.googlesource.com",
		BranchName:  "refs/heads/master",
		// No special meaning, ToT as of the time this line was added.
		StartingCommit:  "363cc579c331cd99385dcb538280395a20dc8524",
		MonorailAPIURL:  "https://monorail-prod.appspot.com/_ah/api/monorail/v1",
		MonorailProject: "fuchsia",
		NotifierEmail:   "notifier@cr-audit-commits.appspotmail.com",
		Rules: map[string]rules.AccountRules{
			"manual-changes": {
				Account: "*",
				Rules: []rules.Rule{
					rules.ChangeReviewed{Robots: fuchsiaRobots},
				},
				Notification: fileBugForTBRViolation,
			},
		},
	},
	"fuchsia-infra-recipes-master": {
		BaseRepoURL: "https://fuchsia.googlesource.com/infra/recipes.git",
		GerritURL:   "https://fuchsia-review.googlesource.com",
		BranchName:  "refs/heads/master",
		// No special meaning, ToT as of the time this line was added.
		StartingCommit:  "674d79765c372ef9b9389dc2e0d027732165f441",
		MonorailAPIURL:  "https://monorail-prod.appspot.com/_ah/api/monorail/v1",
		MonorailProject: "fuchsia",
		NotifierEmail:   "notifier@cr-audit-commits.appspotmail.com",
		Rules: map[string]rules.AccountRules{
			"manual-changes": {
				Account: "*",
				Rules: []rules.Rule{
					rules.ChangeReviewed{Robots: fuchsiaRobots},
				},
				Notification: fileBugForTBRViolation,
			},
		},
	},
	"fuchsia-topaz-master": {
		BaseRepoURL: "https://fuchsia.googlesource.com/topaz.git",
		GerritURL:   "https://fuchsia-review.googlesource.com",
		BranchName:  "refs/heads/master",
		// No special meaning, ToT as of the time this line was added.
		StartingCommit:  "e49be669d88e7ba848ec60c194265280e4005bb6",
		MonorailAPIURL:  "https://monorail-prod.appspot.com/_ah/api/monorail/v1",
		MonorailProject: "chromium",
		NotifierEmail:   "notifier@cr-audit-commits.appspotmail.com",
		Rules: map[string]rules.AccountRules{
			"autoroll-rules-skia": rules.AutoRollRules("skia-fuchsia-autoroll@skia-buildbots.google.com.iam.gserviceaccount.com", []string{"manifest/skia"}, nil),
		},
		OverwriteLastKnownCommit: "d56fc21874e8fafbed8e1dee3990c3b09d118ec2",
	},

	// Skia

	"skia-master": {
		BaseRepoURL: "https://skia.googlesource.com/skia.git",
		GerritURL:   "https://skia-review.googlesource.com",
		BranchName:  "refs/heads/master",
		// No special meaning, ToT as of the time this line was added.
		StartingCommit:  "2cc126fc74270d5ebd3e477be422ba407b887ceb",
		MonorailAPIURL:  "https://monorail-prod.appspot.com/_ah/api/monorail/v1",
		MonorailProject: "chromium",
		NotifierEmail:   "notifier@cr-audit-commits.appspotmail.com",
		Rules: map[string]rules.AccountRules{
			"autoroll-rules-skia": rules.AutoRollRules("skia-autoroll@skia-public.iam.gserviceaccount.com", []string{"DEPS"}, []string{"include/third_party/skcms", "third_party/skcms"}),
			"bookmaker":           rules.AutoRollRules("skia-bookmaker@skia-swarming-bots.iam.gserviceaccount.com", nil, []string{"site/user/api"}),
			"recreate-skps": rules.AutoRollRules(
				"skia-recreate-skps@skia-swarming-bots.iam.gserviceaccount.com",
				[]string{skiaAsset("go_deps"), skiaAsset("skp"), "go.mod", "go.sum", "infra/bots/tasks.json"},
				nil),
		},
		OverwriteLastKnownCommit: "04b9443274cfe8c58ea2d5be25df63bdc2f41177",
	},
	"skia-lottie-ci": {
		BaseRepoURL: "https://skia.googlesource.com/lottie-ci.git",
		GerritURL:   "https://skia-review.googlesource.com",
		BranchName:  "refs/heads/master",
		// No special meaning, ToT as of the time this line was added.
		StartingCommit:  "50f3badef1e2a2b123517f8991ebe4f8086e9654",
		MonorailAPIURL:  "https://monorail-prod.appspot.com/_ah/api/monorail/v1",
		MonorailProject: "chromium",
		NotifierEmail:   "notifier@cr-audit-commits.appspotmail.com",
		Rules: map[string]rules.AccountRules{
			"autoroll-rules-skia": rules.AutoRollRules("skia-autoroll@skia-public.iam.gserviceaccount.com", []string{"DEPS", "go.mod", "go.sum", "infra/bots/tasks.json"}, nil),
		},
		OverwriteLastKnownCommit: "75b310f345734d0d08f519d25f7b8360b38a5551",
	},
}

// GetRuleMap returns a map of each monitored repository to a list of
// account/rules structs.
func GetRuleMap() map[string]*rules.RefConfig {
	// TODO: Load from a configuration store.
	return ruleMap
}
