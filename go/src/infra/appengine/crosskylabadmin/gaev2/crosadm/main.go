// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// This is the main entrypoint for the GAEv2 version of CrOSSkylabAdmin.
// As of right now, it is not functional.
package main

import (
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/config/server/cfgmodule"
	"go.chromium.org/luci/server"
	"go.chromium.org/luci/server/gaeemulation"
	"go.chromium.org/luci/server/module"
)

// Main is the entrypoint for the GAEv2 version of CrOSSkylabAdmin.
func main() {
	modules := []module.Module{
		gaeemulation.NewModuleFromFlags(),
		cfgmodule.NewModuleFromFlags(),
	}

	server.Main(nil, modules, func(srv *server.Server) error {
		logging.Infof(srv.Context, "This GAEv2 app does nothing.")
		return nil
	})
}
