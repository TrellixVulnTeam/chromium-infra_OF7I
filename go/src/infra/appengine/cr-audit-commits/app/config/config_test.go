// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package config

import (
	"context"
	"testing"

	"github.com/golang/protobuf/proto"
	"go.chromium.org/luci/config"
	"go.chromium.org/luci/config/cfgclient"
	cfgmem "go.chromium.org/luci/config/impl/memory"
	"go.chromium.org/luci/gae/impl/memory"
	"go.chromium.org/luci/server/caching"
	"go.chromium.org/luci/server/router"

	cpb "infra/appengine/cr-audit-commits/app/proto"

	. "github.com/smartystreets/goconvey/convey"
)

var (
	sampleValidRefConfig = `
		gerrit_host: "new.googlesource.com"
		gerrit_repo: "new"
		ref: "master"
		starting_commit: "000000"
		monorail_project: "fakeproject"
		notifier_email: "notifier@cr-audit-commits-test.appspotmail.com"
		rules: {
			key: "manual-changes"
			value: {
				account: "*"
				rules: {
					changeReviewed: {
					  	robots: "robot0@example.com"
					  	robots: "robot1@example.com"
					}
				}
				notifications: {
					commentOrFileMonorailIssue: {
						components: "Test>Component"
						  labels: "CommitLog-Audit-Violation"
						  labels: "TBR-Violation"
					}
				}
			}
		}
	`
)

func createConfig() *cpb.Config {
	// returns a RefConfig with all required fields.
	var cfg cpb.RefConfig
	So(proto.UnmarshalText(sampleValidRefConfig, &cfg), ShouldBeNil)

	return &cpb.Config{
		RefConfigs: map[string]*cpb.RefConfig{
			"fakeproject": &cfg,
		},
	}
}

func TestMiddleware(t *testing.T) {
	Convey("loads config and updates context", t, func() {
		c := memory.Use(context.Background())
		c = caching.WithEmptyProcessCache(c)
		c = cfgclient.Use(c, cfgmem.New(map[config.Set]cfgmem.Files{
			"services/${appid}": {
				cachedCfg.Path: createConfig().String(),
			},
		}))

		Middleware(&router.Context{Context: c}, func(c *router.Context) {
			refConfig := Get(c.Context).RefConfigs["fakeproject"]
			So(refConfig.GerritHost, ShouldEqual, "new.googlesource.com")
			So(refConfig.GerritRepo, ShouldEqual, "new")

			accountRules := refConfig.Rules["manual-changes"]
			So(accountRules.Account, ShouldEqual, "*")
			_, ok := accountRules.Rules[0].Rule.(*cpb.Rule_ChangeReviewed)
			So(ok, ShouldEqual, true)
			So(accountRules.Rules[0].GetChangeReviewed().Robots, ShouldResemble, []string{
				"robot0@example.com",
				"robot1@example.com",
			})
			_, ok = accountRules.Notifications[0].Notification.(*cpb.Notification_CommentOrFileMonorailIssue)
			So(ok, ShouldEqual, true)
			So(accountRules.Notifications[0].GetCommentOrFileMonorailIssue().Components, ShouldResemble, []string{
				"Test>Component",
			})
			So(accountRules.Notifications[0].GetCommentOrFileMonorailIssue().Labels, ShouldResemble, []string{
				"CommitLog-Audit-Violation",
				"TBR-Violation",
			})

			So(GetConfigRevision(c.Context), ShouldNotEqual, "")
		})
	})
}
