// Copyright 2021 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package main

import (
	"github.com/maruel/subcommands"

	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/data/text"
)

type goRun struct {
	baseRun
}

func cmdGo() *subcommands.Command {
	return &subcommands.Command{
		UsageLine: `go -- go test [TEST_ARG]...`,
		ShortDesc: "Batch upload results of golang test result format to ResultSink",
		LongDesc: text.Doc(`
			Runs the test command and waits for it to finish, then converts the json output
			test results to ResultSink native format and uploads them to ResultDB via ResultSink.
		`),
		CommandRun: func() subcommands.CommandRun {
			r := &goRun{}
			r.captureOutput = true
			// Ignore global flags, go tests are expected to only produce
			// standard output.
			return r
		},
	}
}

func (r *goRun) Run(a subcommands.Application, args []string, env subcommands.Env) (ret int) {
	args, err := r.ensureArgsValid(args)
	if err != nil {
		return r.done(err)
	}

	ctx := cli.GetContext(a, r, env)
	return r.run(ctx, args, r.generateTestResults)
}
