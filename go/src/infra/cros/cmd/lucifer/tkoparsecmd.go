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

type tkoParseCmd struct {
	testCmd
}

func (tkoParseCmd) Name() string {
	return "tkoparse"
}
func (tkoParseCmd) Synopsis() string {
	return "Parse test results and upload them to TKO. Test run should have completed."
}
func (tkoParseCmd) Usage() string {
	return `tkoparse [FLAGS]

lucifer tkoparse implements the post-test steps (parsing and uploading) of
running an Autotest job. It uses the same flags as lucifer test.
`
}

func (t *tkoParseCmd) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	ctx, res, err := commonSetup(ctx, t.commonOpts)
	if err != nil {
		log.Print(err)
		return subcommands.ExitFailure
	}
	defer res.Close()
	if err := verifySkylabFlags(t.testCmd); err != nil {
		log.Printf("Error checking the flags: %s", err)
		return subcommands.ExitFailure
	}
	t.prepareFlags()
	if err := runParseStep(ctx, t, res.apiClient()); err != nil {
		log.Printf("Error running tko/parse: %s", err)
		return subcommands.ExitFailure
	}
	return subcommands.ExitSuccess
}

// runParseStep is a wrapper of doParsingStep which parses and uploads the test result.
func runParseStep(ctx context.Context, t *tkoParseCmd, ac *api.Client) error {
	event.Send(event.Starting)
	defer event.Send(event.Completed)
	if ctx.Err() == nil {
		return doParsingStep(ctx, &t.testCmd, ac)
	}
	event.Send(event.Aborted)
	return fmt.Errorf("aborted")
}
