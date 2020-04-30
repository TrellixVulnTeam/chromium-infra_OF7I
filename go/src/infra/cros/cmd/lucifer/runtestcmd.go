// Copyright 2018 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"infra/cros/cmd/lucifer/internal/api"
	"infra/cros/cmd/lucifer/internal/event"

	"github.com/google/subcommands"
)

type runTestCmd struct {
	testCmd
}

func (runTestCmd) Name() string {
	return "runtest"
}
func (runTestCmd) Synopsis() string {
	return "Excute actual test. Pre-test steps should have done."
}
func (runTestCmd) Usage() string {
	return `runtest [FLAGS]

lucifer runtest executes the running steps of an Autotest job. It uses
the same flags as lucifer test.
`
}

func (r *runTestCmd) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	ctx, res, err := commonSetup(ctx, r.commonOpts)
	if err != nil {
		log.Print(err)
		return subcommands.ExitFailure
	}
	defer res.Close()
	if err := verifySkylabFlags(r.testCmd); err != nil {
		log.Printf("Error checking the flags: %s", err)
		return subcommands.ExitFailure
	}
	r.prepareFlags()
	if err := runTestStep(ctx, r, res.apiClient()); err != nil {
		log.Printf("Error running test: %s", err)
		return subcommands.ExitFailure
	}
	return subcommands.ExitSuccess
}

// runTestStep is a wrapper of doRunningStep which performs the actual test.
func runTestStep(ctx context.Context, r *runTestCmd, ac *api.Client) error {
	event.Send(event.Starting)
	defer event.Send(event.Completed)
	if ctx.Err() == nil {
		return doRunningStep(ctx, &r.testCmd, ac)
	}
	event.Send(event.Aborted)
	return fmt.Errorf("aborted")
}
