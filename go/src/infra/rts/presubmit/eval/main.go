// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package eval

import (
	"context"
	"flag"
	"fmt"
	"os"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging/gologger"
	"go.chromium.org/luci/common/system/signals"
)

// Main evaluates the selection strategy, prints results and exits the process.
func Main(ctx context.Context, strategy Strategy) {
	ctx, cancel := context.WithCancel(ctx)
	defer signals.HandleInterrupt(cancel)

	ev := &Eval{}
	parseFlags(ev)

	var logCfg = gologger.LoggerConfig{
		Format: `%{message}`,
		Out:    os.Stderr,
	}
	ctx = logCfg.Use(ctx)

	res, err := ev.Run(ctx, strategy)
	if err != nil {
		fatal(err)
	}

	res.Print(os.Stdout, 0 /* minChangeRecall */)
	os.Exit(0)
}

func parseFlags(ev *Eval) {
	if err := ev.RegisterFlags(flag.CommandLine); err != nil {
		fatal(err)
	}
	flag.Parse()
	if len(flag.Args()) > 0 {
		fatal(errors.New("unexpected positional arguments"))
	}
	if err := ev.ValidateFlags(); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "fatal: %s\n", err)
	os.Exit(1)
}
