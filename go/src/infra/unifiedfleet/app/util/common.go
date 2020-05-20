// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package util

// Min returns the smaller integer of the two inputs.
func Min(a, b int) int {
	if a <= b {
		return a
	}
	return b
}
