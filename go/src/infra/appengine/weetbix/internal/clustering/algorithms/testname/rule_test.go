// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package testname

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

func TestRule(t *testing.T) {
	Convey(`Evaluate`, t, func() {
		Convey(`Valid Examples`, func() {
			Convey(`Blink Web Tests`, func() {
				rule := &ClusteringRule{
					Name:         "Blink Web Tests",
					Pattern:      `^ninja://:blink_web_tests/(virtual/[^/]+/)?(?P<testname>([^/]+/)+[^/]+\.[a-zA-Z]+).*$`,
					LikeTemplate: `ninja://:blink\_web\_tests/%${testname}%`,
				}
				eval, err := rule.Compile()
				So(err, ShouldBeNil)

				inputs := []string{
					"ninja://:blink_web_tests/virtual/oopr-canvas2d/fast/canvas/canvas-getImageData.html",
					"ninja://:blink_web_tests/virtual/oopr-canvas2d/fast/canvas/canvas-getImageData.html?param=a",
					"ninja://:blink_web_tests/virtual/oopr-canvas3d/fast/canvas/canvas-getImageData.html?param=b",
					"ninja://:blink_web_tests/fast/canvas/canvas-getImageData.html",
				}
				for _, testname := range inputs {
					like, ok := eval(testname)
					So(ok, ShouldBeTrue)
					So(like, ShouldEqual, `ninja://:blink\_web\_tests/%fast/canvas/canvas-getImageData.html%`)
				}

				_, ok := eval("ninja://:not_blink_web_tests/fast/canvas/canvas-getImageData.html")
				So(ok, ShouldBeFalse)
			})
			Convey(`Google Tests`, func() {
				rule := &ClusteringRule{
					Name: "Google Test (Value-parameterized)",
					// E.g. ninja:{target}/Prefix/ColorSpaceTest.testNullTransform/11
					// Note that "Prefix/" portion may be blank/omitted.
					Pattern:      `^ninja:(?P<target>[\w/]+:\w+)/(\w+/)?(?P<suite>\w+)\.(?P<case>\w+)/\w+$`,
					LikeTemplate: `ninja:${target}/%${suite}.${case}%`,
				}
				eval, err := rule.Compile()
				So(err, ShouldBeNil)

				inputs := []string{
					"ninja://chrome/test:interactive_ui_tests/Name/ColorSpaceTest.testNullTransform/0",
					"ninja://chrome/test:interactive_ui_tests/Name/ColorSpaceTest.testNullTransform/0",
					"ninja://chrome/test:interactive_ui_tests/Name/ColorSpaceTest.testNullTransform/11",
				}
				for _, testname := range inputs {
					like, ok := eval(testname)
					So(ok, ShouldBeTrue)
					So(like, ShouldEqual, "ninja://chrome/test:interactive\\_ui\\_tests/%ColorSpaceTest.testNullTransform%")
				}

				_, ok := eval("ninja://:blink_web_tests/virtual/oopr-canvas2d/fast/canvas/canvas-getImageData.html")
				So(ok, ShouldBeFalse)
			})
		})
		Convey(`Test name escaping in LIKE output`, func() {
			Convey(`Test name is escaped when substituted`, func() {
				rule := &ClusteringRule{
					Name:         "Escape test",
					Pattern:      `^(?P<testname>.*)$`,
					LikeTemplate: `${testname}_%`,
				}
				eval, err := rule.Compile()
				So(err, ShouldBeNil)

				// Verify that the test name is not injected varbatim in the generated
				// like expression, but is escaped to avoid it being interpreted.
				like, ok := eval(`_\%`)
				So(ok, ShouldBeTrue)
				So(like, ShouldEqual, `\_\\\%_%`)
			})
			Convey(`Unsafe LIKE templates are rejected`, func() {
				rule := &ClusteringRule{
					Name:    "Escape test",
					Pattern: `^path\\(?P<testname>.*)$`,
					// The user as incorrectly used an unfinished LIKE escape sequence
					// (with trailing '\') before the testname substitution.
					// If substitution were allowed, this may allow the test name to be
					// interpreted as a LIKE expression instead as literal text.
					// E.g. a test name of `path\%` may yield `path\\%` after template
					// evaluation which invokes the LIKE '%' operator.
					LikeTemplate: `path\${testname}`,
				}
				_, err := rule.Compile()
				So(err, ShouldErrLike, `"path\\" is not a valid standalone LIKE expression: unfinished escape sequence "\" at end of LIKE pattern`)
			})
		})
		Convey(`Substitution operator`, func() {
			Convey(`Dollar sign can be inserted into output`, func() {
				rule := &ClusteringRule{
					Name:         "Insert $",
					Pattern:      `^(?P<testname>.*)$`,
					LikeTemplate: `${testname}$$blah$$$$`,
				}
				eval, err := rule.Compile()
				So(err, ShouldBeNil)

				like, ok := eval(`test`)
				So(ok, ShouldBeTrue)
				So(like, ShouldEqual, `test$blah$$`)
			})
			Convey(`Invalid uses of substitution operator are rejected`, func() {
				rule := &ClusteringRule{
					Name:         "Invalid use of $ (neither $$ or ${name})",
					Pattern:      `^(?P<testname>.*)$`,
					LikeTemplate: `${testname}blah$$$`,
				}
				_, err := rule.Compile()
				So(err, ShouldErrLike, `invalid use of the $ operator at position 17 in "${testname}blah$$$"`)

				rule = &ClusteringRule{
					Name:         "Invalid use of $ (invalid capture group name)",
					Pattern:      `^(?P<testname>.*)$`,
					LikeTemplate: `${template@}blah`,
				}
				_, err = rule.Compile()
				So(err, ShouldErrLike, `invalid use of the $ operator at position 0 in "${template@}blah"`)

				rule = &ClusteringRule{
					Name:         "Capture group name not defined",
					Pattern:      `^(?P<testname>.*)$`,
					LikeTemplate: `${myname}blah`,
				}
				_, err = rule.Compile()
				So(err, ShouldErrLike, `like template contains reference to non-existant capturing group with name "myname"`)
			})
		})
		Convey(`Invalid Pattern`, func() {
			rule := &ClusteringRule{
				Name:         "Invalid Pattern",
				Pattern:      `[`,
				LikeTemplate: ``,
			}
			_, err := rule.Compile()
			So(err, ShouldErrLike, `parsing pattern: error parsing regexp`)
		})
	})
}
