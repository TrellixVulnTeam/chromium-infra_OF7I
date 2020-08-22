// Copyright 2020 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/maruel/subcommands"

	"go.chromium.org/luci/common/data/text"
	"go.chromium.org/luci/common/errors"
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
// TODO(crbug.com/1108016): Implement.
func (r *baseRun) initSinkClient(ctx context.Context) (err error) {
	return errors.New("not implemented yet")
}

// runTestCmd waits for test cmd to complete.
// TODO(crbug.com/1108016): Implement.
func (r *baseRun) runTestCmd(ctx context.Context, args []string) (err error) {
	return errors.New("not implemented yet")
}

func (r *baseRun) done(err error) int {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return ExitCodeCommandFailure
	}
	return 0
}
