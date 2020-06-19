// Copyright 2018 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package config

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestRulesConfig(t *testing.T) {
	t.Parallel()
	Convey("Ensure ruleMap keys are valid", t, func() {
		for k := range GetRuleMap() {
			// This is a special value.
			So(k, ShouldNotEqual, "AuditFailure")
			// ":" is used to separate config name from concrete ref
			// when accepting ref patterns.
			So(k, ShouldNotContainSubstring, ":")
		}
	})
}
