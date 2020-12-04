// Copyright 2018 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package site contains site local constants for the skylab tool.
package site

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"go.chromium.org/luci/auth"
	buildbucket_pb "go.chromium.org/luci/buildbucket/proto"
	"go.chromium.org/luci/common/api/gitiles"
	"go.chromium.org/luci/grpc/prpc"
)

// Environment contains environment specific values.
type Environment struct {
	LUCIProject      string
	SwarmingService  string
	LogDogHost       string
	InventoryService string
	AdminService     string
	QueenService     string
	// QueenDroneHostname is only used by queen-push-duts.
	QueenDroneHostname string
	ServiceAccount     string

	// Buildbucket-specific values.
	CTPBuilderInfo       BuildbucketBuilderInfo
	DUTLeaserBuilderInfo BuildbucketBuilderInfo

	// UFS-specific values
	UFSService string
}

// BuildbucketBuilderInfo contains information for initializing a
// Buildbucket client that talks to a specific builder.
type BuildbucketBuilderInfo struct {
	Host      string
	BuilderID *buildbucket_pb.BuilderID
}

// Wrapped returns the environment wrapped in a helper type to satisfy
// the worker.Environment interface and swarming.Environment interface.
func (e Environment) Wrapped() EnvWrapper {
	return EnvWrapper{e: e}
}

// EnvWrapper wraps Environment to satisfy the worker.Environment
// interface and swarming.Environment interface.
type EnvWrapper struct {
	e Environment
}

// LUCIProject implements worker.Environment.
func (e EnvWrapper) LUCIProject() string {
	return e.e.LUCIProject
}

// LogDogHost implements worker.Environment.
func (e EnvWrapper) LogDogHost() string {
	return e.e.LogDogHost
}

// GenerateLogPrefix implements worker.Environment.
func (e EnvWrapper) GenerateLogPrefix() string {
	return "skylab/" + uuid.New().String()
}

// Prod is the environment for prod.
var Prod = Environment{
	LUCIProject:        "chromeos",
	SwarmingService:    "https://chromeos-swarming.appspot.com/",
	LogDogHost:         "luci-logdog.appspot.com",
	InventoryService:   "cros-lab-inventory.appspot.com",
	AdminService:       "chromeos-skylab-bot-fleet.appspot.com",
	QueenService:       "drone-queen-prod.appspot.com",
	QueenDroneHostname: "drone-queen-ENVIRONMENT_PROD",
	ServiceAccount:     "skylab-admin-task@chromeos-service-accounts.iam.gserviceaccount.com",

	CTPBuilderInfo: BuildbucketBuilderInfo{
		Host: "cr-buildbucket.appspot.com",
		BuilderID: &buildbucket_pb.BuilderID{
			Project: "chromeos",
			Bucket:  "testplatform",
			Builder: "cros_test_platform",
		},
	},
	DUTLeaserBuilderInfo: BuildbucketBuilderInfo{
		Host: "cr-buildbucket.appspot.com",
		BuilderID: &buildbucket_pb.BuilderID{
			Project: "chromeos",
			Bucket:  "test_runner",
			Builder: "dut_leaser",
		},
	},

	UFSService: "ufs.api.cr.dev",
}

// Dev is the environment for dev.
var Dev = Environment{
	LUCIProject:        "chromeos",
	SwarmingService:    "https://chromium-swarm-dev.appspot.com/",
	LogDogHost:         "luci-logdog-dev.appspot.com",
	InventoryService:   "cros-lab-inventory-dev.appspot.com",
	AdminService:       "skylab-staging-bot-fleet.appspot.com",
	QueenService:       "drone-queen-dev.appspot.com",
	QueenDroneHostname: "drone-queen-ENVIRONMENT_STAGING",
	ServiceAccount:     "skylab-admin-task@chromeos-service-accounts-dev.iam.gserviceaccount.com",

	CTPBuilderInfo: BuildbucketBuilderInfo{
		Host: "cr-buildbucket.appspot.com",
		BuilderID: &buildbucket_pb.BuilderID{
			Project: "chromeos",
			Bucket:  "testplatform",
			Builder: "cros_test_platform-dev",
		},
	},
	DUTLeaserBuilderInfo: BuildbucketBuilderInfo{
		Host: "cr-buildbucket.appspot.com",
		BuilderID: &buildbucket_pb.BuilderID{
			Project: "chromeos",
			Bucket:  "test_runner",
			Builder: "dut_leaser",
		},
	},

	UFSService: "staging.ufs.api.cr.dev",
}

// DefaultAuthOptions is an auth.Options struct prefilled with chrome-infra
// defaults.
var DefaultAuthOptions = auth.Options{
	// Note that ClientSecret is not really a secret since it's hardcoded into
	// the source code (and binaries). It's totally fine, as long as it's callback
	// URI is configured to be 'localhost'. If someone decides to reuse such
	// ClientSecret they have to run something on user's local machine anyway
	// to get the refresh_token.
	ClientID:     "446450136466-2hr92jrq8e6i4tnsa56b52vacp7t3936.apps.googleusercontent.com",
	ClientSecret: "uBfbay2KCy9t4QveJ-dOqHtp",
	SecretsDir:   SecretsDir(),
	Scopes:       []string{auth.OAuthScopeEmail, gitiles.OAuthScope},
}

// VersionNumber is the version number for the tool. It follows the Semantic
// Versioning Specification (http://semver.org) and the format is:
// "MAJOR.MINOR.0+BUILD_TIME".
// We can ignore the PATCH part (i.e. it's always 0) to make the maintenance
// work easier.
// We can also print out the build time (e.g. 20060102150405) as the METADATA
// when show version to users.
const VersionNumber = "2.1.2"

// DefaultPRPCOptions is used for PRPC clients.  If it is nil, the
// default value is used.  See prpc.Options for details.
//
// This is provided so it can be overridden for testing.
var DefaultPRPCOptions = prpcOptionWithUserAgent(fmt.Sprintf("skylab/%s", VersionNumber))

// UFSPRPCOptions is used for UFS PRPC clients.
var UFSPRPCOptions = prpcOptionWithUserAgent("skylab/6.0.0")

// SecretsDir returns an absolute path to a directory (in $HOME) to keep secret
// files in (e.g. OAuth refresh tokens) or an empty string if $HOME can't be
// determined (happens in some degenerate cases, it just disables auth token
// cache).
func SecretsDir() string {
	configDir := os.Getenv("XDG_CACHE_HOME")
	if configDir == "" {
		configDir = filepath.Join(os.Getenv("HOME"), ".cache")
	}
	return filepath.Join(configDir, "skylab", "auth")
}

// prpcOptionWithUserAgent create prpc option with custom UserAgent.
//
// DefaultOptions provides Retry ability in case we have issue with service.
func prpcOptionWithUserAgent(userAgent string) *prpc.Options {
	options := *prpc.DefaultOptions()
	options.UserAgent = userAgent
	return &options
}
