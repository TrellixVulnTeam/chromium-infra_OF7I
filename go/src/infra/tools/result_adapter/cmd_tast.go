// Copyright 2021 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package main

import (
	"context"
	"os"
	"strings"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/data/text"
	"go.chromium.org/luci/common/errors"
	sinkpb "go.chromium.org/luci/resultdb/sink/proto/v1"
)

func cmdTast() *subcommands.Command {
	return &subcommands.Command{
		UsageLine: `tast [flags] TEST_CMD [TEST_ARG]...`,
		ShortDesc: "Batch upload results of the tast execution to ResultSink",
		LongDesc: text.Doc(`
			Runs the test command and waits for it to finish, then converts the tast
			test results to ResultSink native format and uploads them to ResultDB via ResultSink.
			A JSON line file e.g. tast/results/streamed_results.jsonl is expected for -result-file.
			The absolute path for tast result folder should be specified by -artifact-directory.
		`),
		CommandRun: func() subcommands.CommandRun {
			r := &tastRun{}
			r.baseRun.RegisterGlobalFlags()
			return r
		},
	}
}

type tastRun struct {
	baseRun
}

func (r *tastRun) validate() (err error) {
	if r.artifactDir == "" {
		return errors.Reason("-artifact-directory is required").Err()
	}
	return r.baseRun.validate()
}

func (r *tastRun) Run(a subcommands.Application, args []string, env subcommands.Env) (ret int) {
	if err := r.validate(); err != nil {
		return r.done(err)
	}

	ctx := cli.GetContext(a, r, env)
	return r.run(ctx, args, r.generateTestResults)
	//TODO(crbug.com/1238139): Upload invocation level logs.
}

// generateTestResults converts test results from results file to sinkpb.TestResult.
func (r *tastRun) generateTestResults(ctx context.Context, _ []byte) ([]*sinkpb.TestResult, error) {
	f, err := os.Open(r.resultFile)
	if err != nil {
		return nil, errors.Annotate(err, "open result file").Err()
	}
	defer f.Close()

	tastFormat := &TastResults{
		BaseDir: strings.TrimSuffix(r.artifactDir, "/"),
	}
	// Convert the results to ResultSink native format.
	if err = tastFormat.ConvertFromJSON(f); err != nil {
		return nil, errors.Annotate(err, "did not recognize as Tast").Err()
	}
	trs, err := tastFormat.ToProtos(ctx, processArtifacts)
	if err != nil {
		return nil, errors.Annotate(err, "converting as Tast results format").Err()
	}
	return trs, nil
}
