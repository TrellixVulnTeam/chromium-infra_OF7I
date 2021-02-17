// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package run

import (
	"flag"
	"fmt"
	"infra/cmd/crosfleet/internal/flagx"
	"infra/cmdsupport/cmdlib"
	"strings"
)

var (
	// DefaultSwarmingPriority is the default priority for a Swarming task.
	DefaultSwarmingPriority = 140
	// MinSwarmingPriority is the lowest-allowed priority for a Swarming task.
	MinSwarmingPriority = 50
	// MaxSwarmingPriority is the highest-allowed priority for a Swarming task.
	MaxSwarmingPriority = 255
)

// testCommonFlags contains parameters common to the "run
// test", "run suite", and "run testplan" subcommands.
type testCommonFlags struct {
	board           string
	model           string
	pool            string
	image           string
	qsAccount       string
	maxRetries      int
	priority        int
	timeoutMins     int
	dimensions      map[string]string
	provisionLabels map[string]string
	tags            map[string]string
	keyvals         map[string]string
	json            bool
}

// Registers run command-specific flags
func (c *testCommonFlags) Register(f *flag.FlagSet) {
	f.StringVar(&c.image, "image", "", "Fully specified image name to run test against, e.g. octopus-release/R89-13609.0.0.")
	f.StringVar(&c.board, "board", "", "Board to run tests on.")
	f.StringVar(&c.model, "model", "", "Model to run tests on.")
	f.StringVar(&c.pool, "pool", "", "Device pool to run tests on.")
	f.StringVar(&c.qsAccount, "qs-account", "", `Optional Quota Scheduler account to use for this task. Overrides -priority flag.
If no account is set, tests are scheduled using -priority flag.`)
	f.IntVar(&c.maxRetries, "max-retries", 0, "Maximum retries allowed. No retry if set to 0.")
	f.IntVar(&c.priority, "priority", DefaultSwarmingPriority, `Swarming scheduling priority for tests, between 50 and 255 (lower values indicate higher priorities).
If a Quota Scheduler account is specified via -qs-account, this value is not used.`)
	f.IntVar(&c.timeoutMins, "timeout-mins", 30, "Test run timeout.")
	f.Var(flagx.KeyVals(&c.dimensions), "dim", "Additional scheduling dimension in format key=val or key:val; may be specified multiple times.")
	f.Var(flagx.KeyVals(&c.dimensions), "dims", "Comma-separated additional scheduling dimensions in same format as -dim.")
	f.Var(flagx.KeyVals(&c.provisionLabels), "provision-label", "Additional provisionable label in format key=val or key:val; may be specified multiple times.")
	f.Var(flagx.KeyVals(&c.provisionLabels), "provision-labels", "Comma-separated additional provisionable labels in same format as -provision-label.")
	f.Var(flagx.KeyVals(&c.tags), "tag", "Swarming tag in format key=val or key:val; may be specified multiple times.")
	f.Var(flagx.KeyVals(&c.tags), "tags", "Comma-separated Swarming tags in same format as -tag.")
	f.Var(flagx.KeyVals(&c.keyvals), "autotest-keyval", "Autotest keyval in format key=val or key:val; may be specified multiple times.")
	f.Var(flagx.KeyVals(&c.keyvals), "autotest-keyvals", "Comma-separated Autotest keyvals in same format as -keyval.")
	f.BoolVar(&c.json, "json", false, "Format output as JSON.")
}

func (c *testCommonFlags) validateArgs(f *flag.FlagSet, mainArgType string) error {
	var errors []string
	if c.board == "" {
		errors = append(errors, "missing board flag")
	}
	if c.pool == "" {
		errors = append(errors, "missing pool flag")
	}
	if c.image == "" {
		errors = append(errors, "missing image flag")
	}
	if c.priority < MinSwarmingPriority || c.priority > MaxSwarmingPriority {
		errors = append(errors, fmt.Sprintf("priority flag should be in [%d, %d]", MinSwarmingPriority, MaxSwarmingPriority))
	}
	if f.NArg() == 0 {
		errors = append(errors, fmt.Sprintf("missing %v arg", mainArgType))
	}

	if len(errors) > 0 {
		return cmdlib.NewUsageError(*f, strings.Join(errors, "\n"))
	}
	return nil
}
