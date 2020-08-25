// Copyright 2020 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/maruel/subcommands"

	"go.chromium.org/luci/common/data/text"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/system/signals"
	"go.chromium.org/luci/grpc/prpc"
	"go.chromium.org/luci/lucictx"
	sinkpb "go.chromium.org/luci/resultdb/sink/proto/v1"
)

// ExitCodeCommandFailure indicates that a given command failed due to internal errors
// or invalid input parameters.
const ExitCodeCommandFailure = 123

// baseRun provides common command run functionality.
type baseRun struct {
	subcommands.CommandRunBase

	// Flags.
	artifactDir string
	resultFile  string

	sinkCtx *lucictx.ResultSink
	sinkC   sinkpb.SinkClient
}

func (r *baseRun) RegisterGlobalFlags() {
	r.Flags.StringVar(&r.artifactDir, "artifact-directory", "", text.Doc(`
				Directory of the artifacts. Required.
			`))
	r.Flags.StringVar(&r.resultFile, "result-file", "", text.Doc(`
				Path to the result output file. Required.
			`))
}

// validate validates the command has required flags.
func (r *baseRun) validate() (err error) {
	if r.artifactDir == "" {
		return errors.Reason("-artifact-directory is required").Err()
	}
	if r.resultFile == "" {
		return errors.Reason("-result-file is required").Err()
	}
	return nil
}

// initSinkClient initializes the result sink client.
func (r *baseRun) initSinkClient(ctx context.Context) (err error) {
	r.sinkCtx = lucictx.GetResultSink(ctx)
	if r.sinkCtx == nil {
		return errors.Reason("no result sink info found in $LUCI_CONTEXT").Err()
	}

	r.sinkC = sinkpb.NewSinkPRPCClient(&prpc.Client{
		Host:    r.sinkCtx.Address,
		Options: &prpc.Options{Insecure: true},
	})

	return nil
}

// runTestCmd waits for test cmd to complete.
// TODO(crbug.com/1108016): Implement.
func (r *baseRun) runTestCmd(ctx context.Context, args []string) (err error) {
	// Kill the subprocess if is asked to stop.
	// Subprocess exiting will unblock result_uploader and will stop soon.
	cmdCtx, cancelCmd := context.WithCancel(ctx)
	defer cancelCmd()
	defer signals.HandleInterrupt(func() {
		logging.Warningf(ctx, "result_uploader: interrupt signal received; killing the subprocess")
		cancelCmd()
	})()

	cmd := exec.CommandContext(cmdCtx, args[0], args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return errors.Annotate(err, "cmd.start").Err()
	}
	return cmd.Wait()
}

func (r *baseRun) done(err error) int {
	if err != nil {
		fmt.Fprintf(os.Stderr, "result_adapter: %s\n", err)
		return ExitCodeCommandFailure
	}
	return 0
}
