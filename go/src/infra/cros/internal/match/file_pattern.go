// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package match

import (
	"github.com/bmatcuk/doublestar"
	"go.chromium.org/chromiumos/infra/proto/go/testplans"
)

// FilePatternMatches returns true if sourcePath matches filePattern.
//
// See comments on the FilePattern message for detailed description of each
// field.
func FilePatternMatches(filePattern *testplans.FilePattern, sourcePath string) (bool, error) {
	// If sourcePath matches any ExcludePattern, return false.
	for _, pattern := range filePattern.GetExcludePatterns() {
		match, err := doublestar.Match(pattern, sourcePath)
		if err != nil {
			return false, err
		}

		if match {
			return false, nil
		}
	}

	// Otherwise, return whether sourcePath matches Pattern.
	return doublestar.Match(filePattern.GetPattern(), sourcePath)
}
