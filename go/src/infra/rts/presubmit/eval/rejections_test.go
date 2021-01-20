// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package eval

import (
	"bytes"
	"strings"
	"testing"

	"infra/rts"
	evalpb "infra/rts/presubmit/eval/proto"

	. "github.com/smartystreets/goconvey/convey"
)

func TestPrintLostRejection(t *testing.T) {
	t.Parallel()

	assert := func(rej *evalpb.Rejection, expectedText string) {
		buf := &bytes.Buffer{}
		p := rejectionPrinter{printer: newPrinter(buf)}
		So(p.rejection(rej, rts.Affectedness{Distance: 5, Rank: 3}), ShouldBeNil)
		expectedText = strings.Replace(expectedText, "\t", "  ", -1)
		So(buf.String(), ShouldEqual, expectedText)
	}

	ps1 := &evalpb.GerritPatchset{
		Change: &evalpb.GerritChange{
			Host:    "chromium-review.googlesource.com",
			Project: "chromium/src",
			Number:  123,
		},
		Patchset: 4,
	}
	ps2 := &evalpb.GerritPatchset{
		Change: &evalpb.GerritChange{
			Host:    "chromium-review.googlesource.com",
			Project: "chromium/src",
			Number:  223,
		},
		Patchset: 4,
	}

	Convey(`PrintLostRejection`, t, func() {
		Convey(`Basic`, func() {
			rej := &evalpb.Rejection{
				Patchsets:          []*evalpb.GerritPatchset{ps1},
				FailedTestVariants: []*evalpb.TestVariant{{Id: "test1"}},
			}

			assert(rej, `Rejection:
	Most affected test: 5.000000 distance, 3 rank
	https://chromium-review.googlesource.com/c/123/4
	Failed and not selected tests:
		- <empty test variant>
			in <unknown file>
			  test1
`)
		})

		Convey(`With file name`, func() {
			rej := &evalpb.Rejection{
				Patchsets: []*evalpb.GerritPatchset{ps1},
				FailedTestVariants: []*evalpb.TestVariant{{
					Id:       "test1",
					FileName: "test.cc",
				}},
			}

			assert(rej, `Rejection:
	Most affected test: 5.000000 distance, 3 rank
	https://chromium-review.googlesource.com/c/123/4
	Failed and not selected tests:
		- <empty test variant>
			in test.cc
			  test1
`)
		})

		Convey(`Multiple variants`, func() {
			rej := &evalpb.Rejection{
				Patchsets: []*evalpb.GerritPatchset{ps1},
				FailedTestVariants: []*evalpb.TestVariant{
					{
						Id:      "test1",
						Variant: []string{"a:0"},
					},
					{
						Id:      "test2",
						Variant: []string{"a:0"},
					},
					{
						Id:      "test1",
						Variant: []string{"a:0", "b:0"},
					},
				},
			}

			assert(rej, `Rejection:
	Most affected test: 5.000000 distance, 3 rank
	https://chromium-review.googlesource.com/c/123/4
	Failed and not selected tests:
		- a:0
		  in <unknown file>
			  test1
			  test2
		- a:0 | b:0
		  in <unknown file>
		    test1
`)
		})

		Convey(`Two patchsets`, func() {
			rej := &evalpb.Rejection{
				Patchsets:          []*evalpb.GerritPatchset{ps1, ps2},
				FailedTestVariants: []*evalpb.TestVariant{{Id: "test1"}},
			}

			assert(rej, `Rejection:
	Most affected test: 5.000000 distance, 3 rank
	- patchsets:
		https://chromium-review.googlesource.com/c/123/4
		https://chromium-review.googlesource.com/c/223/4
	Failed and not selected tests:
		- <empty test variant>
			in <unknown file>
			  test1
`)
		})
	})
}
