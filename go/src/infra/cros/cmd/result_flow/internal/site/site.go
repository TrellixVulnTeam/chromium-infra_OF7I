// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package site contains site local constants for the Result Flow.
package site

import (
	"os"
	"path/filepath"

	"go.chromium.org/luci/auth"
)

// DefaultDeadlineSeconds is the default command deadline in seconds.
const DefaultDeadlineSeconds = 1800

const (
	// CTPBatchSize is the size of one single Buildbucket batch request to fetch
	// CTP builds. Increase the batch size below with caution. It may cause
	// Buildbucket to return 500 errors, due to the response size.
	CTPBatchSize int32 = 3
	// TestRunnerBatchSize is the size of one single Buildbucket batch request
	// to fetch test runner builds.
	TestRunnerBatchSize int32 = 50
)

const (
	authScopeBigquery = "https://www.googleapis.com/auth/bigquery"
	authScopePubsub   = "https://www.googleapis.com/auth/pubsub"
)

// DefaultAuthOptions is an auth.Options struct prefilled with chrome-infra
// defaults. The credentials here are used for local test.
var DefaultAuthOptions = auth.Options{
	// Note that ClientSecret is not really a secret since it's hardcoded into
	// the source code (and binaries). It's totally fine, as long as it's callback
	// URI is configured to be 'localhost'. If someone decides to reuse such
	// ClientSecret they have to run something on user's local machine anyway
	// to get the refresh_token.
	ClientID:     "446450136466-2hr92jrq8e6i4tnsa56b52vacp7t3936.apps.googleusercontent.com",
	ClientSecret: "uBfbay2KCy9t4QveJ-dOqHtp",
	SecretsDir:   SecretsDir(),
	Scopes:       []string{auth.OAuthScopeEmail, authScopeBigquery, authScopePubsub},
}

// SecretsDir returns an absolute path to a directory (in $HOME) to keep secret
// files in (e.g. OAuth refresh tokens) or an empty string if $HOME can't be
// determined (happens in some degenerate cases, it just disables auth token
// cache).
func SecretsDir() string {
	configDir := os.Getenv("XDG_CACHE_HOME")
	if configDir == "" {
		configDir = filepath.Join(os.Getenv("HOME"), ".cache")
	}
	return filepath.Join(configDir, "result_flow", "auth")
}
