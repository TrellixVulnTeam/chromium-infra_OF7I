// Copyright 2017 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package rules

import "fmt"

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
	fileBugForTBRViolation = CommentOrFileMonorailIssue{
		Components: []string{"Infra>Audit"},
		Labels:     []string{"CommitLog-Audit-Violation", "TBR-Violation"},
	}
)

// skiaAsset returns the path to the named Skia asset version file.
func skiaAsset(asset string) string {
	return fmt.Sprintf("infra/bots/assets/%s/VERSION", asset)
}

// ruleMap maps each monitored repository to a list of account/rules structs.
var ruleMap = map[string]*RefConfig{
	// Chromium

	"chromium-src-master": {
		BaseRepoURL: "https://chromium.googlesource.com/chromium/src.git",
		GerritURL:   "https://chromium-review.googlesource.com",
		BranchName:  "master",
		// No special meaning, ToT as of the time this line was added.
		StartingCommit:  "bafa682dc0ce1dde367ba44f31f8ec1ad07e569e",
		MonorailAPIURL:  "https://monorail-prod.appspot.com/_ah/api/monorail/v1",
		MonorailProject: "chromium",
		NotifierEmail:   "notifier@cr-audit-commits.appspotmail.com",
		Rules: map[string]AccountRules{
			"autoroll-rules-chromium": AutoRollRules(
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
			"autoroll-rules-chromium-internal": AutoRollRules("chromium-internal-autoroll@skia-corp.google.com.iam.gserviceaccount.com", []string{"DEPS"}, nil),
			"autoroll-rules-wpt":               AutoRollRules("wpt-autoroller@chops-service-accounts.iam.gserviceaccount.com", nil, []string{"third_party/blink/web_tests"}),
			"findit-rules": {
				Account: "findit-for-me@appspot.gserviceaccount.com",
				Rules: []Rule{
					AutoCommitsPerDay{},
					AutoRevertsPerDay{},
					CulpritAge{},
					CulpritInBuild{},
					FailedBuildIsAppropriateFailure{},
					RevertOfCulprit{},
					OnlyCommitsOwnChange{},
				},
				Notification: CommentOrFileMonorailIssue{
					Components: []string{"Tools>Test>Findit>Autorevert"},
					Labels:     []string{"CommitLog-Audit-Violation"},
				},
			},
			"release-bot-rules": {
				Account: "chrome-release-bot@chromium.org",
				Rules: []Rule{
					OnlyModifiesFilesAndDirsRule{
						Name: "OnlyModifiesReleaseFiles",
						Files: []string{
							"chrome/MAJOR_BRANCH_DATE",
							"chrome/VERSION",
						},
					},
				},
				Notification: CommentOrFileMonorailIssue{
					Components: []string{"Infra>Client>Chrome>Release"},
					Labels:     []string{"CommitLog-Audit-Violation"},
				},
			},
		},
	},
	"chromium-infra": {
		BaseRepoURL: "https://chromium.googlesource.com/infra/infra",
		GerritURL:   "https://chromium-review.googlesource.com",
		BranchName:  "master",
		// No special meaning, ToT as of the time this line was added.
		StartingCommit:  "19683d4800167eb7a1223719d54725808c61b31b",
		MonorailAPIURL:  "https://monorail-prod.appspot.com/_ah/api/monorail/v1",
		MonorailProject: "chromium",
		NotifierEmail:   "notifier@cr-audit-commits.appspotmail.com",
		Rules: map[string]AccountRules{
			"manual-changes": {
				Account: "*",
				Rules: []Rule{
					ChangeReviewed{Robots: chromiumRobots},
				},
				Notification: fileBugForTBRViolation,
			},
			"images-pins-roller": AutoRollRules(
				"images-pins-roller@chops-service-accounts.iam.gserviceaccount.com",
				[]string{"build/images/pins.yaml"},
				nil),
		},
	},
	"chromium-infra-luci-go": {
		BaseRepoURL: "https://chromium.googlesource.com/infra/luci/luci-go",
		GerritURL:   "https://chromium-review.googlesource.com",
		BranchName:  "master",
		// No special meaning, ToT as of the time this line was added.
		StartingCommit:  "fd8c22ff66975b12558be71b8850dee9e02479bd",
		MonorailAPIURL:  "https://monorail-prod.appspot.com/_ah/api/monorail/v1",
		MonorailProject: "chromium",
		NotifierEmail:   "notifier@cr-audit-commits.appspotmail.com",
		Rules: map[string]AccountRules{
			"manual-changes": {
				Account: "*",
				Rules: []Rule{
					ChangeReviewed{Robots: chromiumRobots},
				},
				Notification: fileBugForTBRViolation,
			},
		},
	},
	"chromium-infra-config": {
		BaseRepoURL: "https://chrome-internal.googlesource.com/infradata/config.git",
		GerritURL:   "https://chrome-internal-review.googlesource.com",
		BranchName:  "master",
		// No special meaning, ToT as of the time this line was added.
		StartingCommit:  "62923d0bcfeca4683bb28f5ecbfa34eb840e791e",
		MonorailAPIURL:  "https://monorail-prod.appspot.com/_ah/api/monorail/v1",
		MonorailProject: "chromium",
		NotifierEmail:   "notifier@cr-audit-commits.appspotmail.com",
		Rules: map[string]AccountRules{
			"manual-changes": {
				Account: "*",
				Rules: []Rule{
					ChangeReviewed{Robots: chromiumRobots},
				},
				Notification: fileBugForTBRViolation,
			},
			"image-autoroller": AutoRollRules(
				"image-builder@chops-service-accounts.iam.gserviceaccount.com",
				[]string{
					"configs/gce-provider/vms.cfg",
					"dev-configs/gce-provider-dev/vms.cfg",
				},
				[]string{"images"},
			),
		},
	},
	"chromium-infra-internal": {
		BaseRepoURL: "https://chrome-internal.googlesource.com/infra/infra_internal.git",
		GerritURL:   "https://chrome-internal-review.googlesource.com",
		BranchName:  "master",
		// No special meaning, ToT as of the time this line was added.
		StartingCommit:  "b7ef8a811e812d564a6167be6f8866f496919912",
		MonorailAPIURL:  "https://monorail-prod.appspot.com/_ah/api/monorail/v1",
		MonorailProject: "chromium",
		NotifierEmail:   "notifier@cr-audit-commits.appspotmail.com",
		Rules: map[string]AccountRules{
			"manual-changes": {
				Account: "*",
				Rules: []Rule{
					ChangeReviewed{Robots: chromiumRobots},
				},
				Notification: fileBugForTBRViolation,
			},
		},
	},
	"chromium-src-release-branches": {
		BaseRepoURL:     "https://chromium.googlesource.com/chromium/src.git",
		GerritURL:       "https://chromium-review.googlesource.com",
		MonorailAPIURL:  "https://monorail-prod.appspot.com/_ah/api/monorail/v1",
		MonorailProject: "chromium",
		NotifierEmail:   "notifier@cr-audit-commits.appspotmail.com",
		Rules: map[string]AccountRules{
			"merge-approval-rules": {
				Account: "*",
				Rules: []Rule{
					OnlyMergeApprovedChange{
						AllowedRobots: chromeMergeRobots,
						AllowedUsers:  chromeTPMs,
					},
				},
				Notification: FileBugForMergeApprovalViolation{
					Components: []string{"Programs>PMO>Browser>Release"},
					Labels:     []string{"CommitLog-Audit-Violation", "Merge-Without-Approval"},
				},
			},
			"merge-ack-rules": {
				Account: "*",
				Rules: []Rule{
					AcknowledgeMerge{},
				},
				Notification: CommentOnBugToAcknowledgeMerge{},
			},
		},
		DynamicRefFunction: ReleaseConfig,
	},

	// Fuchsia

	"fuchsia-infra-infra-master": {
		BaseRepoURL: "https://fuchsia.googlesource.com/infra/infra.git",
		GerritURL:   "https://fuchsia-review.googlesource.com",
		BranchName:  "refs/heads/master",
		// No special meaning, ToT as of the time this line was added.
		StartingCommit:  "bd088ea214e2da0b4b3df13388d4c02d97fe0a56",
		MonorailAPIURL:  "https://monorail-prod.appspot.com/_ah/api/monorail/v1",
		MonorailProject: "fuchsia",
		NotifierEmail:   "notifier@cr-audit-commits.appspotmail.com",
		Rules: map[string]AccountRules{
			"manual-changes": {
				Account: "*",
				Rules: []Rule{
					ChangeReviewed{Robots: fuchsiaRobots},
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
		StartingCommit:  "3b321ef9895192d07bb043c64cdcb7aab16be595",
		MonorailAPIURL:  "https://monorail-prod.appspot.com/_ah/api/monorail/v1",
		MonorailProject: "fuchsia",
		NotifierEmail:   "notifier@cr-audit-commits.appspotmail.com",
		Rules: map[string]AccountRules{
			"manual-changes": {
				Account: "*",
				Rules: []Rule{
					ChangeReviewed{Robots: fuchsiaRobots},
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
		StartingCommit:  "54c641caf04e0247cb15718553239bd00dced4eb",
		MonorailAPIURL:  "https://monorail-prod.appspot.com/_ah/api/monorail/v1",
		MonorailProject: "fuchsia",
		NotifierEmail:   "notifier@cr-audit-commits.appspotmail.com",
		Rules: map[string]AccountRules{
			"manual-changes": {
				Account: "*",
				Rules: []Rule{
					ChangeReviewed{Robots: fuchsiaRobots},
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
		StartingCommit:  "ec7b9088a64bb6a71d8e327a0d04ee9a2f6bb9ec",
		MonorailAPIURL:  "https://monorail-prod.appspot.com/_ah/api/monorail/v1",
		MonorailProject: "chromium",
		NotifierEmail:   "notifier@cr-audit-commits.appspotmail.com",
		Rules: map[string]AccountRules{
			"autoroll-rules-skia": AutoRollRules("skia-fuchsia-autoroll@skia-buildbots.google.com.iam.gserviceaccount.com", []string{"manifest/skia"}, nil),
		},
	},

	// Skia

	"skia-master": {
		BaseRepoURL: "https://skia.googlesource.com/skia.git",
		GerritURL:   "https://skia-review.googlesource.com",
		BranchName:  "refs/heads/master",
		// No special meaning, ToT as of the time this line was added.
		StartingCommit:  "82a33425166aacd0726bdd283c6de749420819a8",
		MonorailAPIURL:  "https://monorail-prod.appspot.com/_ah/api/monorail/v1",
		MonorailProject: "chromium",
		NotifierEmail:   "notifier@cr-audit-commits.appspotmail.com",
		Rules: map[string]AccountRules{
			"autoroll-rules-skia": AutoRollRules("skia-autoroll@skia-public.iam.gserviceaccount.com", []string{"DEPS"}, []string{"include/third_party/skcms", "third_party/skcms"}),
			"bookmaker":           AutoRollRules("skia-bookmaker@skia-swarming-bots.iam.gserviceaccount.com", nil, []string{"site/user/api"}),
			"recreate-skps": AutoRollRules(
				"skia-recreate-skps@skia-swarming-bots.iam.gserviceaccount.com",
				[]string{skiaAsset("go_deps"), skiaAsset("skp"), "go.mod", "go.sum", "infra/bots/tasks.json"},
				nil),
		},
	},
	"skia-lottie-ci": {
		BaseRepoURL: "https://skia.googlesource.com/lottie-ci.git",
		GerritURL:   "https://skia-review.googlesource.com",
		BranchName:  "refs/heads/master",
		// No special meaning, ToT as of the time this line was added.
		StartingCommit:  "6844651ced137fd86d73a11cd0c4d74e71c6fb98",
		MonorailAPIURL:  "https://monorail-prod.appspot.com/_ah/api/monorail/v1",
		MonorailProject: "chromium",
		NotifierEmail:   "notifier@cr-audit-commits.appspotmail.com",
		Rules: map[string]AccountRules{
			"autoroll-rules-skia": AutoRollRules("skia-autoroll@skia-public.iam.gserviceaccount.com", []string{"DEPS", "go.mod", "go.sum", "infra/bots/tasks.json"}, nil),
		},
	},
}

// GetRuleMap returns a map of each monitored repository to a list of
// account/rules structs.
func GetRuleMap() map[string]*RefConfig {
	// TODO: Load from a configuration store.
	return ruleMap
}
