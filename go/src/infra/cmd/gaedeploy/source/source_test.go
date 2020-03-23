// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package source

import (
	"bytes"
	"context"
	"crypto/sha256"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

func TestSource(t *testing.T) {
	t.Parallel()

	Convey("gs://...", t, func() {
		Convey("Works", func() {
			src, err := New("gs://stuff", strings.Repeat("a", 64))
			So(err, ShouldBeNil)
			So(src, ShouldResemble, &gsSource{path: "gs://stuff", sha256: bytes.Repeat([]byte{170}, 32)})
			So(src.SHA256(), ShouldResemble, bytes.Repeat([]byte{170}, 32))
		})

		Convey("Wants hash", func() {
			_, err := New("gs://stuff", "")
			So(err, ShouldErrLike, "-tarball-sha256 is required")
		})

		Convey("Bad digest format", func() {
			_, err := New("gs://stuff", "ZZZ")
			So(err, ShouldErrLike, "not hex")

			_, err = New("gs://stuff", "aaaa")
			So(err, ShouldErrLike, "wrong length")
		})
	})

	Convey("Local file", t, func() {
		Convey("Missing", func() {
			_, err := New("missing_file", "")
			So(err, ShouldErrLike, "can't open the file")
		})

		Convey("Present", func() {
			f, err := ioutil.TempFile("", "gaedeploy_test")
			So(err, ShouldBeNil)
			defer os.Remove(f.Name())

			_, err = f.Write([]byte("boo"))
			So(err, ShouldBeNil)
			So(f.Close(), ShouldBeNil)

			expected := sha256.New()
			expected.Write([]byte("boo"))

			fs, err := New(f.Name(), "")
			So(err, ShouldBeNil)
			So(fs, ShouldResemble, &fileSource{
				path:   f.Name(),
				sha256: expected.Sum(nil),
			})
			So(fs.SHA256(), ShouldResemble, expected.Sum(nil))

			rc, err := fs.Open(context.Background(), "unused")
			So(err, ShouldBeNil)
			blob, err := ioutil.ReadAll(rc)
			So(err, ShouldBeNil)
			So(blob, ShouldResemble, []byte("boo"))
			So(rc.Close(), ShouldBeNil)
		})
	})
}
