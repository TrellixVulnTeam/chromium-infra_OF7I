// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package model

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestAddLine(t *testing.T) {
	Convey("Add line or file path", t, func() {
		signal := &CompileFailureSignal{}
		signal.AddLine("a/b", 12)
		So(signal.Files, ShouldResemble, map[string][]int{"a/b": {12}})
		signal.AddLine("a/b", 14)
		So(signal.Files, ShouldResemble, map[string][]int{"a/b": {12, 14}})
		signal.AddLine("c/d", 8)
		So(signal.Files, ShouldResemble, map[string][]int{"a/b": {12, 14}, "c/d": {8}})
		signal.AddLine("a/b", 14)
		So(signal.Files, ShouldResemble, map[string][]int{"a/b": {12, 14}, "c/d": {8}})
		signal.AddFilePath("x/y")
		So(signal.Files, ShouldResemble, map[string][]int{"a/b": {12, 14}, "c/d": {8}, "x/y": {}})
		signal.AddFilePath("x/y")
		So(signal.Files, ShouldResemble, map[string][]int{"a/b": {12, 14}, "c/d": {8}, "x/y": {}})
	})
}
