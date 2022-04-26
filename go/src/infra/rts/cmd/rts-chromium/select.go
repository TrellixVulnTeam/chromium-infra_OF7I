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
	"strings"

	"github.com/maruel/subcommands"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/encoding/protojson"

	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/data/stringset"
	"go.chromium.org/luci/common/data/text"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"

	"infra/rts/filegraph/git"
	"infra/rts/internal/chromium"
	"infra/rts/internal/gitutil"
	evalpb "infra/rts/presubmit/eval/proto"
)

func cmdSelect() *subcommands.Command {
	return &subcommands.Command{
		UsageLine: `select -changed-files <path> -model-dir <path> -out <path>`,
		ShortDesc: "compute the set of test files to skip",
		LongDesc: text.Doc(`
			Compute the set of test files to skip.

			Flags -changed-files, -model-dir and -out are required.
		`),
		CommandRun: func() subcommands.CommandRun {
			r := &selectRun{}
			r.Flags.StringVar(&r.checkout, "checkout", "", "Path to a src.git checkout")
			r.Flags.StringVar(&r.modelDir, "model-dir", "", text.Doc(`
				Path to the directory with the model files.
				Normally it is coming from CIPD package "chromium/rts/model"
				and precomputed by "rts-chromium create-model" command.
			`))
			r.Flags.StringVar(&r.out, "out", "", text.Doc(`
				Path to a directory where to write test filter files.
				A file per test target is written, e.g. browser_tests.filter.
				The file format is described in https://chromium.googlesource.com/chromium/src/+/HEAD/testing/buildbot/filters/README.md.
				Before writing, all .filter files in the directory are deleted.

				The out directory may be empty. It may happen if the selection strategy
				decides to run all tests, e.g. if //DEPS is changed.
			`))
			r.Flags.Float64Var(&r.targetChangeRecall, "target-change-recall", 0.99, text.Doc(`
				The target fraction of bad changes to be caught by the selection strategy.
				It must be a value in (0.0, 1.0) range.
			`))
			r.Flags.BoolVar(&r.ignoreExceptions, "ignore-exceptions", false, "For debugging. Whether we should ignore exceptions.")
			return r
		},
	}
}

type selectRun struct {
	baseCommandRun

	// Direct input.

	checkout           string
	modelDir           string
	out                string
	targetChangeRecall float64
	ignoreExceptions   bool

	// Indirect input.

	testFiles    map[string]*chromium.TestFile // indexed by source-absolute test file name
	changedFiles stringset.Set                 // files different between origin/main and the working tree
	strategy     git.SelectionStrategy
}

