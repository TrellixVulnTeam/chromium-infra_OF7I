// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package heuristics

import (
	"regexp"
	"strings"
)

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

// looksLikeValidPool heuristically checks a string to see if it looks like a valid pool.
// A heuristically valid pool name contains only a-z, A-Z, 0-9, -, and _ .
// A pool name cannot begin with - and 0-9 .
var LooksLikeValidPool = regexp.MustCompile(`\A[A-Za-z_][-A-za-z0-9_]*\z`).MatchString
