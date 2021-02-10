// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"go.chromium.org/luci/common/errors"

	"infra/rts"
	"infra/rts/filegraph/git"
	"infra/rts/presubmit/eval"
)

// mustAlwaysRunTest returns true if the test file must never be skipped.
func mustAlwaysRunTest(testFile string) bool {
	switch {
	// Always run all third-party tests (never skip them),
	// except //third_party/blink which is actually first party.
	case strings.Contains(testFile, "/third_party/") && !strings.HasPrefix(testFile, "//third_party/blink/"):
		return true

	default:
		return false
	}
}

var (
	// requireAllTests is a list of patterns of files that require running all
	// tests.
	requireAllTests = []string{
		// A CL changes the way tests run or their configurations.
		"//testing/.+",

		// The full list of modified files is not available, and the
		// graph does not include DEPSed file changes anyway.
		"//DEPS",
	}
	requireAllTestsRegexp = regexp.MustCompile(fmt.Sprintf("^(%s)$", strings.Join(requireAllTests, "|")))
)

// selectTests calls skipFile for test files that should be skipped.
func (r *selectRun) selectTests(skipFile func(*TestFile) error) (err error) {
	// Check if any of the changed files requires all tests.
	for f := range r.changedFiles {
		if requireAllTestsRegexp.MatchString(f) {
			return nil
		}
	}
	r.strategy.Select(r.changedFiles.ToSlice(), func(fileName string) (keepGoing bool) {
		file, ok := r.testFiles[fileName]
		if !ok {
			return true
		}
		err = skipFile(file)
		return err == nil
	})
	return
}

func (r *createModelRun) evalStrategy(er *git.EdgeReader) eval.Strategy {
	fgStrategy := r.fg.EvalStrategy(er)
	return func(ctx context.Context, in eval.Input, out *eval.Output) error {
		for _, f := range in.ChangedFiles {
			switch {
			case f.Repo != "https://chromium.googlesource.com/chromium/src":
				return errors.Reason("unexpected repo %q", f.Repo).Err()
			case requireAllTestsRegexp.MatchString(f.Path):
				return nil
			}
		}

		if err := fgStrategy(ctx, in, out); err != nil {
			return err
		}

		// No matter what filegraph said, never skip certain tests.
		for i, tv := range in.TestVariants {
			if mustAlwaysRunTest(tv.FileName) {
				out.TestVariantAffectedness[i] = rts.Affectedness{Distance: 0}
			}
		}
		return nil
	}
}
