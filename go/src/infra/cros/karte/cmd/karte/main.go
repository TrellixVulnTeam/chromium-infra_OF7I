// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

// This is the karte command line client.

import (
	"context"
	"os"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/data/rand/mathrand"
	"go.chromium.org/luci/common/logging/gologger"

	kartecli "infra/cros/karte/cli"
)

// newApplication returns an application object for the karte command line client.
func newApplication() *cli.Application {
	return &cli.Application{
		Name:  "karte",
		Title: "The karte client application",
		Context: func(ctx context.Context) context.Context {
			return gologger.StdConfig.Use(ctx)
		},
		Commands: []*subcommands.Command{
			subcommands.Section("main API"),
			kartecli.Backfill,
			kartecli.CheckServer,
			kartecli.CreateAction,
			kartecli.ListActions,
			kartecli.ListSwarmingTasks,
			kartecli.PersistSingleAction,
			kartecli.PersistActionRange,
			kartecli.ExampleCreateUpdateAction,
			subcommands.Section("cron"),
			kartecli.CronPersistToBigquery,
		},
	}
}

func main() {
	mathrand.SeedRandomly()
	os.Exit(subcommands.Run(newApplication(), nil))
}
