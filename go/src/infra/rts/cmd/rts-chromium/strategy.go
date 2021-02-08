// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"path"
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
	// Skip C++ files that may have main() function or be a dependency of another test.
	case cppFileRegexp.MatchString(testFile) && alwaysRunCPPBaseTestFileRegexp.MatchString(path.Base(testFile)):
		return true

	// Always run all third-party tests (never skip them),
	// except //third_party/blink which is actually first party.
	case strings.Contains(testFile, "/third_party/") && !strings.HasPrefix(testFile, "//third_party/blink/"):
		return true

	default:
		return false
	}
}

var (
	requireAllTestsRegexp          *regexp.Regexp
	alwaysRunCPPBaseTestFileRegexp *regexp.Regexp
	// This list of extensions is derived from
	// https://source.chromium.org/search?q=-f:%5C.(cpp%7Cc%7Ccc%7Ccxx%7Ch%7Chh%7Chpp%7Chxx%7Cmm%7Cm%7Cinc)$%20lang:cpp%20-f:third_party%20-f:aosp
	// The fact that this query returns 0 results means the list of extensions
	// is exhaustive.
	cppFileRegexp = regexp.MustCompile(`(?i)\.(cpp|c|cc|cxx|h|hh|hpp|hxx|mm|m|inc)$`)
)

// requireAllTests is a list of patterns of files that require running all
// tests.
var requireAllTests = []string{
	// A CL changes the way tests run or their configurations.
	"//testing/.+",

	// The full list of modified files is not available, and the
	// graph does not include DEPSed file changes anyway.
	"//DEPS",
}

// bannedCPPBaseTestFileWords is the list of words in a test's filename
// without directory name (AKA base name of a test file) that indicate that
// the test file is likely to be unsafe to exclude. For example, it contains the
// main() function, or is dependency of another test file.
var bannedCPPBaseTestFileWords = []string{
	"base",
	"common",
	"helper",
	"launcher",
	"main",
	"run",
	"runner",
	"util",

	// These are concrete test file names that have main() function.
	"accessibility_selection_test",
	"accessibility_test",
	"api_bindings_system_unittest",
	"async_gen",
	"ax_platform_node_win_unittest",
	"browser_actions_bar_browsertest",
	"browser_navigator_browsertest",
	"cert_issuer_source_sync_unittest",
	"component_patcher_unittest",
	"cookie_store_change_unittest",
	"cookie_store_unittest",
	"dedicated_worker_test",
	"default_keyboard_extension_browser_test",
	"download_browsertest",
	"extension_prefs_unittest",
	"extension_settings_browsertest",
	"file_stream_reader_test",
	"file_stream_writer_test",
	"ftp_directory_listing_parser_unittest",
	"fullscreen_controller_state_tests",
	"gles2_cmd_decoder_unittest",
	"gles2_implementation_unittest_autogen",
	"gl_image_test_template",
	"google_api_keys_unittest",
	"gpu_context_tests",
	"gurl_abstract_tests",
	"headless_protocol_browsertest",
	"help_app_ui_browsertest",
	"hit_testing_browsertest",
	"identity_internals_ui_browsertest",
	"integration_tests",
	"invalidation_service_test_template",
	"ipc_sync_message_unittest",
	"is_potentially_trustworthy_unittest",
	"keyboard_lock_browsertest",
	"media_app_ui_browsertest",
	"media_browsertest",
	"media_router_integration_browsertest",
	"mojo_core_unittest",
	"origin_abstract_tests",
	"paint_and_raster_invalidation_test",
	"paint_controller_paint_test",
	"paint_controller_test",
	"paint_property_tree_builder_test",
	"password_store_origin_unittest",
	"pointer_lock_browsertest",
	"profile_info_cache_unittest",
	"raster_implementation_unittest_autogen",
	"rtc_ice_transport_test",
	"site_per_process_browsertest",
	"spoken_feedback_browsertest",
	"sync_policy_test",
	"system_web_app_manager_browsertest",
	"tab_drag_controller_interactive_uitest",
	"tab_modal_confirm_dialog_browsertest",
	"telemetry_extension_ui_browsertest",
	"toolbar_actions_bar_unittest",
	"trust_token_browsertest",
	"uri_unittest",
	"vaapi_unittest",
	"webgpu_test",
	"web_view_apitest",
}

func init() {
	requireAllTestsRegexp = regexp.MustCompile(fmt.Sprintf("^(%s)$", strings.Join(requireAllTests, "|")))

	// Ensure bannedTestFileWords contain only alphanumeric runes, otherwise
	// regexp below won't work correctly.
	nonAlphanumeric := regexp.MustCompile(`\W`)
	for _, w := range bannedCPPBaseTestFileWords {
		if nonAlphanumeric.MatchString(w) {
			panic("bad word: " + w)
		}
	}
	alwaysRunCPPBaseTestFileRegexp = regexp.MustCompile(fmt.Sprintf(`(?i)(^|[_\W])(%s)($|[_\W])`, strings.Join(bannedCPPBaseTestFileWords, "|")))
}

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
