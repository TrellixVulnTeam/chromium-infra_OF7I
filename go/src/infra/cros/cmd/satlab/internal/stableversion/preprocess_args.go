// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package stableversion

import (
	"fmt"
	"infra/cros/cmd/satlab/internal/commands"
	"infra/cros/cmd/satlab/internal/site"
	"infra/libs/skylab/common/heuristics"
	"strings"

	"go.chromium.org/luci/common/errors"
)

type getDHBID = func() (string, error)

type isRemoteAccess = func() (bool, error)

// preprocessHostname takes a hostname, commonly something like "host1" and adds
// a prefix like "satlab-$SATLAB_ID" yielding "satlab-$SATLAB_ID-host1".
//
// If the satlabID is set in the common flags then it was set in the command line
// and the user intended to override it explicitly.
//
// If the argument that we're given already looks like a hostname, then the user
// probably intended to modify a specific device, so we do not preprocess the input.
func preprocessHostname(common site.CommonFlags, hostname string, getDHBID getDHBID, isRemoteAccess isRemoteAccess) (string, error) {
	// By default, these values really should be the real versions "GetDockerHostBoxIdentifier" and "LooksLikeSatlabRemoteAccessContainer".
	// Set them here so that we never accidentally call a nil function by mistake on the production path.
	if getDHBID == nil {
		getDHBID = commands.GetDockerHostBoxIdentifier
	}
	if isRemoteAccess == nil {
		isRemoteAccess = heuristics.LooksLikeSatlabRemoteAccessContainer
	}
	// If the hostname is empty, then the user did not provide it.
	// Try to provide a direct, helpful error message.
	if hostname == "" {
		return "", errors.Reason("hostname cannot be empty").Err()
	}

	// An explicit satlabID was provided. The user may or may not be in the satlab-remote-access
	// container. Providing this argument means that the user intended to work on a specific satlab
	// device by its ID so help them out.
	if common.SatlabID != "" {
		if heuristics.LooksLikeSatlabDevice(hostname) {
			return "", errors.Reason("explicit satlab ID provided %q for hostname that already has satlab prefix %q", common.SatlabID, hostname).Err()
		}
		satlabID := strings.ToLower(common.SatlabID)
		return fmt.Sprintf("satlab-%s-%s", satlabID, hostname), nil
	}

	// If no satlab prefix was provided on the command line, but the hostname has a satlab prefix anyway,
	// then we also can't assume that we're in a satlab-remote-access container, similar to the common.SatlabID case above.
	if heuristics.LooksLikeSatlabDevice(hostname) {
		return hostname, nil
	}

	// Finally, we fall back to the case that needs a satlab-remote-access container.
	// In order to give better error messages, we look first at whether we are a satlab
	// remote access container or not.
	//
	// If we encounter an error here, then all subsequent steps to get the satlabID are going to fail anyway.
	ok, err := isRemoteAccess()
	if err != nil {
		return "", errors.Annotate(err, "preprocess hostname").Err()
	}
	if !ok {
		return "", errors.Reason("cannot process host without satlab prefix outside satlab-remote-access").Err()
	}

	// We're probably in the satlab remote access container. Get the satlab ID.
	satlabID, err := getDHBID()
	if err != nil {
		// This really shouldn't happen, but is technically possible if, for example, get_host_identifier is not
		// executable for some reason.
		return "", errors.Annotate(err, "preprocess hostname").Err()
	}
	return fmt.Sprintf("satlab-%s-%s", satlabID, hostname), nil
}
