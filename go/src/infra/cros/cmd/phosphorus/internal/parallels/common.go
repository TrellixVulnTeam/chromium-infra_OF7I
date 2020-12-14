// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package parallels contains commands used in the build_parallels_image
// recipe.
package parallels

import (
	bpipb "go.chromium.org/chromiumos/infra/proto/go/uprev/build_parallels_image"
)

// validateConfig returns the list of missing required config
// arguments.
func validateConfig(c *bpipb.Config) []string {
	var missingArgs []string

	if c.GetCrosUfsService() == "" {
		missingArgs = append(missingArgs, "UFS service")
	}

	return missingArgs
}
