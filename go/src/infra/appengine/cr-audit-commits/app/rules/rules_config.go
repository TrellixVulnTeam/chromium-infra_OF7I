// Copyright 2017 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package rules

var chromiumRobots = []string{
	"chromium-autoroll@skia-public.iam.gserviceaccount.com",
	"image-builder@chops-service-accounts.iam.gserviceaccount.com",
	"recipe-mega-autoroller@chops-service-accounts.iam.gserviceaccount.com",
}

var fuchsiaRobots = []string{
	"docs-roller@fuchsia-infra.iam.gserviceaccount.com",
	"global-integration-roller@fuchsia-infra.iam.gserviceaccount.com",
}

// RuleMap maps each monitored repository to a list of account/rules structs.
var RuleMap = map[string]*RefConfig{
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
			"autoroll-rules-chromium": AutoRollRulesForFilesAndDirs(
				"chromium-autoroll@skia-public.iam.gserviceaccount.com",
				[]string{
					fileAFDO,
					fileDEPS,
					fileFreeTypeReadme,
					fileFreeTypeConfigH,
					fileFreeTypeOptionH,
					fileFuchsiaSDKLinux,
					fileFuchsiaSDKMac,
					filePerfetto,
					filePgoMac,
					filePgoWin32,
					filePgoWin64,
				}, []string{
					dirCrosProfile,
				}),
			"autoroll-rules-chromium-internal": AutoRollRulesDEPS("chromium-internal-autoroll@skia-corp.google.com.iam.gserviceaccount.com"),
			"autoroll-rules-wpt":               AutoRollRulesLayoutTests("wpt-autoroller@chops-service-accounts.iam.gserviceaccount.com"),
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
				NotificationFunction: FileBugForFinditViolation,
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
				NotificationFunction: FileBugForReleaseBotViolation,
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
				NotificationFunction: FileBugForTBRViolation,
			},
			"images-pins-roller": AutoRollRulesForFileList(
				"images-pins-roller@chops-service-accounts.iam.gserviceaccount.com",
				[]string{"build/images/pins.yaml"},
			),
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
				NotificationFunction: FileBugForTBRViolation,
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
				NotificationFunction: FileBugForTBRViolation,
			},
			"image-autoroller": AutoRollRulesForFilesAndDirs(
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
				NotificationFunction: FileBugForTBRViolation,
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
					OnlyMergeApprovedChange{},
				},
				NotificationFunction: FileBugForMergeApprovalViolation,
			},
			"merge-ack-rules": {
				Account: "*",
				Rules: []Rule{
					AcknowledgeMerge{},
				},
				NotificationFunction: CommentOnBugToAcknowledgeMerge,
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
				NotificationFunction: FileBugForTBRViolation,
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
				NotificationFunction: FileBugForTBRViolation,
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
				NotificationFunction: FileBugForTBRViolation,
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
			"autoroll-rules-skia": AutoRollRulesSkiaManifest("skia-fuchsia-autoroll@skia-buildbots.google.com.iam.gserviceaccount.com"),
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
			"autoroll-rules-skia": AutoRollRulesForFilesAndDirs("skia-autoroll@skia-public.iam.gserviceaccount.com", []string{fileDEPS}, dirsSKCMS),
			"bookmaker":           AutoRollRulesAPIDocs("skia-bookmaker@skia-swarming-bots.iam.gserviceaccount.com"),
			"recreate-skps":       AutoRollRulesForFilesAndDirs("skia-recreate-skps@skia-swarming-bots.iam.gserviceaccount.com", []string{SkiaAsset("go_deps"), SkiaAsset("skp"), "go.mod", "go.sum", "infra/bots/tasks.json"}, []string{}),
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
			"autoroll-rules-skia": AutoRollRulesDEPSAndTasks("skia-autoroll@skia-public.iam.gserviceaccount.com"),
		},
	},
}
