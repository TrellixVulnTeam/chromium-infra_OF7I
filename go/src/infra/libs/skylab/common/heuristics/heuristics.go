// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package heuristics

import (
	"os"
	"regexp"
	"strings"

	"go.chromium.org/luci/common/errors"
)

// LooksLikeSatlabRemoteAccessContainer determines whether the container we are running on looks like
// a satlab remote access container.
func LooksLikeSatlabRemoteAccessContainer() (bool, error) {
	// TODO(gregorynisbet):
	// Do not use "errors.Is" inside the body of this function until CrOSSkylabAdmin
	// and other code that transitively depends on this module are migrated to go113 or
	// higher.
	_, err := os.Stat("/usr/local/bin/get_host_identifier")
	if err != nil {
		if strings.Contains(err.Error(), "no such file or directory") {
			return false, nil
		}
		return false, errors.Annotate(err, "looks like satlab remote access container").Err()
	}
	return true, nil
}

// LooksLikeSatlabDevice returns whether a hostname or botID appears to be a satlab-managed device.
// This function exists so that we use the same heuristic everywhere when identifying satlab devices.
func LooksLikeSatlabDevice(hostname string) bool {
	h := strings.TrimPrefix(hostname, "crossk-")
	return strings.HasPrefix(h, "satlab")
}

// LooksLikeLabstation returns whether a hostname or botID appears to be a labstation or not.
// This function exists so that we always use the same heuristic everywhere when identifying labstations.
func LooksLikeLabstation(hostname string) bool {
	return strings.Contains(hostname, "labstation")
}

// LooksLikeHeader heuristically determines whether a CSV line looks like
// a CSV header for the MCSV format.
func LooksLikeHeader(rec []string) bool {
	if len(rec) == 0 {
		return false
	}
	return strings.EqualFold(rec[0], "name")
}

// LooksLikeCrosskBotName checks whether the name in question begins with "crossk-".
// This prefix reliably identifies a CrOSSkylabAdmin swarming bot (and distinguishes it from a DUT hostname).
func LooksLikeCrosskBotName(name string) bool {
	return strings.HasPrefix(name, "crossk-")
}

// NormalizeBotNameToDeviceName takes a bot name or a DUT name and normalizes it to a DUT name.
// If the input is not a bot name or DUT name, then the results are undefined.
func NormalizeBotNameToDeviceName(name string) string {
	return strings.TrimPrefix(name, "crossk-")
}

// looksLikeValidPool heuristically checks a string to see if it looks like a valid pool.
// A heuristically valid pool name contains only a-z, A-Z, 0-9, -, and _ .
// A pool name cannot begin with - and 0-9 .
var LooksLikeValidPool = regexp.MustCompile(`\A[A-Za-z_][-A-Za-z0-9_]*\z`).MatchString

// NormalizeTextualData lowercases data and removes leading and trailing whitespace.
func NormalizeTextualData(data string) string {
	return strings.ToLower(strings.TrimSpace(data))
}
