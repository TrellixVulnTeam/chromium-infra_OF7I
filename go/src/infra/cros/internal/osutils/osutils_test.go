// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
// +build linux

package osutils

import (
	"strings"
	"testing"

	"infra/cros/internal/assert"
)

func TestFindInPathParents(t *testing.T) {
	start_path := "test_data/a/b/c"
	end_path := "test_data"

	var target_path, res string
	target_path = "hello.txt"

	// FindInPathParents finds test_data/a/b/hello.txt.
	res = FindInPathParents(target_path, start_path, end_path, PathExists)
	assert.Assert(t, strings.HasSuffix(res, "test_data/a/b/hello.txt"))
	// FindInPathParents finds test_data/a/b/hello.txt when given a filepath
	// as a starting path.

	full_path := "test_data/a/b/hello.txt"
	res = FindInPathParents(target_path, full_path, end_path, PathExists)
	assert.Assert(t, strings.HasSuffix(res, "test_data/a/b/hello.txt"))

	// FileInPathParents does not find test_data/a/b/hello.txt when passed
	// DirExists (which checks for directories)
	res = FindInPathParents(target_path, start_path, end_path, DirExists)
	assert.StringsEqual(t, res, "")

	// FileInPathParents finds directory test_data/a/b/my_dir.
	target_path = "my_dir"
	res = FindInPathParents(target_path, start_path, end_path, DirExists)
	assert.Assert(t, strings.HasSuffix(res, "test_data/a/b/my_dir"))

	// FileInPathParents finds file goodbye.txt with path my_dir/goodbye.txt.
	target_path = "my_dir/goodbye.txt"
	res = FindInPathParents(target_path, start_path, end_path, PathExists)
	assert.Assert(t, strings.HasSuffix(res, "test_data/a/b/my_dir/goodbye.txt"))

	// FileInPathParents does not find file goodbye.txt because the path
	// does not exist off of any parent directory of start_path.
	target_path = "goodbye.txt"
	res = FindInPathParents(target_path, start_path, end_path, PathExists)
	assert.StringsEqual(t, res, "")

	// FindInPathParents does not find foo.txt because it stops looking at
	// end_path.
	target_path = "foo.txt"
	res = FindInPathParents(target_path, start_path, end_path, PathExists)
	assert.Assert(t, strings.HasSuffix(res, "test_data/a/foo.txt"))
	res = FindInPathParents(target_path, start_path, "test_data/a/b", PathExists)
	assert.StringsEqual(t, res, "")
}
