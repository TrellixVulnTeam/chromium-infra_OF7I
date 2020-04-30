// Copyright 2018 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Command lucifer runs tests and admin tasks.
//
// The GOOGLE_APPLICATION_CREDENTIALS environment variable specifies
// GCP service account credentials for metrics.
//
// See infra/cros/cmd/lucifer/internal/event for status events are reported.
//
// See infra/cros/cmd/lucifer/internal/abortsock for how aborting works.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/google/subcommands"
)

func main() {
	log.SetPrefix(fmt.Sprintf("%s: ", filepath.Base(os.Args[0])))
	log.Printf("Running with args: %s", os.Args)
	log.Printf("GOOGLE_APPLICATION_CREDENTIALS is %s",
		os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"))
	subcommands.Register(subcommands.HelpCommand(), "")
	subcommands.Register(subcommands.FlagsCommand(), "")
	subcommands.Register(subcommands.CommandsCommand(), "")
	subcommands.Register(&adminTaskCmd{}, "")
	subcommands.Register(&deployTaskCmd{}, "")
	subcommands.Register(&testCmd{}, "")
	subcommands.Register(&prejobCmd{}, "")
	subcommands.Register(&runTestCmd{}, "")
	subcommands.Register(&tkoParseCmd{}, "")
	flag.Parse()
	ctx := context.Background()
	ret := int(subcommands.Execute(ctx))
	log.Printf("Exiting with %d", ret)
	os.Exit(ret)
}
