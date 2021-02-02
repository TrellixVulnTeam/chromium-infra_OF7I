// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package rts

// Affectedness is how much a test is affected by the code change.
// The zero value means a test is very affected.
type Affectedness struct {
	// Distance is a non-negative number, where 0.0 means the code change is
	// extremely likely to affect the test, and +inf means extremely unlikely.
	// If a test's distance is less or equal than a given MaxDistance threshold,
	// then the test is selected.
	// A selection strategy doesn't have to use +inf as the upper boundary if the
	// threshold uses the same scale.
	Distance float64
}
