// Copyright 2032 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package site

import (
	"os"
	"path/filepath"

	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/common/api/gitiles"
)

const progName = "karte"

// DefaultAuthOptions is an auth.Options struct prefilled with chrome-infra
// defaults.
var DefaultAuthOptions = auth.Options{
	// TODO(gregorynisbet): replace with real credential
	ClientID:     "",
	ClientSecret: "",
	SecretsDir:   SecretsDir(),
	Scopes:       []string{auth.OAuthScopeEmail, gitiles.OAuthScope},
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
	return filepath.Join(configDir, progName, "auth")
}
