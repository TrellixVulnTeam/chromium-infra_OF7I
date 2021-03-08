// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Command luciexe implements the cros_test_platform build logic.
//
// This is a luciexe binary as defined at
// https://godoc.org/go.chromium.org/luci/luciexe
//
// This binary is intended to slowly replace all the sub-commands of the
// cros_test_platform binary, eventually replacing the cros_test_platform
// recipe and Go binary.
package main

import (
	"compress/zlib"
	"context"
	"flag"
	"fmt"
	"os"

	"infra/cmd/cros_test_platform/luciexe/execute"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/luciexe/exe"

	bbpb "go.chromium.org/luci/buildbucket/proto"
)

func main() {
	exe.Run(luciEXEMain, exe.WithZlibCompression(zlib.BestCompression))
}

// The exe.MainFn entry point for this luciexe binary.
//
// At the moment, this binary replaces only the skylab-execute step
// of cros_test_platform.
func luciEXEMain(ctx context.Context, input *bbpb.Build, userArgs []string, send exe.BuildSender) (merr error) {
	// crbug.com/1112514: The input Build Status is not currently specified in
	// the luciexe protocol. bbagent sets it, but recipe_engine's sub_build()
	// doesn't. Play safe.
	input.Status = bbpb.Status_STARTED
	send()

	// All errors in skylab-execute step are interpreted as infrastructure
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

	ca, err := parseArgs(userArgs)
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

	fs := flag.NewFlagSet("cros_test_platform", flag.ContinueOnError)
	fs.StringVar(&ca.InputPath, "input_json", "", "Path to JSON ExecuteRequests to read.")
	fs.StringVar(&ca.OutputPath, "output_json", "", "Path to JSON ExecuteResponses to write.")
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
