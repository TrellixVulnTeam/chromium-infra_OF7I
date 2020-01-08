// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmd

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
)

// ProgramName is the name of the current executable.
const ProgramName = "stable_version2"

// OmahaStatusFile is the name of the file with stable version information in it.
const OmahaStatusFile = "omaha_status.json"

// OmahaGSPath is the full google-storage path to the omaha status file
const OmahaGSPath = "gs://chromeos-build-release-console/omaha_status.json"

// GerritHost is the Gerrit host that manages the repo with the stable version config file.
const GerritHost = "chrome-internal-review.googlesource.com"

// GitilesHost is the host of the gitiles service.
const GitilesHost = "chrome-internal.googlesource.com"

// Project is the path to the repo root within Gitiles.
const Project = "chromeos/infra/config"

// Branch is the name of the branch that we're modifying.
const Branch = "master"

// StableVersionConfigPath is the path to the stable version config file relative to the repo root.
const StableVersionConfigPath = "lab_platform/stable_version_data/stable_versions.cfg"

// PrintError writes an error to stderr with the correct program name.
func PrintError(w io.Writer, err error) {
	fmt.Fprintf(w, "%s: %s\n", ProgramName, err)
}

// SetupLogging sets the log level
func SetupLogging(ctx context.Context) context.Context {
	return logging.SetLevel(ctx, logging.Debug)
}

// NewAuthenticatedTransport creates a new authenticated transport
func NewAuthenticatedTransport(ctx context.Context, f *authcli.Flags) (http.RoundTripper, error) {
	o, err := f.Options()
	if err != nil {
		return nil, errors.Annotate(err, "create authenticated transport").Err()
	}
	a := auth.NewAuthenticator(ctx, auth.SilentLogin, o)
	return a.Transport()
}

// NewHTTPClient creates a new HTTP Client with the given authentication options.
func NewHTTPClient(ctx context.Context, f *authcli.Flags) (*http.Client, error) {
	o, err := f.Options()
	if err != nil {
		return nil, errors.Annotate(err, "failed to get auth options").Err()
	}
	a := auth.NewAuthenticator(ctx, auth.OptionalLogin, o)
	c, err := a.Client()
	if err != nil {
		return nil, errors.Annotate(err, "failed to create HTTP client").Err()
	}
	return c, nil
}
