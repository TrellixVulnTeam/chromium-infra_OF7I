// Copyright 2022 The LUCI Authors. All rights reserved.
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

func cmdGtestJson() *subcommands.Command {
	return &subcommands.Command{
		UsageLine: `gtest_json [flags] TEST_CMD [TEST_ARG]...`,
		ShortDesc: "Batch upload results from the gtest flag 'gtest_output:json' to ResultSink",
		LongDesc: text.Doc(`
			Runs the test command and waits for it to finish, then converts the results
			from the gtest flag 'gtest_output:json' to ResultSink native format and uploads
			them to ResultDB via ResultSink.
		`),
		CommandRun: func() subcommands.CommandRun {
			r := &gtestJsonRun{}
			r.baseRun.RegisterGlobalFlags()
			return r
		},
	}
}

type gtestJsonRun struct {
	baseRun
}

func (r *gtestJsonRun) Run(a subcommands.Application, args []string, env subcommands.Env) (ret int) {
	if err := r.validate(); err != nil {
		return r.done(err)
	}

	ctx := cli.GetContext(a, r, env)
	return r.run(ctx, args, r.generateTestResults)
}

// generateTestResults converts test results from results file to sinkpb.TestResult.
func (r *gtestJsonRun) generateTestResults(ctx context.Context, _ []byte) ([]*sinkpb.TestResult, error) {
	f, err := os.Open(r.resultFile)
	if err != nil {
		return nil, errors.Annotate(err, "open result file").Err()
	}
	defer f.Close()

	// convert the results to ResultSink native format.
	gtestFormat := &GTestJsonResults{}
	if err = gtestFormat.ConvertFromJSON(f); err != nil {
		return nil, errors.Annotate(err, "did not recognize as GTest").Err()
	}

	trs, err := gtestFormat.ToProtos(ctx)
	if err != nil {
		return nil, errors.Annotate(err, "converting as GTest results format").Err()
	}
	return trs, nil
}
