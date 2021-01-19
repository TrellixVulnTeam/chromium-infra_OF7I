// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"compress/zlib"
	"context"
	"flag"
	"fmt"
	"os"

	bbpb "go.chromium.org/luci/buildbucket/proto"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/luciexe/exe"
	"infra/cros/cmd/cros_test_runner/execute"
)

func main() {
	exe.Run(luciEXEMain, exe.WithZlibCompression(zlib.BestCompression))
}

func luciEXEMain(ctx context.Context, input *bbpb.Build, userargs []string, send exe.BuildSender) (merr error) {
	// crbug.com/1112514: The input Build Status is not currently specified in
	// the luciexe protocol. bbagent sets it, but recipe_engine's sub_build()
	// doesn't. Play safe.
	input.Status = bbpb.Status_STARTED
	send()

	// All errors in cros_test_runner are interpreted as infrastructure
	// failures by the recipe.
	// Test failures and user errors are reported in other recipe steps.
	//
	// Thus,
	// [1] All errors from this binary are tagged as INFRA_FAILURE
	// [2] Non-INFRA_FAILUREs are suppressed (before they get to this top-level
	//     function)
	defer func() {
		merr = exe.InfraErrorTag.Apply(merr)
	}()

	ca, err := parseArgs(userargs)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		flag.PrintDefaults()
		return err
	}

	ctx = setupLogging(ctx)
	if err := execute.Run(ctx, execute.Args{
		InputPath:      ca.InputPath,
		OutputPath:     ca.OutputPath,
		SwarmingTaskID: os.Getenv("SWARMING_TASK_ID"),

		Build: input,
		Send:  send,
	}); err != nil {
		logApplicationError(ctx, err)
		return err
	}
	return nil
}

func setupLogging(ctx context.Context) context.Context {
	return logging.SetLevel(ctx, logging.Info)
}

// logApplicationError logs the error returned to the entry function of an
// application.
func logApplicationError(ctx context.Context, err error) {
	errors.Log(ctx, err)
	// Also log to error stream, since logs are directed at the main output
	// stream.
	fmt.Fprintf(os.Stderr, "%s\n", err)
}

type cliArgs struct {
	InputPath  string
	OutputPath string
}

func parseArgs(args []string) (cliArgs, error) {
	ca := cliArgs{}

	fs := flag.NewFlagSet("test_runner", flag.ContinueOnError)
	fs.StringVar(&ca.InputPath, "input_json", "", "Path to JSON skylab_test_runner.Request to read.")
	fs.StringVar(&ca.OutputPath, "output_json", "", "Path to JSON skylab_test_runner.Result to write.")
	if err := fs.Parse(args); err != nil {
		return ca, err
	}

	errs := errors.NewMultiError()
	if ca.InputPath == "" {
		errs = append(errs, errors.Reason("-input_json not specified").Err())
	}
	if ca.OutputPath == "" {
		errs = append(errs, errors.Reason("-output_json not specified").Err())
	}

	if errs.First() != nil {
		return ca, errs
	}
	return ca, nil
}
