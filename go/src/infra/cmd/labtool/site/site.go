// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package site contains site local constants for the skylab tool
package site

import (
	"flag"

	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/grpc/prpc"
	"go.chromium.org/luci/hardcoded/chromeinfra"
)

// Environment contains environment specific values.
type Environment struct {
	InventoryService string
}

// Prod is the environment for prod.
var Prod = Environment{
	InventoryService: "cros-lab-inventory.appspot.com",
}

// Dev is the environment for dev.
var Dev = Environment{
	InventoryService: "cros-lab-inventory-dev.appspot.com",
}

// EnvFlags controls selection of the environment: either prod (default) or dev.
type EnvFlags struct {
	dev bool
}

// Register sets up the -dev argument.
func (f *EnvFlags) Register(fl *flag.FlagSet) {
	fl.BoolVar(&f.dev, "dev", false, "Run in dev environment.")
}

// Env returns the environment, either dev or prod.
func (f EnvFlags) Env() Environment {
	if f.dev {
		return Dev
	}
	return Prod
}

// DefaultAuthOptions is an auth.Options struct prefilled with chrome-infra
// defaults.
var DefaultAuthOptions = chromeinfra.SetDefaultAuthOptions(auth.Options{
	Scopes: []string{auth.OAuthScopeEmail},
})

// DefaultPRPCOptions is used for PRPC clients.  If it is nil, the
// default value is used.  See prpc.Options for details.
//
// This is provided so it can be overridden for testing.
var DefaultPRPCOptions *prpc.Options
