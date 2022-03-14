// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package util

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestExtractorUtil(t *testing.T) {
	Convey("NormalizeFilePath", t, func() {
		data := map[string]string{
			"../a/b/c.cc":    "a/b/c.cc",
			"a/b/./c.cc":     "a/b/c.cc",
			"a/b/../c.cc":    "a/c.cc",
			"a\\b\\.\\c.cc":  "a/b/c.cc",
			"a\\\\b\\\\c.cc": "a/b/c.cc",
			"//a/b/c.cc":     "a/b/c.cc",
		}
		for fp, nfp := range data {
			So(NormalizeFilePath(fp), ShouldEqual, nfp)
		}
	})
}
