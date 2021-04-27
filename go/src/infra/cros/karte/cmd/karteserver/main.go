// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

// This is the entrypoint for the Karte service in production and dev.
// Control is transferred here, inside the Docker container, when the
// application starts.

import (
	"context"
	"os"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/data/rand/mathrand"
	"go.chromium.org/luci/common/logging/gologger"
)

// newApplication returns an application object for the karte entrypoint that's
// intended to be run inside the container.
func newApplication() *cli.Application {
	return &cli.Application{
		Name:  "karteserver",
		Title: "The karte server application",
		Context: func(ctx context.Context) context.Context {
			return gologger.StdConfig.Use(ctx)
		},
		Commands: []*subcommands.Command{},
	}
}

func main() {
	mathrand.SeedRandomly()
	os.Exit(subcommands.Run(newApplication(), nil))
}
