// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package eval

import (
	"testing"

	evalpb "infra/rts/presubmit/eval/proto"

	. "github.com/smartystreets/goconvey/convey"
)

func TestPSURL(t *testing.T) {
	t.Parallel()
	Convey(`psURL`, t, func() {
		patchSet := &evalpb.GerritPatchset{
			Change: &evalpb.GerritChange{
				Host:   "example.googlesource.com",
				Number: 123,
			},
			Patchset: 4,
		}
		So(psURL(patchSet), ShouldEqual, "https://example.googlesource.com/c/123/4")
	})
}
