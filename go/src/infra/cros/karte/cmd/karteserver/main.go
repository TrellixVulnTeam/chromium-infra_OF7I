// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

// This is the entrypoint for the Karte service in production and dev.
// Control is transferred here, inside the Docker container, when the
// application starts.

import (
	"fmt"

	"go.chromium.org/luci/server"
	"go.chromium.org/luci/server/gaeemulation"
	"go.chromium.org/luci/server/module"
)

// Entrypoint is the server entrypoint. It installs services
// and sets up background processes.
func entrypoint(srv *server.Server) error {
	return fmt.Errorf("not implemented")
}

// Transfer control to the LUCI server
func main() {
	modules := []module.Module{
		gaeemulation.NewModuleFromFlags(),
	}

	server.Main(nil, modules, entrypoint)
}
