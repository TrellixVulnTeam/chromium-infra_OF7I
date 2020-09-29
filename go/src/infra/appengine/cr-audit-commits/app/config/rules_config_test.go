// Copyright 2018 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package config

import (
	"context"
	"testing"

	"go.chromium.org/luci/gae/impl/memory"
	"go.chromium.org/luci/server/router"

	"infra/appengine/cr-audit-commits/app/rules"

	. "github.com/smartystreets/goconvey/convey"
)

func TestRulesConfig(t *testing.T) {
	t.Parallel()
	Convey("Test RulesConfig", t, func() {
		c := memory.Use(context.Background())
		c = getConfigTestingContext(c, sampleValidRefConfig)

		Convey("Ensure ruleMap keys are valid", func() {
			Middleware(&router.Context{Context: c}, func(c *router.Context) {
				for k := range GetUpdatedRuleMap(c.Context) {
					// This is a special value.
					So(k, ShouldNotEqual, "AuditFailure")
					// ":" is used to separate config name from concrete ref
					// when accepting ref patterns.
					So(k, ShouldNotContainSubstring, ":")
				}
			})
		})
		Convey("Ensure cfg load from LUCI-config service can update ruleMap", func() {
			Middleware(&router.Context{Context: c}, func(c *router.Context) {
				updatedRuleMap := GetUpdatedRuleMap(c.Context)
				So(updatedRuleMap["fakeproject"], ShouldNotBeEmpty)

				refConfig := updatedRuleMap["fakeproject"]
				So(refConfig.BaseRepoURL, ShouldEqual, "https://new.googlesource.com/new")
				So(refConfig.GerritURL, ShouldEqual, "https://new-review.googlesource.com")
				// TODO: For test environment, the MonorailAPIURL should be different.
				So(refConfig.MonorailAPIURL, ShouldEqual, "https://monorail-prod.appspot.com/_ah/api/monorail/v1")
				So(refConfig.MonorailProject, ShouldEqual, "fakeproject")
				So(refConfig.NotifierEmail, ShouldEqual, "notifier@cr-audit-commits-test.appspotmail.com")

				accountRule := refConfig.Rules["manual-changes"]
				So(accountRule.Account, ShouldEqual, "*")

				_, ok := accountRule.Rules[0].(rules.AcknowledgeMerge)
				So(ok, ShouldEqual, true)
				_, ok = accountRule.Rules[1].(rules.AutoCommitsPerDay)
				So(ok, ShouldEqual, true)
				_, ok = accountRule.Rules[2].(rules.AutoRevertsPerDay)
				So(ok, ShouldEqual, true)
				_, ok = accountRule.Rules[3].(rules.ChangeReviewed)
				So(ok, ShouldEqual, true)
				So(accountRule.Rules[3].(rules.ChangeReviewed).GetRobots(), ShouldResemble, []string{"robot0@example.com", "robot1@example.com"})
				_, ok = accountRule.Rules[4].(rules.CulpritAge)
				So(ok, ShouldEqual, true)
				_, ok = accountRule.Rules[5].(rules.CulpritInBuild)
				So(ok, ShouldEqual, true)
				_, ok = accountRule.Rules[6].(rules.FailedBuildIsAppropriateFailure)
				So(ok, ShouldEqual, true)
				_, ok = accountRule.Rules[7].(rules.OnlyCommitsOwnChange)
				So(ok, ShouldEqual, true)
				_, ok = accountRule.Rules[8].(rules.OnlyMergeApprovedChange)
				So(ok, ShouldEqual, true)
				So(accountRule.Rules[8].(rules.OnlyMergeApprovedChange).GetAllowedUsers(), ShouldResemble, []string{"user0@example.com"})
				So(accountRule.Rules[8].(rules.OnlyMergeApprovedChange).GetAllowedRobots(), ShouldResemble, []string{"robot0@example.com"})
				_, ok = accountRule.Rules[9].(rules.OnlyModifiesFilesAndDirsRule)
				So(ok, ShouldEqual, true)
				So(accountRule.Rules[9].(rules.OnlyModifiesFilesAndDirsRule).GetName(), ShouldEqual, "OnlyModifiesReleaseFiles")
				So(accountRule.Rules[9].(rules.OnlyModifiesFilesAndDirsRule).GetFiles(), ShouldResemble, []string{"filea", "fileb"})
				So(accountRule.Rules[9].(rules.OnlyModifiesFilesAndDirsRule).GetDirs(), ShouldResemble, []string{"dira"})
				_, ok = accountRule.Rules[10].(rules.RevertOfCulprit)
				So(ok, ShouldEqual, true)
				So(accountRule.Notification.(rules.CommentOrFileMonorailIssue).GetComponents(), ShouldResemble, []string{"Test>Component"})
				So(accountRule.Notification.(rules.CommentOrFileMonorailIssue).GetLabels(), ShouldResemble, []string{"CommitLog-Audit-Violation", "TBR-Violation"})
			})
		})
	})
}
