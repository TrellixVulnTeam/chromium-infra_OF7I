// Copyright 2021 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

// Package plugin is the entry point for code inside the plugin process.
package plugin

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"go.chromium.org/luci/common/data/rand/mathrand"

	"infra/tools/migrator"
	"infra/tools/migrator/internal/plugsupport"
)

// Main is the main entry point for the plugin called from the plugin code.
//
// It assumes the plugin binary was called by the `migrator` CLI tool.
func Main(factory migrator.InstantiateAPI) {
	if len(os.Args) != 2 {
		fatal("expecting 1 positional argument")
	}
	blob, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		fatal("failed to read the command file")
	}
	var cmd plugsupport.Command
	if err := json.Unmarshal(blob, &cmd); err != nil {
		fatal("failed to unmarshal the command file")
	}
	mathrand.SeedRandomly()
	os.Exit(plugsupport.Handle(factory, cmd))
}

func fatal(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}
