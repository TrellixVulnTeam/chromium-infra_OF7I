// Copyright 2018 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"flag"
	"log"

	"infra/cros/cmd/lucifer/internal/api"
	"infra/cros/cmd/lucifer/internal/autotest/atutil"
	"infra/cros/cmd/lucifer/internal/event"

	"github.com/google/subcommands"
)

type prejobCmd struct {
	testCmd
}

func (prejobCmd) Name() string {
	return "prejob"
}
func (prejobCmd) Synopsis() string {
	return "Run the prejob for a test. No actual test execution."
}
func (prejobCmd) Usage() string {
	return `prejob [FLAGS]

lucifer prejob implements the pre-test steps (provision and/or prejob)
of running an Autotest job. It uses the same flags as lucifer test.
`
}

func (p *prejobCmd) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	ctx, res, err := commonSetup(ctx, p.commonOpts)
	if err != nil {
		log.Print(err)
		return subcommands.ExitFailure
	}
	defer res.Close()
	if err := verifySkylabFlags(p.testCmd); err != nil {
		log.Printf("Error checking the flags: %s", err)
		return subcommands.ExitFailure
	}
	p.prepareFlags()
	if err := runPrejobStep(ctx, p, res.apiClient()); err != nil {
		log.Printf("Error running prejob: %s", err)
		return subcommands.ExitFailure
	}
	return subcommands.ExitSuccess
}

// runPrejobStep is a wrapper of doProvisioningStep or doPrejobStep
// depending on the labels input.
func runPrejobStep(ctx context.Context, p *prejobCmd, ac *api.Client) error {
	event.Send(event.Starting)
	defer event.Send(event.Completed)
	if len(p.provisionLabels) > 0 {
		return doProvisioningStep(ctx, &p.testCmd, ac)
	}
	if p.prejobTask != atutil.NoTask {
		return doPrejobStep(ctx, &p.testCmd, ac)
	}
	// TODO(linxinan): return an no-op result rather than nil.
	return nil
}
