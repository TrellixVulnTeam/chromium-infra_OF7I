// Copyright 2018 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package config

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/golang/protobuf/proto"
	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/luci/config"
	"go.chromium.org/luci/config/cfgclient"
	cfgmem "go.chromium.org/luci/config/impl/memory"
	"go.chromium.org/luci/config/validation"
	"go.chromium.org/luci/gae/impl/memory"
	"go.chromium.org/luci/server/caching"
	"go.chromium.org/luci/server/router"
	cpb "infra/appengine/cr-audit-commits/app/proto"
	"infra/appengine/cr-audit-commits/app/rules"
)

func TestRulesConfig(t *testing.T) {
	Convey("Test RulesConfig", t, func() {
		c := memory.Use(context.Background())
		c = getConfigTestingContext(c, sampleValidRefConfig)

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
				_, ok = accountRule.Notification.(rules.CommentOnBugToAcknowledgeMerge)
				So(ok, ShouldEqual, true)
			})
		})
	})
}

// TODO(crbug.com/1078072): will be deleted after removing local config.
func TestDevConfig(t *testing.T) {
	Convey("Ensure template.cfg is equal to local rule map", t, func() {
		content, err := ioutil.ReadFile(
			"../devcfg/config-template.cfg",
		)
		So(err, ShouldBeNil)

		cfg := &cpb.Config{}
		So(proto.UnmarshalText(string(content), cfg), ShouldBeNil)

		validate := func(cfg *cpb.Config) error {
			c := validation.Context{Context: context.Background()}
			validateConfig(&c, cfg)
			return c.Finalize()
		}
		So(validate(cfg), ShouldBeNil)

		c := memory.Use(context.Background())
		c = caching.WithEmptyProcessCache(c)
		c = cfgclient.Use(c, cfgmem.New(map[config.Set]cfgmem.Files{
			"services/${appid}": {
				cachedCfg.Path: string(content),
			},
		}))

		Middleware(&router.Context{Context: c}, func(c *router.Context) {
			updatedRuleMap := GetUpdatedRuleMap(c.Context)

			for key, value := range updatedRuleMap {
				So(ruleMap[key].BaseRepoURL, ShouldEqual, value.BaseRepoURL)
				So(ruleMap[key].GerritURL, ShouldEqual, value.GerritURL)
				So(ruleMap[key].BranchName, ShouldEqual, value.BranchName)
				So(ruleMap[key].StartingCommit, ShouldEqual, value.StartingCommit)
				So(ruleMap[key].MonorailProject, ShouldEqual, value.MonorailProject)
				So(ruleMap[key].Rules, ShouldResemble, value.Rules)
			}
		})
	})
}
