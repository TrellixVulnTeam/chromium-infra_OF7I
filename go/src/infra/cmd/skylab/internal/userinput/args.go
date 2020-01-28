// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package userinput

import (
	"strings"
)

// ValidBug returns true if the given bug string is acceptably formatted.
func ValidBug(bug string) bool {
	if strings.HasPrefix(bug, "b/") {
		return true
	}
	if strings.HasPrefix(bug, "crbug.com/") {
		return true
	}
	return false
}
