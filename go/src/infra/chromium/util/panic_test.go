// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package util

import (
	"errors"
	"os/exec"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

func TestPanicIf(t *testing.T) {
	t.Parallel()

	Convey("PanicIf", t, func() {

		Convey("does not panic on false", func() {
			f := func() { PanicIf(false, "test message") }

			So(f, ShouldNotPanic)
		})

		Convey("panics on true", func() {
			f := func() { PanicIf(true, "test message: %s", "foo") }

			So(f, ShouldPanicLike, "test message: foo")
		})

	})
}

func TestPanicOnError(t *testing.T) {
	t.Parallel()

	Convey("PanicOnError", t, func() {

		Convey("does not panic with no error", func() {
			var err error

			f := func() { PanicOnError(err) }

			So(f, ShouldNotPanic)
		})

		Convey("panics on error", func() {
			err := errors.New("test error")

			f := func() { PanicOnError(err) }

			So(f, ShouldPanicLike, "test error")
		})

		Convey("includes stderr for exec.ExitError", func() {
			err := &exec.ExitError{Stderr: []byte("test stderr")}

			f := func() { PanicOnError(err) }

			So(f, ShouldPanicLike, "test stderr")
		})

	})
}
