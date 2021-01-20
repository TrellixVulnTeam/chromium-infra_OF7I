// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package git

import (
	"context"
	"math"
	"testing"

	"infra/rts"
	"infra/rts/presubmit/eval"
	evalpb "infra/rts/presubmit/eval/proto"

	. "github.com/smartystreets/goconvey/convey"
)

func TestEvalStrategy(t *testing.T) {
	t.Parallel()

	Convey(`apply`, t, func() {
		ctx := context.Background()

		g := &Graph{}
		g.ensureInitialized()

		applyChanges := func(changes []fileChange) {
			err := g.apply(changes, 100)
			So(err, ShouldBeNil)
		}

		applyChanges([]fileChange{
			{Path: "a", Status: 'A'},
		})
		applyChanges([]fileChange{
			{Path: "a", Status: 'M'},
			{Path: "b", Status: 'M'},
		})
		applyChanges([]fileChange{
			{Path: "b", Status: 'M'},
			{Path: "c/d", Status: 'A'},
		})
		applyChanges([]fileChange{
			{Path: "unreachable", Status: 'A'},
		})

		assertAffectedness := func(in eval.Input, expectedDistance float64, expectedRank int) {
			out := &eval.Output{
				TestVariantAffectedness: make([]rts.Affectedness, 1),
			}
			err := g.EvalStrategy(ctx, in, out)
			So(err, ShouldBeNil)
			af := out.TestVariantAffectedness[0]
			if math.IsInf(expectedDistance, 1) {
				So(math.IsInf(af.Distance, 1), ShouldBeTrue)
			} else {
				So(af.Distance, ShouldAlmostEqual, expectedDistance)
			}
			So(af.Rank, ShouldEqual, expectedRank)
		}

		Convey(`a -> b`, func() {
			in := eval.Input{
				ChangedFiles: []*evalpb.SourceFile{
					{Path: "//a"},
				},
				TestVariants: []*evalpb.TestVariant{
					{FileName: "//b"},
				},
			}
			assertAffectedness(in, -math.Log(0.5), 2)
		})

		Convey(`a -> unrechable`, func() {
			in := eval.Input{
				ChangedFiles: []*evalpb.SourceFile{
					{Path: "//a"},
				},
				TestVariants: []*evalpb.TestVariant{
					{FileName: "//unreachable"},
				},
			}
			assertAffectedness(in, math.Inf(1), math.MaxInt32)
		})

		Convey(`Unknown test`, func() {
			in := eval.Input{
				ChangedFiles: []*evalpb.SourceFile{
					{Path: "//a"},
				},
				TestVariants: []*evalpb.TestVariant{
					{FileName: "//unknown"},
				},
			}
			assertAffectedness(in, 0, 0)
		})

		Convey(`New test`, func() {
			in := eval.Input{
				ChangedFiles: []*evalpb.SourceFile{
					{Path: "//new_test"},
				},
				TestVariants: []*evalpb.TestVariant{
					{FileName: "//new_test"},
				},
			}
			assertAffectedness(in, 0, 0)
		})

		Convey(`One of tests is unknown`, func() {
			in := eval.Input{
				ChangedFiles: []*evalpb.SourceFile{
					{Path: "//a"},
				},
				TestVariants: []*evalpb.TestVariant{
					{FileName: "//b"},
					{FileName: "//unknown"},
				},
			}
			out := &eval.Output{
				TestVariantAffectedness: make([]rts.Affectedness, 2),
			}
			err := g.EvalStrategy(ctx, in, out)
			So(err, ShouldBeNil)
			So(out.TestVariantAffectedness[0].Distance, ShouldAlmostEqual, -math.Log(0.5))
			So(out.TestVariantAffectedness[0].Rank, ShouldEqual, 2)
			So(out.TestVariantAffectedness[1].Distance, ShouldEqual, 0)
			So(out.TestVariantAffectedness[1].Rank, ShouldEqual, 0)
		})

		Convey(`Test without a file name`, func() {
			in := eval.Input{
				ChangedFiles: []*evalpb.SourceFile{
					{Path: "//a"},
				},
				TestVariants: []*evalpb.TestVariant{
					{},
				},
			}
			assertAffectedness(in, 0, 0)
		})

		Convey(`Unknown changed file`, func() {
			in := eval.Input{
				ChangedFiles: []*evalpb.SourceFile{
					{Path: "//unknown"},
				},
				TestVariants: []*evalpb.TestVariant{
					{FileName: "//b"},
				},
			}
			assertAffectedness(in, math.Inf(1), math.MaxInt32)
		})
	})
}
