// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package site contains site local constants for the shivas
package site

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/grpc/prpc"
	"go.chromium.org/luci/hardcoded/chromeinfra"

	"go.chromium.org/luci/common/gcloud/gs"
)

// Environment contains environment specific values.
type Environment struct {
	InventoryService    string
	UnifiedFleetService string
}

// Prod is the environment for prod.
var Prod = Environment{
	InventoryService: "cros-lab-inventory.appspot.com",
	//TODO(eshwarn) : Change it to prod during release
	UnifiedFleetService: "staging.ufs.api.cr.dev",
}

// Dev is the environment for dev.
var Dev = Environment{
	InventoryService:    "cros-lab-inventory-dev.appspot.com",
	UnifiedFleetService: "staging.ufs.api.cr.dev",
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
	Scopes:     append(gs.ReadWriteScopes, auth.OAuthScopeEmail),
	SecretsDir: SecretsDir(),
})

// SecretsDir customizes the location for auth-related secrets.
func SecretsDir() string {
	configDir := os.Getenv("XDG_CACHE_HOME")
	if configDir == "" {
		configDir = filepath.Join(os.Getenv("HOME"), ".cache")
	}
	return filepath.Join(configDir, "shivas", "auth")
}

// VersionNumber is the version number for the tool. It follows the Semantic
// Versioning Specification (http://semver.org) and the format is:
// "MAJOR.MINOR.0+BUILD_TIME".
// We can ignore the PATCH part (i.e. it's always 0) to make the maintenance
// work easier.
// We can also print out the build time (e.g. 20060102150405) as the METADATA
// when show version to users.
var VersionNumber = fmt.Sprintf("%d.%d.%d", Major, Minor, Patch)

// Major is the Major version number
const Major = 2

// Minor is the Minor version number
const Minor = 0

// Patch is the PAtch version number
const Patch = 0

// DefaultPRPCOptions is used for PRPC clients.  If it is nil, the
// default value is used.  See prpc.Options for details.
//
// This is provided so it can be overridden for testing.
var DefaultPRPCOptions = &prpc.Options{
	UserAgent: fmt.Sprintf("shivas/%s", VersionNumber),
}

// CipdInstalledPath is the installed path for shivas package.
var CipdInstalledPath = "chromiumos/infra/shivas/"
