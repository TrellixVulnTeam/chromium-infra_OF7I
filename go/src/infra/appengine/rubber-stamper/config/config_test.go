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
)

var sampleValidCfg = `
	host_configs: {
		key: "test-host"
		value: {}
	}
`

func TestMiddleware(t *testing.T) {
	Convey("loads config and updates context", t, func() {
		var cfg Config
		proto.UnmarshalText(sampleValidCfg, &cfg)

		c := memory.Use(context.Background())
		c = caching.WithEmptyProcessCache(c)
		c = cfgclient.Use(c, cfgmem.New(map[config.Set]cfgmem.Files{
			"services/${appid}": {
				cachedCfg.Path: cfg.String(),
			},
		}))

		Middleware(&router.Context{Context: c}, func(c *router.Context) {
			hostConfigs := Get(c.Context).HostConfigs
			_, ok := hostConfigs["test-host"]
			So(ok, ShouldEqual, true)
		})
	})
}
