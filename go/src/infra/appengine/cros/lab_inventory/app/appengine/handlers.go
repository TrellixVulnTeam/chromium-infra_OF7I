// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"net/http"

	"go.chromium.org/luci/appengine/gaemiddleware/standard"
	"go.chromium.org/luci/common/data/rand/mathrand"
	"go.chromium.org/luci/server/auth"
	"go.chromium.org/luci/server/auth/openid"
	"go.chromium.org/luci/server/router"
	"google.golang.org/appengine"

	"infra/appengine/cros/lab_inventory/app/config"
	"infra/appengine/cros/lab_inventory/app/cron"
	"infra/appengine/cros/lab_inventory/app/frontend"
	"infra/appengine/cros/lab_inventory/app/pubsub"
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

	// Add authenticator for handling JWT tokens. This is required to
	// authenticate PubSub push responses sent as HTTP POST requests. See
	// https://cloud.google.com/pubsub/docs/push?hl=en#authentication_and_authorization
	openIDCheck := auth.Authenticator{
		Methods: []auth.Method{
			&openid.GoogleIDTokenAuthMethod{
				AudienceCheck: openid.AudienceMatchesHost,
			},
		},
	}

	// Add authenticator to middleware chain at the end, This is because
	// not all POST requests use JWT for authentication. Including this
	// in all the handlers will result in failures
	mwBase = mwBase.Extend(openIDCheck.GetMiddleware())
	pubsub.InstallHandlers(r, mwBase)

	http.DefaultServeMux.Handle("/", r)

	appengine.Main()
}