func (r *selectRun) validateFlags() error {
	switch {
	case r.checkout == "":
		return errors.New("-checkout is required")
	case r.modelDir == "":
		return errors.New("-model-dir is required")
	case r.out == "":
		return errors.New("-out is required")
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

	if err := chromium.PrepareOutDir(r.out, "*.filter"); err != nil {
		return r.done(errors.Annotate(err, "failed to prepare filter file dir %q", r.out).Err())
	}

	// Do this check only after existing .filter files are deleted.
	if len(r.changedFiles) == 0 {
		logging.Warningf(ctx, "no changed files detected")
		return 0
	}
	r.logChangedFiles(ctx)

	logging.Infof(ctx, "chosen threshold: %f", r.strategy.MaxDistance)

	// Select the tests and write .filter files.
	err := r.writeFilterFiles()
	if disableRTS.In(err) {
		logging.Warningf(ctx, "disabling RTS: %s", err)
		err = nil
	}
	return r.done(err)
}

// writeFilterFiles writes filter files in r.filterFilesDir directory.
func (r *selectRun) writeFilterFiles() error {
	// Maps a test target to the list of tests to skip.
	testsToSkip := map[string][]string{}
	err := r.selectTests(func(testFileToSkip *chromium.TestFile) error {
		for _, target := range testFileToSkip.TestTargets {
			testsToSkip[target] = append(testsToSkip[target], testFileToSkip.TestNames...)
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Write the files.
	for target, testNames := range testsToSkip {
		fileName := filepath.Join(r.out, target+".filter")
		if err := writeFilterFile(fileName, testNames); err != nil {
			return errors.Annotate(err, "failed to write %q", fileName).Err()
		}
		fmt.Printf("wrote %s\n", fileName)
	}
	return nil
}

func (r *selectRun) logChangedFiles(ctx context.Context) {
	msg := &strings.Builder{}
	msg.WriteString("detected changed files:\n")
	for f := range r.changedFiles {
		fmt.Fprintf(msg, "  %s\n", f)
	}
	logging.Infof(ctx, "%s", msg)
}

// testNameReplacer is used to prepare a test name to be used in a .filter file.
var testNameReplacer = strings.NewReplacer(
	// Escape stars, since filter file lines are actually globs.
	"*", "\\*",

	// Java test names use "#" as a separator of class name and method name,
	// but the filter files accept "." instead (probably because comments start
	// with "#"). Thus replace "#" with ".".
	// Note: only Java tests use "#" in their test names.
	"#", ".",
)

func writeFilterFile(fileName string, toSkip []string) error {
	f, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, name := range toSkip {
		name = testNameReplacer.Replace(name)
		if _, err := fmt.Fprintf(f, "-%s\n", name); err != nil {
			return err
		}
	}
	return f.Close()
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
		err := r.loadStrategy(filepath.Join(gitGraphDir, "config.json"))
		return errors.Annotate(err, "failed to load eval results").Err()
	})

	eg.Go(func() (err error) {
		err = r.loadTestFileSet(filepath.Join(r.modelDir, "test-files.jsonl"))
		return errors.Annotate(err, "failed to load test files set").Err()
	})

	eg.Go(func() (err error) {
		err = r.loadChangedFiles()
		return errors.Annotate(err, "failed to load changed files").Err()
	})

	return eg.Wait()
}

// loadStrategy initializes r.strategy fields, except r.strategy.Graph.
func (r *selectRun) loadStrategy(cfgFileName string) error {
	cfgBytes, err := ioutil.ReadFile(cfgFileName)
	if err != nil {
		return err
	}
	cfg := &chromium.GitBasedStrategyConfig{}
	if err := protojson.Unmarshal(cfgBytes, cfg); err != nil {
		return err
	}

	r.strategy.EdgeReader = &git.EdgeReader{
		ChangeLogDistanceFactor:     float64(cfg.ChangeLogDistanceFactor),
		FileStructureDistanceFactor: float64(cfg.FileStructureDistanceFactor),
	}
	threshold := chooseThreshold(cfg.Thresholds, r.targetChangeRecall)
	if threshold == nil {
		return errors.Reason("no threshold for target change recall %.4f", r.targetChangeRecall).Err()
	}
	r.strategy.MaxDistance = float64(threshold.MaxDistance)
	return nil
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

	r.testFiles = map[string]*chromium.TestFile{}
	return chromium.ReadTestFiles(bufio.NewReader(f), func(file *chromium.TestFile) error {
		r.testFiles[file.Path] = file
		return nil
	})
}

// loadChangedFiles initializes r.changedFiles.
func (r *selectRun) loadChangedFiles() error {
	changedFiles, err := gitutil.ChangedFiles(r.checkout, "origin/main")
	if err != nil {
		return err
	}

	r.changedFiles = stringset.New(len(changedFiles))
	for _, f := range changedFiles {
		r.changedFiles.Add("//" + f)
	}
	return nil
}

// chooseThreshold returns the distance threshold based on
// r.targetChangeRecall and r.evalResult.
func chooseThreshold(thresholds []*evalpb.Threshold, targetChangeRecall float64) *evalpb.Threshold {
	var ret *evalpb.Threshold
	for _, t := range thresholds {
		if t.ChangeRecall < float32(targetChangeRecall) {
			continue
		}
		if ret == nil || ret.ChangeRecall > t.ChangeRecall {
			ret = t
		}
	}
	return ret
}
