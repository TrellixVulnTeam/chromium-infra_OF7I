// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package utils

import (
	"os"
	"strings"
)

func noPrompt() bool {
	return strings.ToLower(os.Getenv("NO_PROMPT")) == "true"
}
