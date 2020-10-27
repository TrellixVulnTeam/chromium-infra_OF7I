// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/server"
	"go.chromium.org/luci/server/router"
)

func main() {
	server.Main(nil, nil, func(srv *server.Server) error {

		srv.Routes.GET("/", router.MiddlewareChain{}, func(c *router.Context) {
			logging.Debugf(c.Context, "Hello world")
			c.Writer.Write([]byte("Hello, world. This is Rubber-Stamper."))
		})

		return nil
	})
}
