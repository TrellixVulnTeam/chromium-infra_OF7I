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
	goodGet      string = "test/src/good_get.mm"
	badGet       string = "test/src/bad_get.mm"
	goodDelegate string = "test/src/good_delegate.mm"
	badDelegate  string = "test/src/bad_delegate.mm"
	goodProperty string = "test/src/good_property.mm"
	badProperty  string = "test/src/bad_property.mm"
)

func TestGetPrefix(t *testing.T) {

	Convey("Produces no comment for file with correct function names", t, func() {
		So(checkSourceFile("", goodGet), ShouldBeNil)
	})

	Convey("Produces no comment for file with correct delegate specifiers", t, func() {
		So(checkSourceFile("", goodDelegate), ShouldBeNil)
	})

	Convey("Produces no comment for file with correct delegate specifiers", t, func() {
		So(checkSourceFile("", goodProperty), ShouldBeNil)
	})

	Convey("Flags strong delegates", t, func() {
		c := checkSourceFile("", badDelegate)
		So(c, ShouldNotBeNil)
		So(c, ShouldResemble, []*tricium.Data_Comment{

			{
				Category:  "ObjectiveCStyle/StrongDelegate",
				Message:   "In Objective-C delegates are normally weak. Strong delegates can cause retain cycles.",
				Path:      "test/src/bad_delegate.mm",
				StartLine: 4,
				EndLine:   4,
				StartChar: 0,
				EndChar:   43,
			},

			{
				Category:  "ObjectiveCStyle/StrongDelegate",
				Message:   "In Objective-C delegates are normally weak. Strong delegates can cause retain cycles.",
				Path:      "test/src/bad_delegate.mm",
				StartLine: 5,
				EndLine:   5,
				StartChar: 0,
				EndChar:   53,
			},

			{
				Category:  "ObjectiveCStyle/StrongDelegate",
				Message:   "In Objective-C delegates are normally weak. Strong delegates can cause retain cycles.",
				Path:      "test/src/bad_delegate.mm",
				StartLine: 6,
				EndLine:   6,
				StartChar: 0,
				EndChar:   54,
			},

			{
				Category:  "ObjectiveCStyle/StrongDelegate",
				Message:   "In Objective-C delegates are normally weak. Strong delegates can cause retain cycles.",
				Path:      "test/src/bad_delegate.mm",
				StartLine: 7,
				EndLine:   7,
				StartChar: 0,
				EndChar:   51,
			},
			{
				Category:  "ObjectiveCStyle/StrongDelegate",
				Message:   "In Objective-C delegates are normally weak. Strong delegates can cause retain cycles.",
				Path:      "test/src/bad_delegate.mm",
				StartLine: 8,
				EndLine:   8,
				StartChar: 0,
				EndChar:   54,
			},
			{
				Category:  "ObjectiveCStyle/StrongDelegate",
				Message:   "In Objective-C delegates are normally weak. Strong delegates can cause retain cycles.",
				Path:      "test/src/bad_delegate.mm",
				StartLine: 9,
				EndLine:   9,
				StartChar: 0,
				EndChar:   43,
			},
			{
				Category:  "ObjectiveCStyle/StrongDelegate",
				Message:   "In Objective-C delegates are normally weak. Strong delegates can cause retain cycles.",
				Path:      "test/src/bad_delegate.mm",
				StartLine: 10,
				EndLine:   10,
				StartChar: 0,
				EndChar:   46,
			},
			{
				Category:  "ObjectiveCStyle/StrongDelegate",
				Message:   "In Objective-C delegates are normally weak. Strong delegates can cause retain cycles.",
				Path:      "test/src/bad_delegate.mm",
				StartLine: 11,
				EndLine:   11,
				StartChar: 0,
				EndChar:   46,
			},
			{
				Category:  "ObjectiveCStyle/StrongDelegate",
				Message:   "In Objective-C delegates are normally weak. Strong delegates can cause retain cycles.",
				Path:      "test/src/bad_delegate.mm",
				StartLine: 12,
				EndLine:   12,
				StartChar: 0,
				EndChar:   45,
			},
			{
				Category:  "ObjectiveCStyle/StrongDelegate",
				Message:   "In Objective-C delegates are normally weak. Strong delegates can cause retain cycles.",
				Path:      "test/src/bad_delegate.mm",
				StartLine: 13,
				EndLine:   13,
				StartChar: 0,
				EndChar:   50,
			},
			{
				Category:  "ObjectiveCStyle/ExplicitOwnership",
				Message:   "Consider using an explicit ownership specifier. The default is strong, which can cause retain cycles.",
				Path:      "test/src/bad_delegate.mm",
				StartLine: 9,
				EndLine:   9,
				StartChar: 0,
				EndChar:   43,
			},
			{
				Category:  "ObjectiveCStyle/ExplicitOwnership",
				Message:   "Consider using an explicit ownership specifier. The default is strong, which can cause retain cycles.",
				Path:      "test/src/bad_delegate.mm",
				StartLine: 10,
				EndLine:   10,
				StartChar: 0,
				EndChar:   46,
			},
			{
				Category:  "ObjectiveCStyle/ExplicitOwnership",
				Message:   "Consider using an explicit ownership specifier. The default is strong, which can cause retain cycles.",
				Path:      "test/src/bad_delegate.mm",
				StartLine: 11,
				EndLine:   11,
				StartChar: 0,
				EndChar:   46,
			},
			{
				Category:  "ObjectiveCStyle/ExplicitOwnership",
				Message:   "Consider using an explicit ownership specifier. The default is strong, which can cause retain cycles.",
				Path:      "test/src/bad_delegate.mm",
				StartLine: 12,
				EndLine:   12,
				StartChar: 0,
				EndChar:   45,
			},
			{
				Category:  "ObjectiveCStyle/ExplicitOwnership",
				Message:   "Consider using an explicit ownership specifier. The default is strong, which can cause retain cycles.",
				Path:      "test/src/bad_delegate.mm",
				StartLine: 13,
				EndLine:   13,
				StartChar: 0,
				EndChar:   50,
			},
		})
	})

	Convey("Flags properties without explicit ownership", t, func() {
		c := checkSourceFile("", badProperty)
		So(c, ShouldNotBeNil)
		So(c, ShouldResemble, []*tricium.Data_Comment{

			{
				Category:  "ObjectiveCStyle/ExplicitOwnership",
				Message:   "Consider using an explicit ownership specifier. The default is strong, which can cause retain cycles.",
				Path:      "test/src/bad_property.mm",
				StartLine: 4,
				EndLine:   4,
				StartChar: 0,
				EndChar:   22,
			},
			{
				Category:  "ObjectiveCStyle/ExplicitOwnership",
				Message:   "Consider using an explicit ownership specifier. The default is strong, which can cause retain cycles.",
				Path:      "test/src/bad_property.mm",
				StartLine: 5,
				EndLine:   5,
				StartChar: 0,
				EndChar:   19,
			},
			{
				Category:  "ObjectiveCStyle/ExplicitOwnership",
				Message:   "Consider using an explicit ownership specifier. The default is strong, which can cause retain cycles.",
				Path:      "test/src/bad_property.mm",
				StartLine: 6,
				EndLine:   6,
				StartChar: 0,
				EndChar:   27,
			},
			{
				Category:  "ObjectiveCStyle/ExplicitOwnership",
				Message:   "Consider using an explicit ownership specifier. The default is strong, which can cause retain cycles.",
				Path:      "test/src/bad_property.mm",
				StartLine: 7,
				EndLine:   7,
				StartChar: 0,
				EndChar:   30,
			},
			{
				Category:  "ObjectiveCStyle/ExplicitOwnership",
				Message:   "Consider using an explicit ownership specifier. The default is strong, which can cause retain cycles.",
				Path:      "test/src/bad_property.mm",
				StartLine: 8,
				EndLine:   8,
				StartChar: 0,
				EndChar:   29,
			},
			{
				Category:  "ObjectiveCStyle/ExplicitOwnership",
				Message:   "Consider using an explicit ownership specifier. The default is strong, which can cause retain cycles.",
				Path:      "test/src/bad_property.mm",
				StartLine: 9,
				EndLine:   9,
				StartChar: 0,
				EndChar:   44,
			},
		})
	})
	Convey("Flags functions have unnecessary get prefixes", t, func() {
		c := checkSourceFile("", badGet)
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

			{
				Category:  "ObjectiveCStyle/Get",
				Message:   "The use of \"get\" is unnecessary, unless one or more values are returned indirectly. See: https://developer.apple.com/library/archive/documentation/Cocoa/Conceptual/CodingGuidelines/Articles/NamingMethods.html#:~:text=The%20use%20of%20%22get%22%20is%20unnecessary,%20unless%20one%20or%20more%20values%20are%20returned%20indirectly.",
				Path:      "test/src/bad_get.mm",
				StartLine: 16,
				EndLine:   16,
				StartChar: 0,
				EndChar:   21,
			},
		})
	})
}
