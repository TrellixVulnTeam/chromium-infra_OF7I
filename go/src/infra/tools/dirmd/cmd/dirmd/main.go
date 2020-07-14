// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"os"

	"go.chromium.org/luci/common/data/rand/mathrand"

	"infra/tools/dirmd/cli"
)

func main() {
	mathrand.SeedRandomly()
	os.Exit(cli.Main(os.Args[1:]))
}
