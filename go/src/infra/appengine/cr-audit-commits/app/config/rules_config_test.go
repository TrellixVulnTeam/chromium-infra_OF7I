// Copyright 2018 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package config

import (
	"context"
	"testing"

	"go.chromium.org/luci/config"
	"go.chromium.org/luci/config/cfgclient"
	cfgmem "go.chromium.org/luci/config/impl/memory"
	"go.chromium.org/luci/gae/impl/memory"
	"go.chromium.org/luci/server/caching"
	"go.chromium.org/luci/server/router"

	cpb "infra/appengine/cr-audit-commits/app/proto"
	"infra/appengine/cr-audit-commits/app/rules"

	. "github.com/smartystreets/goconvey/convey"
)

func TestRulesConfig(t *testing.T) {
	t.Parallel()
	Convey("Ensure ruleMap keys are valid", t, func() {
		for k := range GetRuleMap() {
			// This is a special value.
			So(k, ShouldNotEqual, "AuditFailure")
			// ":" is used to separate config name from concrete ref
			// when accepting ref patterns.
			So(k, ShouldNotContainSubstring, ":")
		}
	})
	Convey("Ensure cfg load from LUCI-config service can update ruleMap", t, func() {
		c := memory.Use(context.Background())
		c = caching.WithEmptyProcessCache(c)
		c = cfgclient.Use(c, cfgmem.New(map[config.Set]cfgmem.Files{
			"services/${appid}": {
				cachedCfg.Path: createConfig().String(),
			},
		}))

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
			So(accountRule.Rules[0], ShouldResemble, rules.ChangeReviewed{
				ChangeReviewed: &cpb.ChangeReviewed{
					Robots: []string{
						"robot0@example.com",
						"robot1@example.com",
					},
				},
			})
			So(accountRule.Notification, ShouldResemble, rules.CommentOrFileMonorailIssue{
				CommentOrFileMonorailIssue: &cpb.CommentOrFileMonorailIssue{
					Components: []string{"Test>Component"},
					Labels:     []string{"CommitLog-Audit-Violation", "TBR-Violation"},
				},
			})
		})
	})
}
