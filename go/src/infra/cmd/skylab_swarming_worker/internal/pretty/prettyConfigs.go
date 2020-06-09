// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// PLEASE DO NOT USE IT OUTSIDE SSW

package pretty

import (
	prettyTest "github.com/kylelemons/godebug/pretty"
)

// PrettyConfig default config to run pretty test for recursive validation
var PrettyConfig = &prettyTest.Config{
	TrackCycles: true,
}
