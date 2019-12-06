// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"net/http"

	"go.chromium.org/luci/appengine/gaemiddleware/standard"
	"go.chromium.org/luci/common/data/rand/mathrand"
	"go.chromium.org/luci/server/router"
	"google.golang.org/appengine"

	"infra/appengine/cros/lab_inventory/app/config"
	"infra/appengine/cros/lab_inventory/app/cron"
	"infra/appengine/cros/lab_inventory/app/frontend"
)

func main() {
	// Dev server likes to restart a lot, and upon a restart math/rand seed is
	// always set to 1, resulting in lots of presumably "random" IDs not being
	// very random. Seed it with real randomness.
	mathrand.SeedRandomly()
	r := router.New()
	mwBase := standard.Base().Extend(config.Middleware)
	// Install auth, config and tsmon handlers.
	standard.InstallHandlers(r)
	frontend.InstallHandlers(r, mwBase)
	cron.InstallHandlers(r, mwBase)

	config.SetupValidation()

	http.DefaultServeMux.Handle("/", r)

	appengine.Main()
}
