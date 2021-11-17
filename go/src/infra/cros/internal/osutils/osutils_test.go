// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
//go:build linux
// +build linux

package osutils

import (
	"os"
	"os/user"
	"path/filepath"
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

func TestResolveHomeRelPath(t *testing.T) {
	usr, err := user.Current()
	assert.NilError(t, err)

	got, err := ResolveHomeRelPath("~")
	assert.NilError(t, err)
	assert.StringsEqual(t, got, usr.HomeDir)

	got, err = ResolveHomeRelPath("~/foo")
	assert.NilError(t, err)
	assert.StringsEqual(t, got, filepath.Join(usr.HomeDir, "foo"))

	got, err = ResolveHomeRelPath("foo/~/foo")
	assert.NilError(t, err)
	assert.StringsEqual(t, got, "foo/~/foo")
}

func TestRecursiveChmod(t *testing.T) {
	testDir := t.TempDir()

	if err := os.MkdirAll(filepath.Join(testDir, "a", "b"), 0775); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(testDir, "test1.txt"), []byte("abc"), 0773); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(testDir, "a", "b", "test2.txt"), []byte("def"), 0770); err != nil {
		t.Fatal(err)
	}

	if err := RecursiveChmod(testDir, 0777); err != nil {
		t.Fatal(err)
	}

	filesToCheck := []string{
		filepath.Join(testDir, "test1.txt"),
		filepath.Join(testDir, "a", "b", "test2.txt"),
	}

	dirsToCheck := []string{
		testDir,
		filepath.Join(testDir, "a"),
		filepath.Join(testDir, "a", "b"),
	}

	for _, path := range filesToCheck {
		fileInfo, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}

		if fileInfo.Mode() != 0777 {
			t.Errorf("path %q has mode %d, want 0777", path, fileInfo.Mode())
		}
	}

	for _, path := range dirsToCheck {
		fileInfo, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}

		expectedMode := (os.ModeDir | 0777)

		if fileInfo.Mode() != expectedMode {
			t.Errorf("path %q has mode %d, want %d", path, fileInfo.Mode(), expectedMode)
		}
	}
}

func TestRecursiveChmodErrors(t *testing.T) {
	err := RecursiveChmod("invalidroot", os.ModePerm)
	if err == nil {
		t.Fatal("expected error from RecursiveChmod")
	}

	if err.Error() != "lstat invalidroot: no such file or directory" {
		t.Fatalf("got error %q, expected \"lstat invalidroot: no such file or directory\"", err)
	}
}
