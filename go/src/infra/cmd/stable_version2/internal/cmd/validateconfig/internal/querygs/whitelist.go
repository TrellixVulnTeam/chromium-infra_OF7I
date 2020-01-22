// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// This portion of the querygs package contains a list of boards and models
// that are known to be absent from metadata.json files. For instance,
// labstations are not present in any metadata.json file.

package querygs

var missingBoardWhitelist map[string]bool = stringSliceToStringSet([]string{})

var failedToLookupWhiteList map[string]bool = stringSliceToStringSet([]string{})

func stringSliceToStringSet(input []string) map[string]bool {
	var out = make(map[string]bool, len(input))
	for _, item := range input {
		out[item] = true
	}
	return out
}
