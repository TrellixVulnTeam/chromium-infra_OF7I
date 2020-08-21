// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package utils

import (
	"os"
	"strings"
)

func noPrompt() bool {
	return strings.ToLower(os.Getenv("SHIVAS_NO_PROMPT")) == "true" || strings.ToLower(os.Getenv("SHIVAS_NO_PROMPT")) == "1"
}

// FullMode checks if full mode is enabled
func FullMode(full bool) bool {
	return full || (strings.ToLower(os.Getenv("SHIVAS_FULL_MODE")) == "true") || strings.ToLower(os.Getenv("SHIVAS_FULL_MODE")) == "1"
}
