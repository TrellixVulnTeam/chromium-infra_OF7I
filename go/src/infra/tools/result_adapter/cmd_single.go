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
	sinkpb "go.chromium.org/luci/resultdb/sink/proto/v1"
)

func cmdSingle() *subcommands.Command {
	return &subcommands.Command{
		UsageLine: `single [flags] TEST_CMD [TEST_ARG]...`,
		ShortDesc: "Upload test results for a test suite with a single test to ResultSink",
		LongDesc: text.Doc(`
			Runs the test command and waits for it to finish, then converts the
			test results to ResultSink native format and uploads them to ResultDB via ResultSink.
		`),
		CommandRun: func() subcommands.CommandRun {
			r := &singleRun{}
			r.baseRun.RegisterGlobalFlags()
			return r
		},
	}
}

type singleRun struct {
	baseRun
}

func (r *singleRun) Run(a subcommands.Application, args []string, env subcommands.Env) (ret int) {
	if err := r.validate(); err != nil {
		return r.done(err)
	}

	ctx := cli.GetContext(a, r, env)
	return r.run(ctx, args, r.generateTestResults)
}

// generateTestResults converts test results from results file to sinkpb.TestResult.
func (r *singleRun) generateTestResults(ctx context.Context, _ []byte) ([]*sinkpb.TestResult, error) {
	f, err := os.Open(r.resultFile)
	if err != nil {
		return nil, errors.Annotate(err, "open result file").Err()
	}
	defer f.Close()

	// convert the results to ResultSink native format.
	singleFormat := &SingleResult{}
	if err = singleFormat.ConvertFromJSON(f); err != nil {
		return nil, errors.Annotate(err, "did not recognize as a test suite with a single test").Err()
	}

	trs, err := singleFormat.ToProtos(ctx)
	if err != nil {
		return nil, errors.Annotate(err, "converting as a test suite with a single test").Err()
	}
	return trs, nil
}
