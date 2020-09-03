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
		gerrit_url: "https://new-review.googlesource.com"
		starting_commit: "000000"
		monorail_project: "fakeproject"
		notifier_email: "notifier@cr-audit-commits-test.appspotmail.com"
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
			So(Get(c.Context).RefConfigs["fakeproject"].GerritHost, ShouldEqual, "new.googlesource.com")
			So(Get(c.Context).RefConfigs["fakeproject"].GerritUrl, ShouldEqual, "https://new-review.googlesource.com")
			So(GetConfigRevision(c.Context), ShouldNotEqual, "")
		})
	})
}
