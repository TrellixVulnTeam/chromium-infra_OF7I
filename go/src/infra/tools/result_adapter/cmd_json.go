// Copyright 2020 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package main

import (
	"context"
	"os"
	"path/filepath"

	"github.com/maruel/subcommands"

	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/data/text"
	"go.chromium.org/luci/common/errors"
	sinkpb "go.chromium.org/luci/resultdb/sink/proto/v1"
)

func cmdJSON() *subcommands.Command {
	return &subcommands.Command{
		UsageLine: `json [flags] TEST_CMD [TEST_ARG]...`,
		ShortDesc: "Batch upload results of json test result format to ResultSink",
		LongDesc: text.Doc(`
			Runs the test command and waits for it to finish, then converts the json
			test results to ResultSink native format and uploads them to ResultDB via ResultSink.
		`),
		CommandRun: func() subcommands.CommandRun {
			r := &jsonRun{}
			r.baseRun.RegisterGlobalFlags()
			r.Flags.BoolVar(&r.testLocations, "test-location", false, text.Doc(`
				Flag for set testLocation in each TestResult.
				If true, testIds will be used as testLocations.

				It only makes sense to set the flag for blink_web_tests and webgl_conformance_tests.
			`))
			return r
		},
	}
}

type jsonRun struct {
	baseRun

	testLocations bool
}

func (r *jsonRun) Run(a subcommands.Application, args []string, env subcommands.Env) (ret int) {
	if err := r.validate(); err != nil {
		return r.done(err)
	}

	ctx := cli.GetContext(a, r, env)
	return r.run(ctx, args, r.generateTestResults)
}

func (r *jsonRun) validate() (err error) {
	if r.artifactDir == "" {
		return errors.Reason("-artifact-directory is required").Err()
	}
	return r.baseRun.validate()
}

// generateTestResults converts test results from results file to sinkpb.TestResult.
func (r *jsonRun) generateTestResults(ctx context.Context, _ []byte) ([]*sinkpb.TestResult, error) {
	// Get artifacts.
	normPathToFullPath, err := r.processArtifacts()
	if err != nil {
		return nil, errors.Annotate(err, "open artifact directory").Err()
	}

	// Get results.
	f, err := os.Open(r.resultFile)
	if err != nil {
		return nil, errors.Annotate(err, "open result file").Err()
	}
	defer f.Close()

	jsonFormat := &JSONTestResults{}
	if err = jsonFormat.ConvertFromJSON(f); err != nil {
		return nil, errors.Annotate(err, "did not recognize as json test result format").Err()
	}

	// Convert the results to ResultSink native format.
	trs, err := jsonFormat.ToProtos(ctx, normPathToFullPath, r.testLocations)
	if err != nil {
		return nil, errors.Annotate(err, "converting as json results format").Err()
	}
	return trs, nil
}

// processArtifacts walks the files in r.artifactDir then returns a map from normalized relative path to full path.
func (r *jsonRun) processArtifacts() (normPathToFullPath map[string]string, err error) {
	normPathToFullPath = map[string]string{}
	err = filepath.Walk(r.artifactDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			// normPath is the normalized relative path to r.artifactDir.
			relPath, err := filepath.Rel(r.artifactDir, path)
			if err != nil {
				return err
			}
			normPath := normalizePath(relPath)
			normPathToFullPath[normPath] = path
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return normPathToFullPath, err
}
