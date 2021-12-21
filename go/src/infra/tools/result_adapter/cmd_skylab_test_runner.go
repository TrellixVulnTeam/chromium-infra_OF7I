// Copyright 2021 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package main

import (
	"context"
	"os"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/data/text"
	"go.chromium.org/luci/common/errors"
	sinkpb "go.chromium.org/luci/resultdb/sink/proto/v1"
)

func cmdSkylabTestRunner() *subcommands.Command {
	return &subcommands.Command{
		UsageLine: `skylab-test-runner [flags] TEST_CMD [TEST_ARG]...`,
		ShortDesc: "Batch upload skylab_test_runner results to ResultSink",
		LongDesc: text.Doc(`
			Runs the test command and waits for it to finish, then converts the skylab_test_runner
			results to ResultSink native format and uploads them to ResultDB via ResultSink.
			A JSON line file is expected for -result-file.
		`),
		CommandRun: func() subcommands.CommandRun {
			r := &skylabTestRunnerRun{}
			r.baseRun.RegisterGlobalFlags()
			return r
		},
	}
}

type skylabTestRunnerRun struct {
	baseRun
}

func (r *skylabTestRunnerRun) validate() (err error) {
	return r.baseRun.validate()
}

func (r *skylabTestRunnerRun) Run(a subcommands.Application, args []string, env subcommands.Env) (ret int) {
	if err := r.validate(); err != nil {
		return r.done(err)
	}

	ctx := cli.GetContext(a, r, env)
	return r.run(ctx, args, r.generateTestResults)
}

// generateTestResults converts test results from results file to sinkpb.TestResult.
func (r *skylabTestRunnerRun) generateTestResults(ctx context.Context, _ []byte) ([]*sinkpb.TestResult, error) {
	f, err := os.Open(r.resultFile)
	if err != nil {
		return nil, errors.Annotate(err, "open result file").Err()
	}
	defer f.Close()

	skylabTestRunnerFormat := &TestRunnerResult{}
	// Convert the results to ResultSink native format.
	if err = skylabTestRunnerFormat.ConvertFromJSON(f); err != nil {
		return nil, errors.Annotate(err, "did not recognize as skylab_test_runner Result").Err()
	}
	trs, err := skylabTestRunnerFormat.ToProtos(ctx)
	if err != nil {
		return nil, errors.Annotate(err, "converting as skylab_test_runner Results").Err()
	}
	return trs, nil
}
