// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// package main implements the App Engine based HTTP server to handle request
// to Rotation Proxy
package main

import (
	"go.chromium.org/luci/server"
	"go.chromium.org/luci/server/gaeemulation"
	"go.chromium.org/luci/server/module"
	rpb "infra/appengine/rotation-proxy/proto"
)

func main() {
	modules := []module.Module{
		gaeemulation.NewModuleFromFlags(),
	}

	server.Main(nil, modules, func(srv *server.Server) error {
		rpb.RegisterRotationProxyServiceServer(srv.PRPC, &RotationProxyServer{})
		return nil
	})
}
