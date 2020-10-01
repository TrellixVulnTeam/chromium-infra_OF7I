// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package config

import (
	"context"
	"testing"

	"github.com/golang/protobuf/proto"
	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/luci/config"
	"go.chromium.org/luci/config/cfgclient"
	cfgmem "go.chromium.org/luci/config/impl/memory"
	"go.chromium.org/luci/gae/impl/memory"
	"go.chromium.org/luci/server/caching"
	"go.chromium.org/luci/server/router"
	cpb "infra/appengine/cr-audit-commits/app/proto"
)

var (
	sampleValidRefConfig = `
		gerrit_host: "new.googlesource.com"
		gerrit_repo: "new"
		ref: "master"
		starting_commit: "123456f31f1d4556abc4fb34c3d8e7e40972c60a"
		monorail_project: "fakeproject"
		rules: {
			key: "manual-changes"
			value: {
				account: "*"
				rules: {acknowledge_merge: {}}
				rules: {auto_commits_per_day: {}}
				rules: {auto_reverts_per_day: {}}
				rules: {
					change_reviewed: {
						robots: "robot0@example.com"
						robots: "robot1@example.com"
					}
				}
				rules: {culprit_age: {}}
				rules: {culprit_in_build: {}}
				rules: {failed_build_is_appropriate_failure: {}}
				rules: {only_commits_own_change: {}}
				rules: {
					only_merge_approved_change: {
						allowed_users: "user0@example.com"
						allowed_robots: "robot0@example.com"
					}
				}
				rules: {
					only_modifies_files_and_dirs_rule: {
						name: "OnlyModifiesReleaseFiles"
						files: "filea"
						files: "fileb"
						dirs: "dira"
					}
				}
				rules: {revert_of_culprit: {}}
				notifications: {
					comment_on_bug_to_acknowledge_merge: {}
				}
				notifications: {
					comment_or_file_monorail_issue: {
						components: "Test>Component"
							labels: "CommitLog-Audit-Violation"
							labels: "TBR-Violation"
					}
				}
				notifications: {
					file_bug_for_merge_approval_violation: {
						components: "Test>Component"
							labels: "CommitLog-Audit-Violation"
					}
				}
			}
		}
	`
)

func createConfig(s string) *cpb.Config {
	// returns a RefConfig with all required fields.
	var cfg cpb.RefConfig
	proto.UnmarshalText(s, &cfg)

	return &cpb.Config{
		RefConfigs: map[string]*cpb.RefConfig{
			"fakeproject": &cfg,
		},
	}
}

// getConfigTestingContext generates a test context to allow use config.
func getConfigTestingContext(c context.Context, s string) context.Context {
	// Put content from sampleValidRefConfig into cachedCfg.
	c = caching.WithEmptyProcessCache(c)
	c = cfgclient.Use(c, cfgmem.New(map[config.Set]cfgmem.Files{
		"services/${appid}": {
			cachedCfg.Path: createConfig(s).String(),
		},
	}))
	return c
}

func TestMiddleware(t *testing.T) {
	Convey("loads config and updates context", t, func() {
		c := memory.Use(context.Background())
		c = getConfigTestingContext(c, sampleValidRefConfig)

		Middleware(&router.Context{Context: c}, func(c *router.Context) {
			refConfig := Get(c.Context).RefConfigs["fakeproject"]
			So(refConfig.GerritHost, ShouldEqual, "new.googlesource.com")
			So(refConfig.GerritRepo, ShouldEqual, "new")

			accountRules := refConfig.Rules["manual-changes"]
			So(accountRules.Account, ShouldEqual, "*")
			_, ok := accountRules.Rules[0].Rule.(*cpb.Rule_AcknowledgeMerge)
			So(ok, ShouldEqual, true)
			_, ok = accountRules.Rules[1].Rule.(*cpb.Rule_AutoCommitsPerDay)
			So(ok, ShouldEqual, true)
			_, ok = accountRules.Rules[2].Rule.(*cpb.Rule_AutoRevertsPerDay)
			So(ok, ShouldEqual, true)
			_, ok = accountRules.Rules[3].Rule.(*cpb.Rule_ChangeReviewed)
			So(ok, ShouldEqual, true)
			So(accountRules.Rules[3].GetChangeReviewed(), ShouldResemble, &cpb.ChangeReviewed{
				Robots: []string{"robot0@example.com", "robot1@example.com"},
			})
			_, ok = accountRules.Rules[4].Rule.(*cpb.Rule_CulpritAge)
			So(ok, ShouldEqual, true)
			_, ok = accountRules.Rules[5].Rule.(*cpb.Rule_CulpritInBuild)
			So(ok, ShouldEqual, true)
			_, ok = accountRules.Rules[6].Rule.(*cpb.Rule_FailedBuildIsAppropriateFailure)
			So(ok, ShouldEqual, true)
			_, ok = accountRules.Rules[7].Rule.(*cpb.Rule_OnlyCommitsOwnChange)
			So(ok, ShouldEqual, true)
			_, ok = accountRules.Rules[8].Rule.(*cpb.Rule_OnlyMergeApprovedChange)
			So(ok, ShouldEqual, true)
			So(accountRules.Rules[8].GetOnlyMergeApprovedChange(), ShouldResemble, &cpb.OnlyMergeApprovedChange{
				AllowedUsers:  []string{"user0@example.com"},
				AllowedRobots: []string{"robot0@example.com"},
			})
			_, ok = accountRules.Rules[9].Rule.(*cpb.Rule_OnlyModifiesFilesAndDirsRule)
			So(ok, ShouldEqual, true)
			So(accountRules.Rules[9].GetOnlyModifiesFilesAndDirsRule(), ShouldResemble, &cpb.OnlyModifiesFilesAndDirsRule{
				Name:  "OnlyModifiesReleaseFiles",
				Files: []string{"filea", "fileb"},
				Dirs:  []string{"dira"},
			})
			_, ok = accountRules.Rules[10].Rule.(*cpb.Rule_RevertOfCulprit)
			So(ok, ShouldEqual, true)
			_, ok = accountRules.Notifications[0].Notification.(*cpb.Notification_CommentOnBugToAcknowledgeMerge)
			So(ok, ShouldEqual, true)
			_, ok = accountRules.Notifications[1].Notification.(*cpb.Notification_CommentOrFileMonorailIssue)
			So(ok, ShouldEqual, true)
			So(accountRules.Notifications[1].GetCommentOrFileMonorailIssue(), ShouldResemble, &cpb.CommentOrFileMonorailIssue{
				Components: []string{"Test>Component"},
				Labels:     []string{"CommitLog-Audit-Violation", "TBR-Violation"},
			})
			_, ok = accountRules.Notifications[2].Notification.(*cpb.Notification_FileBugForMergeApprovalViolation)
			So(ok, ShouldEqual, true)
			So(accountRules.Notifications[2].GetFileBugForMergeApprovalViolation(), ShouldResemble, &cpb.FileBugForMergeApprovalViolation{
				Components: []string{"Test>Component"},
				Labels:     []string{"CommitLog-Audit-Violation"},
			})
			So(GetConfigRevision(c.Context), ShouldNotEqual, "")
		})
	})
}
