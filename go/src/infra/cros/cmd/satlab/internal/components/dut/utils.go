// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dut

import (
	"fmt"
	"os"
	"strings"

	"infra/cros/cmd/satlab/internal/commands"
	"infra/cros/cmd/satlab/internal/site"

	"go.chromium.org/luci/common/errors"
)

// Flagmap is a map from the name of a flag to its value(s).
type flagmap = map[string][]string

// GetDockerHostBoxIdentifier gets the identifier for the satlab DHB, either from the command line, or
// by running a command inside the current container if no flag was given on the command line.
//
// Note that this function always returns the satlab ID in lowercase.
func getDockerHostBoxIdentifier(common site.CommonFlags) (string, error) {
	// Use the string provided in the common flags by default.
	if common.SatlabID != "" {
		return strings.ToLower(common.SatlabID), nil
	}

	dockerHostBoxIdentifier, err := commands.GetDockerHostBoxIdentifier()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to determine -satlab prefix, use %s to pass explicitly\n", common.SatlabID)
		return "", errors.Annotate(err, "get docker host box").Err()
	}

	return dockerHostBoxIdentifier, nil
}
