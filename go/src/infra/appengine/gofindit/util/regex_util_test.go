// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package util

import (
	"regexp"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestRegexUtil(t *testing.T) {
	Convey("MatchedNamedGroup", t, func() {
		pattern := regexp.MustCompile(`(?P<g1>test1)(test2)(?P<g2>test3)`)
		matches, err := MatchedNamedGroup(pattern, `test1test2test3`)
		So(err, ShouldBeNil)
		So(matches, ShouldResemble, map[string]string{
			"g1": "test1",
			"g2": "test3",
		})
		_, err = MatchedNamedGroup(pattern, `test`)
		So(err, ShouldNotBeNil)
	})
}
