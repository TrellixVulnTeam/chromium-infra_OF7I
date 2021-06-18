// Copyright 2020 The LUCI Authors. All rights reserved.
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
	"go.chromium.org/luci/common/logging"
	sinkpb "go.chromium.org/luci/resultdb/sink/proto/v1"
)

func cmdGtest() *subcommands.Command {
	return &subcommands.Command{
		UsageLine: `gtest [flags] TEST_CMD [TEST_ARG]...`,
		ShortDesc: "Batch upload results of the gtest execution to ResultSink",
		LongDesc: text.Doc(`
			Runs the test command and waits for it to finish, then converts the gtest
			test results to ResultSink native format and uploads them to ResultDB via ResultSink.
		`),
		CommandRun: func() subcommands.CommandRun {
			r := &gtestRun{}
			r.baseRun.RegisterGlobalFlags()
			return r
		},
	}
}

type gtestRun struct {
	baseRun
}

func (r *gtestRun) Run(a subcommands.Application, args []string, env subcommands.Env) (ret int) {
	if err := r.validate(); err != nil {
		return r.done(err)
	}

	ctx := cli.GetContext(a, r, env)
	return r.run(ctx, args, r.generateTestResults)
}

// generateTestResults converts test results from results file to sinkpb.TestResult.
func (r *gtestRun) generateTestResults(ctx context.Context, _ []byte) ([]*sinkpb.TestResult, error) {
	f, err := os.Open(r.resultFile)
	if err != nil {
		return nil, errors.Annotate(err, "open result file").Err()
	}
	defer f.Close()

	// convert the results to ResultSink native format.
	gtestFormat := &GTestResults{}
	if err = gtestFormat.ConvertFromJSON(f); err != nil {
		return nil, errors.Annotate(err, "did not recognize as GTest").Err()
	}

	if len(gtestFormat.AllTests) == 0 {
		logging.Warningf(ctx, `"all_tests" field is empty in JSON`)
		return nil, nil
	}

	trs, err := gtestFormat.ToProtos(ctx)
	if err != nil {
		return nil, errors.Annotate(err, "converting as GTest results format").Err()
	}
	return trs, nil
}
