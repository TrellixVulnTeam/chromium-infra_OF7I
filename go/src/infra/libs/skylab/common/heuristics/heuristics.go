// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package heuristics

import "strings"

// LooksLikeLabstation returns whether a hostname or botID appears to be a labstation or not.
// This function exists so that we always use the same heuristic everywhere when identifying labstations.
func LooksLikeLabstation(hostname string) bool {
	return strings.Contains(hostname, "labstation")
}
