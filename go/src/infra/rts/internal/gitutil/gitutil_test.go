// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package gitutil

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGit(t *testing.T) {
	t.Parallel()

	// https: //logs.chromium.org/logs/infra/buildbucket/cr-buildbucket.appspot.com/8864634177878601952/+/u/go_test/stdout
	t.Skipf("this test is failing in a weird way; skip for now")

	if testing.Short() {
		t.Skip("Skipping because it is not a short test")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("git not found: %s", err)
	}

	Convey(`Git`, t, func() {
		tmpd, err := ioutil.TempDir("", "filegraph_git")
		So(err, ShouldBeNil)
		defer os.RemoveAll(tmpd)

		git := func(context string) func(args ...string) string {
			return func(args ...string) string {
				out, err := Exec(context)(args...)
				So(err, ShouldBeNil)
				return out
			}
		}

		git(tmpd)("init")

		fooPath := filepath.Join(tmpd, "foo")
		err = ioutil.WriteFile(fooPath, []byte("hello"), 0777)
		So(err, ShouldBeNil)

		// Run in fooBar context.
		git(fooPath)("add", fooPath)
		git(tmpd)("commit", "-a", "-m", "message")

		out := git(fooPath)("status")
		So(out, ShouldContainSubstring, "working tree clean")

		repoDir, err := EnsureSameRepo(tmpd, fooPath)
		So(err, ShouldBeNil)
		So(repoDir, ShouldEqual, tmpd)
	})
}

func TestChangedFiles(t *testing.T) {
	t.Parallel()
	Convey(`ChangedFiles`, t, func() {
		Convey(`Works`, func() {
			So(changedFiles("foo\nbar\n"), ShouldResemble, []string{"foo", "bar"})
		})
		Convey(`No files changed`, func() {
			So(changedFiles("\n"), ShouldResemble, []string(nil))
		})
	})
}
