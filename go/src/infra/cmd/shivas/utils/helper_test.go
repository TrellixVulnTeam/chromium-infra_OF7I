// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package utils

import (
	"flag"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGetUpdateMask(t *testing.T) {
	Convey("test flags - happy path", t, func() {
		f := &flag.FlagSet{}
		var testTrue, testFalse bool
		var testStrExist, testStrEmtpy string
		f.BoolVar(&testTrue, "test-true", false, "")
		f.BoolVar(&testFalse, "test-false", false, "")
		f.StringVar(&testStrExist, "test-str-exist", "", "")
		f.StringVar(&testStrEmtpy, "test-str-empty", "", "")
		f.Set("test-true", "true")
		f.Set("test-str-exist", "test")
		paths := map[string]string{
			"test-true":      "mask-bool-true",
			"test-false":     "mask-bool-false",
			"test-str-exist": "mask-str-exist",
			"test-str-empty": "mask-str-empty",
		}
		mask := GetUpdateMask(f, paths)
		So(mask.Paths, ShouldResemble, []string{"mask-bool-true", "mask-str-exist"})
	})

	Convey("test flags - duplicated paths", t, func() {
		f := &flag.FlagSet{}
		var testTrue, testTrue2 bool
		var testStrExist, testStrEmtpy string
		f.BoolVar(&testTrue, "test-true", false, "")
		f.BoolVar(&testTrue2, "test-true2", false, "")
		f.StringVar(&testStrExist, "test-str-exist", "", "")
		f.StringVar(&testStrEmtpy, "test-str-empty", "", "")
		f.Set("test-true", "true")
		f.Set("test-true2", "true")
		f.Set("test-str-exist", "test")
		paths := map[string]string{
			"test-true":      "mask-bool",
			"test-true2":     "mask-bool",
			"test-str-exist": "mask-str-exist",
			"test-str-empty": "mask-str-empty",
		}
		mask := GetUpdateMask(f, paths)
		So(mask.Paths, ShouldResemble, []string{"mask-bool", "mask-str-exist"})
	})
}
