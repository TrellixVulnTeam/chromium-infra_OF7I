// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Satlab is a wrapper around shivas.

package main

import (
	"fmt"
	"os"

	satlab "infra/cros/cmd/satlab/internal"
)

// Main runs the satlab command and exits abnormally if ./satlab finished with an error.
func main() {
	if err := satlab.Entrypoint(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		os.Exit(1)
	}
}
