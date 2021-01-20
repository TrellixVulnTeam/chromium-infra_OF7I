// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"strings"

	"go.chromium.org/luci/common/errors"

	"infra/rts/presubmit/eval"
)

// selectTests calls skipFile for test files that should be skipped.
func (r *selectRun) selectTests(skipFile func(name string) error) (err error) {
	// Check if any of the changed files requires all tests.
	for f := range r.changedFiles {
		if requiresAllTests(f) {
			return nil
		}
	}
	r.strategy.Select(r.changedFiles.ToSlice(), func(fileName string) (keepGoing bool) {
		if !r.testFiles.Has(fileName) {
			return true
		}
		err = skipFile(fileName)
		return err == nil
	})
	return
}

func (r *createModelRun) selectTests(ctx context.Context, in eval.Input, out *eval.Output) error {
	for _, f := range in.ChangedFiles {
		switch {
		case f.Repo != "https://chromium.googlesource.com/chromium/src":
			return errors.Reason("unexpected repo %q", f.Repo).Err()
		case requiresAllTests(f.Path):
			return nil
		}
	}

	return r.fg.EvalStrategy(ctx, in, out)
}

// requiresAllTests returns true if changedFile requires running all tests.
// If a CL changes such a file, RTS gets disabled.
func requiresAllTests(changedFile string) bool {
	switch {
	case strings.HasPrefix(changedFile, "//testing/"):
		// This CL changes the way tests run or their configurations.
		// Run all tests.
		return true

	case changedFile == "//DEPS":
		// The full list of modified files is not available, and the
		// graph does not include DEPSed file changes anyway.
		return true

	default:
		return false
	}
}
