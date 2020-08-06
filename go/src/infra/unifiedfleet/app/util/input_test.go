// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package util

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	ufspb "infra/unifiedfleet/api/v1/proto"
)

func TestUFSStateCoverage(t *testing.T) {
	Convey("test the ufs state mapping covers all UFS state enum", t, func() {
		got := make(map[string]bool, len(StrToUFSState))
		for _, v := range StrToUFSState {
			got[v] = true
		}
		for l := range ufspb.State_value {
			if l == ufspb.State_STATE_UNSPECIFIED.String() {
				continue
			}
			_, ok := got[l]
			So(ok, ShouldBeTrue)
		}
	})

	Convey("test the ufs state mapping doesn't cover any non-UFS state enum", t, func() {
		for _, v := range StrToUFSState {
			_, ok := ufspb.State_value[v]
			So(ok, ShouldBeTrue)
		}
	})
}
