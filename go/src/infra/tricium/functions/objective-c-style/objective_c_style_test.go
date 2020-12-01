// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"infra/tricium/api/v1"
)

// These tests read from files on the filesystem, so modifying the tests may
// require modifying the example test files.
const (
	good string = "test/src/good_get.mm"
	bad  string = "test/src/bad_get.mm"
)

func TestGetPrefix(t *testing.T) {

	Convey("Produces no comment for file with correct function names", t, func() {
		So(checkGetPrefix("", good), ShouldBeNil)
	})

	Convey("Flags functions have unnecessary get prefixes", t, func() {
		c := checkGetPrefix("", bad)
		So(c, ShouldNotBeNil)
		So(c, ShouldResemble, []*tricium.Data_Comment{
			{
				Category:  "ObjectiveCStyle/Get",
				Message:   "The use of \"get\" is unnecessary, unless one or more values are returned indirectly. See: https://developer.apple.com/library/archive/documentation/Cocoa/Conceptual/CodingGuidelines/Articles/NamingMethods.html#:~:text=The%20use%20of%20%22get%22%20is%20unnecessary,%20unless%20one%20or%20more%20values%20are%20returned%20indirectly.",
				Path:      "test/src/bad_get.mm",
				StartLine: 4,
				EndLine:   4,
				StartChar: 0,
				EndChar:   21,
			},
			{
				Category:  "ObjectiveCStyle/Get",
				Message:   "The use of \"get\" is unnecessary, unless one or more values are returned indirectly. See: https://developer.apple.com/library/archive/documentation/Cocoa/Conceptual/CodingGuidelines/Articles/NamingMethods.html#:~:text=The%20use%20of%20%22get%22%20is%20unnecessary,%20unless%20one%20or%20more%20values%20are%20returned%20indirectly.",
				Path:      "test/src/bad_get.mm",
				StartLine: 8,
				EndLine:   8,
				StartChar: 0,
				EndChar:   20,
			},
			{
				Category:  "ObjectiveCStyle/Get",
				Message:   "The use of \"get\" is unnecessary, unless one or more values are returned indirectly. See: https://developer.apple.com/library/archive/documentation/Cocoa/Conceptual/CodingGuidelines/Articles/NamingMethods.html#:~:text=The%20use%20of%20%22get%22%20is%20unnecessary,%20unless%20one%20or%20more%20values%20are%20returned%20indirectly.",
				Path:      "test/src/bad_get.mm",
				StartLine: 12,
				EndLine:   12,
				StartChar: 0,
				EndChar:   41,
			},
		})
	})
}
