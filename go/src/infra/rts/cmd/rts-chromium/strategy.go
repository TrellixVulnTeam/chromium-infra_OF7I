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
	"infra/rts/presubmit/eval"
)

var neverSkipTestFileRegexp *regexp.Regexp

// bannedTestFileWords is the list of words in test file names that indicate
// that the test file is likely to be unsafe to exclude. For example, it
// contains the main() function, or is dependency of another test file.
var bannedTestFileWords = []string{
	"base",
	"common",
	"helper",
	"launcher",
	"main",
	"run",
	"runner",
	"third_party",
	"util",

	// These are concrete test file names that have main() function.
	"gles2_cmd_decoder_unittest",
	"api_bindings_system_unittest",
	"media_router_integration_browsertest",
	"extension_settings_browsertest",
	"mojo_core_unittest",
	"vaapi_unittest",
}

func init() {
	// Ensure bannedTestFileWords contain only alphanumeric runes, otherwise
	// regexp below won't work correctly.
	nonAlphanumeric := regexp.MustCompile(`\W`)
	for _, w := range bannedTestFileWords {
		if nonAlphanumeric.MatchString(w) {
			panic("bad word: " + w)
		}
	}
	neverSkipTestFileRegexp = regexp.MustCompile(fmt.Sprintf(`(?i)[_\W^](%s)[_\W$]`, strings.Join(bannedTestFileWords, "|")))
}

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

	if err := r.fg.EvalStrategy(ctx, in, out); err != nil {
		return err
	}

	// No matter what filegraph said, never skip certain tests.
	for i, tv := range in.TestVariants {
		if neverSkipTestFileRegexp.MatchString(tv.FileName) {
			out.TestVariantAffectedness[i] = rts.Affectedness{Distance: 0}
		}
	}
	return nil
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
