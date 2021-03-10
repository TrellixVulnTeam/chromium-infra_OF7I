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
	"strings"

	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/gcloud/gs"
	"go.chromium.org/luci/grpc/prpc"
	"go.chromium.org/luci/hardcoded/chromeinfra"

	ufsUtil "infra/unifiedfleet/app/util"
)

// Environment contains environment specific values.
type Environment struct {
	AdminService           string
	InventoryService       string
	UnifiedFleetService    string
	SwarmingService        string
	SwarmingServiceAccount string
	LogdogService          string
	QueenService           string
}

// Prod is the environment for prod.
var Prod = Environment{
	// TODO(gregorynisbet): remove once stable version moves to UFS.
	AdminService:     "chromeos-skylab-bot-fleet.appspot.com",
	InventoryService: "cros-lab-inventory.appspot.com",
	//TODO(eshwarn) : Change it to prod during release
	UnifiedFleetService: "ufs.api.cr.dev",
	SwarmingService:     "https://chromeos-swarming.appspot.com/",
	// TODO(crbug/1128496): remove when fixed
	SwarmingServiceAccount: "skylab-admin-task@chromeos-service-accounts.iam.gserviceaccount.com",
	LogdogService:          "luci-logdog.appspot.com",
	QueenService:           "drone-queen-prod.appspot.com",
}

// Dev is the environment for dev.
var Dev = Environment{
	// TODO(gregorynisbet): remove once stable version moves to UFS.
	AdminService:        "skylab-staging-bot-fleet.appspot.com",
	InventoryService:    "cros-lab-inventory-dev.appspot.com",
	UnifiedFleetService: "staging.ufs.api.cr.dev",
	SwarmingService:     "https://chromium-swarm-dev.appspot.com/",
	// TODO(crbug/1128496): remove when fixed
	SwarmingServiceAccount: "skylab-admin-task@chromeos-service-accounts-dev.iam.gserviceaccount.com",
	LogdogService:          "luci-logdog-dev.appspot.com",
	QueenService:           "drone-queen-dev.appspot.com",
}

// CommonFlags controls some commonly-used CLI flags.
type CommonFlags struct {
	verbose bool
}

// Register sets up the common flags.
func (f *CommonFlags) Register(fl *flag.FlagSet) {
	fl.BoolVar(&f.verbose, "verbose", false, "log more details")
}

// Verbose returns if the command is set to verbose mode.
func (f *CommonFlags) Verbose() bool {
	return f.verbose
}

// OutputFlags controls output-related CLI flags.
type OutputFlags struct {
	json   bool
	tsv    bool
	full   bool
	noemit bool
}

// Register sets up the output flags.
func (f *OutputFlags) Register(fl *flag.FlagSet) {
	fl.BoolVar(&f.json, "json", false, "log output in json format")
	fl.BoolVar(&f.tsv, "tsv", false, "log output in tsv format (without title)")
	fl.BoolVar(&f.full, "full", false, "log full output in specified format, only works for GET command(latency is high). Users can also set os env SHIVAS_FULL_MODE to enable this.")
	fl.BoolVar(&f.noemit, "noemit", false, "specifies NOT to emit/print unpopulated fields in json format. Users can also set os env SHIVAS_NO_JSON_EMIT to enable this.")
}

// JSON returns if the output is logged in json format
func (f *OutputFlags) JSON() bool {
	return f.json
}

// Tsv returns if the output is logged in tsv format (without title)
func (f *OutputFlags) Tsv() bool {
	return f.tsv
}

// Full returns if the full format of output is logged in tsv format (without title)
func (f *OutputFlags) Full() bool {
	return f.full
}

// NoEmit returns if output json should NOT print/emit unpopulated fields
func (f *OutputFlags) NoEmit() bool {
	return f.noemit
}

// EnvFlags controls selection of the environment: either prod (default) or dev.
type EnvFlags struct {
	dev       bool
	namespace string
}

// Register sets up the -dev argument.
func (f *EnvFlags) Register(fl *flag.FlagSet) {
	fl.BoolVar(&f.dev, "dev", false, "Run in dev environment.")
	fl.StringVar(&f.namespace, "namespace", "", fmt.Sprintf("namespace where data resides. Users can also set os env SHIVAS_NAMESPACE. Valid namespaces: [%s]", strings.Join(ufsUtil.ValidClientNamespaceStr(), ", ")))
}

// Env returns the environment, either dev or prod.
func (f EnvFlags) Env() Environment {
	if f.dev {
		return Dev
	}
	return Prod
}

// Namespace returns the namespace
func (f EnvFlags) Namespace() (string, error) {
	ns := strings.ToLower(f.namespace)
	if ns == "" {
		ns = strings.ToLower(os.Getenv("SHIVAS_NAMESPACE"))
	}
	if ns != "" && ufsUtil.IsClientNamespace(ns) {
		return ns, nil
	}
	if ns == "" {
		return ns, errors.New(fmt.Sprintf("namespace is a required field. Users can also set os env SHIVAS_NAMESPACE. Valid namespaces: [%s]", strings.Join(ufsUtil.ValidClientNamespaceStr(), ", ")))
	}
	return ns, errors.New(fmt.Sprintf("namespace %s is invalid. Users can also set os env SHIVAS_NAMESPACE. Valid namespaces: [%s]", ns, strings.Join(ufsUtil.ValidClientNamespaceStr(), ", ")))
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
const Major = 7

// Minor is the Minor version number
const Minor = 1

// Patch is the PAtch version number
const Patch = 0

// DefaultPRPCOptions is used for PRPC clients.  If it is nil, the
// default value is used.  See prpc.Options for details.
//
// This is provided so it can be overridden for testing.
var DefaultPRPCOptions = prpcOptionWithUserAgent(fmt.Sprintf("shivas/%s", VersionNumber))

// CipdInstalledPath is the installed path for shivas package.
var CipdInstalledPath = "infra/shivas/"

// prpcOptionWithUserAgent create prpc option with custom UserAgent.
//
// DefaultOptions provides Retry ability in case we have issue with service.
func prpcOptionWithUserAgent(userAgent string) *prpc.Options {
	options := *prpc.DefaultOptions()
	options.UserAgent = userAgent
	return &options
}
