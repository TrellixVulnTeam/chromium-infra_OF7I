// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package eval

import (
	"bytes"
	"container/heap"
	"fmt"
	"strings"
	"testing"

	"infra/rts"
	evalpb "infra/rts/presubmit/eval/proto"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

func TestBucketSlice(t *testing.T) {
	t.Parallel()
	Convey(`bucketSlice`, t, func() {
		Convey(`inc`, func() {
			thresholds := make([]*evalpb.Threshold, 10)
			for i := 0; i < len(thresholds); i++ {
				thresholds[i] = &evalpb.Threshold{
					MaxDistance: float32(i),
				}
			}
			b := make(bucketSlice, len(thresholds)+1)

			Convey(`2`, func() {
				b.inc(thresholds, rts.Affectedness{Distance: 2}, 1)
				So(b[2], ShouldEqual, 1)
			})
			Convey(`3`, func() {
				b.inc(thresholds, rts.Affectedness{Distance: 3}, 1)
				So(b[3], ShouldEqual, 1)
			})
			Convey(`10`, func() {
				b.inc(thresholds, rts.Affectedness{Distance: 10}, 1)
				So(b[10], ShouldEqual, 1)
			})
			Convey(`0`, func() {
				// This data point was not lost by any threshold.
				b.inc(thresholds, rts.Affectedness{Distance: 0}, 1)
				So(b[0], ShouldEqual, 1)
			})
			Convey(`11`, func() {
				// This data point was lost by all thresholds.
				b.inc(thresholds, rts.Affectedness{Distance: 11}, 1)
				So(b[10], ShouldEqual, 1)
			})
		})
		Convey(`makeCumulative`, func() {
			b := make(bucketSlice, 10)

			assert10 := func(expected string) {
				expected = strings.TrimPrefix(expected, "\n")
				expected = strings.Replace(expected, "\t", "", -1)

				var buf bytes.Buffer
				for _, v := range b {
					fmt.Fprintf(&buf, "%d", v)
				}
				So(buf.String(), ShouldEqual, expected)
			}

			Convey(`b[0] = 1`, func() {
				b[0] = 1
				b.makeCumulative()
				assert10(`1000000000`)
			})

			Convey(`b[5] = 1`, func() {
				b[5] = 1
				b.makeCumulative()
				assert10(`1111110000`)
			})

			Convey(`b[2] = 1, b[4] = 2`, func() {
				b[2] = 1
				b[4] = 2
				b.makeCumulative()
				assert10(`3332200000`)
			})
		})
	})
}

func TestMostAffected(t *testing.T) {
	t.Parallel()
	Convey(`Test[]rts.Affectedness`, t, func() {
		Convey(`mostAffected`, func() {
			Convey(`Works`, func() {
				most, err := mostAffected([]rts.Affectedness{
					{Distance: 1},
					{Distance: 0},
				})
				So(err, ShouldBeNil)
				So(most, ShouldResemble, rts.Affectedness{Distance: 0})
			})

			Convey(`Empty`, func() {
				_, err := mostAffected(nil)
				So(err, ShouldErrLike, "empty")
			})

			Convey(`Single`, func() {
				most, err := mostAffected([]rts.Affectedness{{Distance: 0}})
				So(err, ShouldBeNil)
				So(most, ShouldResemble, rts.Affectedness{Distance: 0})
			})
		})
	})
}

func TestQuantiles(t *testing.T) {
	t.Parallel()
	Convey(`Quantiles`, t, func() {
		Convey(`median of 1, 2, 3, 4`, func() {
			afs := []rts.Affectedness{
				{Distance: 1},
				{Distance: 2},
				{Distance: 3},
				{Distance: 4},
			}
			So(distanceQuantiles(afs, 2), ShouldResemble, []float32{2, 4})
		})
		Convey(`4-quantiles of 1, 2, 3, 4`, func() {
			afs := []rts.Affectedness{
				{Distance: 1},
				{Distance: 2},
				{Distance: 3},
				{Distance: 4},
			}
			So(distanceQuantiles(afs, 4), ShouldResemble, []float32{1, 2, 3, 4})
		})
		Convey(`10-quantiles of 1, 2, 3, 4`, func() {
			afs := []rts.Affectedness{
				{Distance: 1},
				{Distance: 2},
				{Distance: 3},
				{Distance: 4},
			}
			So(distanceQuantiles(afs, 10), ShouldResemble, []float32{1, 1, 2, 2, 2, 3, 3, 4, 4, 4})
		})
	})
}

func TestFurthestRejections(t *testing.T) {
	Convey("FurthestRejections", t, func() {
		furthest := make(furthestRejections, 3)
		furthest.Consider(affectedRejection{MostAffected: rts.Affectedness{Distance: 1}})
		furthest.Consider(affectedRejection{MostAffected: rts.Affectedness{Distance: 2}})
		furthest.Consider(affectedRejection{MostAffected: rts.Affectedness{Distance: 3}})
		furthest.Consider(affectedRejection{MostAffected: rts.Affectedness{Distance: 4}})
		furthest.Consider(affectedRejection{MostAffected: rts.Affectedness{Distance: 5}})

		So(len(furthest), ShouldEqual, 3)
		So(heap.Pop(&furthest), ShouldResemble, affectedRejection{MostAffected: rts.Affectedness{Distance: 3}})
		So(heap.Pop(&furthest), ShouldResemble, affectedRejection{MostAffected: rts.Affectedness{Distance: 4}})
		So(heap.Pop(&furthest), ShouldResemble, affectedRejection{MostAffected: rts.Affectedness{Distance: 5}})
	})
}
