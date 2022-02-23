// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package clustering

import "strconv"

// EscapeToGraphical escapes the input so that it only contains graphic unicode characters.
// Use on test names and failure reasons before presenting to any UI context.
func EscapeToGraphical(value string) string {
	quotedEscaped := strconv.QuoteToGraphic(value)
	// Remove starting end ending double-quotes.
	return quotedEscaped[1 : len(quotedEscaped)-1]
}
