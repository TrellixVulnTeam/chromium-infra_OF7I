// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/maruel/subcommands"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/encoding/protojson"

	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/data/stringset"
	"go.chromium.org/luci/common/data/text"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"

	"infra/rts/filegraph/git"
	evalpb "infra/rts/presubmit/eval/proto"
)

func cmdSelect() *subcommands.Command {
	return &subcommands.Command{
		UsageLine: `select -changed-files <path> -model-dir <path> -filter-files-dir <path>`,
		ShortDesc: "compute the set of test files to skip",
		LongDesc: text.Doc(`
			Compute the set of test files to skip.

			Flags -changed-files, -model-dir and -skip-test-files are required.
		`),
		CommandRun: func() subcommands.CommandRun {
			r := &selectRun{}
			r.Flags.StringVar(&r.changedFilesPath, "changed-files", "", text.Doc(`
				Path to the file with changed files.
				Each line of the file must be a filename, with "//" prefix.
			`))
			r.Flags.StringVar(&r.modelDir, "model-dir", "", text.Doc(`
				Path to the directory with the model files.
				Normally it is coming from CIPD package "chromium/rts/model"
				and precomputed by "rts-chromium create-model" command.
			`))
			r.Flags.StringVar(&r.filterFilesDir, "filter-files-dir", "", text.Doc(`
				Path to a directory where to write test filter files.
				A file per test suite is written, e.g. browser_tests.filter.
				The file format is described in https://chromium.googlesource.com/chromium/src/+/master/testing/buildbot/filters/README.md.
				Before writing, all .filter files in the directory are deleted.
			`))
			r.Flags.Float64Var(&r.targetChangeRecall, "target-change-recall", 0.99, text.Doc(`
				The target fraction of bad changes to be caught by the selection strategy.
				It must be a value in (0.0, 1.0) range.
			`))
			return r
		},
	}
}

type selectRun struct {
	baseCommandRun

	// Direct input.

	changedFilesPath   string
	modelDir           string
	filterFilesDir     string
	targetChangeRecall float64

	// Indirect input.

	testFiles    map[string]*TestFile // indexed by source-absolute test file name
	changedFiles stringset.Set
	strategy     git.SelectionStrategy
	evalResult   *evalpb.Results
}

func (r *selectRun) validateFlags() error {
	switch {
	case r.changedFilesPath == "":
		return errors.New("-changed-files is required")
	case r.modelDir == "":
		return errors.New("-model-dir is required")
	case r.filterFilesDir == "":
		return errors.New("-filter-files-dir is required")
	case !(r.targetChangeRecall > 0 && r.targetChangeRecall < 1):
		return errors.New("-target-change-recall must be in (0.0, 1.0) range")
	default:
		return nil
	}
}

func (r *selectRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	ctx := cli.GetContext(a, r, env)
	if len(args) != 0 {
		return r.done(errors.New("unexpected positional arguments"))
	}

	if err := r.validateFlags(); err != nil {
		return r.done(err)
	}

	if err := r.loadInput(ctx); err != nil {
		return r.done(err)
	}

	threshold := r.chooseThreshold()
	if threshold == nil {
		return r.done(errors.Reason("no threshold for target change recall %.4f", r.targetChangeRecall).Err())
	}
	r.strategy.MaxDistance = float64(threshold.MaxDistance)
	logging.Infof(ctx, "chosen threshold: %f", r.strategy.MaxDistance)

	return r.done(r.writeFilterFiles())
}

// writeFilterFiles writes filter files in r.filterFilesDir directory.
func (r *selectRun) writeFilterFiles() error {
	if err := prepareOutDir(r.filterFilesDir, "*.filter"); err != nil {
		return errors.Annotate(err, "failed to prepare filter file dir %q", r.filterFilesDir).Err()
	}

	// maps a test suite to the list of tests to skip.
	testsToSkip := map[string][]string{}
	err := r.selectTests(func(testFileToSkip *TestFile) error {
		for _, testSuite := range testFileToSkip.TestSuites {
			testsToSkip[testSuite] = append(testsToSkip[testSuite], testFileToSkip.TestNames...)
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Write the files.
	for testSuite, testNames := range testsToSkip {
		fileName := filepath.Join(r.filterFilesDir, testSuite+".filter")
		if err := writeFilterFile(fileName, testNames); err != nil {
			return errors.Annotate(err, "failed to write %q", fileName).Err()
		}
		fmt.Printf("wrote %s\n", fileName)
	}
	return nil
}

func writeFilterFile(fileName string, toSkip []string) error {
	f, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, name := range toSkip {
		if _, err := fmt.Fprintf(f, "-%s\n", name); err != nil {
			return err
		}
	}
	return f.Close()
}

// chooseThreshold returns the distance threshold based on
// r.targetChangeRecall and r.evalResult.
func (r *selectRun) chooseThreshold() *evalpb.Threshold {
	var ret *evalpb.Threshold
	for _, t := range r.evalResult.Thresholds {
		if t.ChangeRecall < float32(r.targetChangeRecall) {
			continue
		}
		if ret == nil || ret.ChangeRecall > t.ChangeRecall {
			ret = t
		}
	}
	return ret
}

// loadInput loads all the input of the subcommand.
func (r *selectRun) loadInput(ctx context.Context) error {
	eg, ctx := errgroup.WithContext(ctx)
	defer eg.Wait()

	gitGraphDir := filepath.Join(r.modelDir, "git-file-graph")
	eg.Go(func() error {
		err := r.loadGraph(filepath.Join(gitGraphDir, "graph.fg"))
		return errors.Annotate(err, "failed to load file graph").Err()
	})
	eg.Go(func() error {
		err := r.loadEvalResult(filepath.Join(gitGraphDir, "eval-results.json"))
		return errors.Annotate(err, "failed to load eval results").Err()
	})

	eg.Go(func() (err error) {
		err = r.loadTestFileSet(filepath.Join(r.modelDir, "test-files.jsonl"))
		return errors.Annotate(err, "failed to load test files set").Err()
	})

	eg.Go(func() (err error) {
		r.changedFiles, err = loadStringSet(r.changedFilesPath)
		return errors.Annotate(err, "failed to load changed files set").Err()
	})

	return eg.Wait()
}

// loadEvalResult loads r.evalResult, including thresholds.
func (r *selectRun) loadEvalResult(fileName string) error {
	resBytes, err := ioutil.ReadFile(fileName)
	if err != nil {
		return err
	}
	r.evalResult = &evalpb.Results{}
	return protojson.Unmarshal(resBytes, r.evalResult)
}

// loadGraph loads r.strategy.Graph from the model.
func (r *selectRun) loadGraph(fileName string) error {
	f, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer f.Close()

	// Note: it might be dangerous to sync with the current checkout.
	// There might have been such change in the repo that the chosen threshold,
	// the model or both are no longer good. Thus, do not sync.
	r.strategy.Graph = &git.Graph{}
	return r.strategy.Graph.Read(bufio.NewReader(f))
}

// loadTestFileSet loads r.testFiles.
func (r *selectRun) loadTestFileSet(fileName string) error {
	f, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer f.Close()

	r.testFiles = map[string]*TestFile{}
	return readTestFiles(bufio.NewReader(f), func(file *TestFile) error {
		r.testFiles[file.Path] = file
		return nil
	})
}

// loadStringSet loads a set of newline-separated strings from a text file.
func loadStringSet(fileName string) (stringset.Set, error) {
	f, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	set := stringset.New(0)
	scan := bufio.NewScanner(f)
	for scan.Scan() {
		set.Add(scan.Text())
	}
	return set, scan.Err()
}
