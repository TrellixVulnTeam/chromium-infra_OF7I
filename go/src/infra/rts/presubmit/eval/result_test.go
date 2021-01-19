// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package eval

import (
	"bytes"
	"testing"
	"time"

	"infra/rts"

	. "github.com/smartystreets/goconvey/convey"
)

func TestResultPrint(t *testing.T) {
	t.Parallel()

	Convey(`Print`, t, func() {
		r := Result{
			TotalRejections:   100,
			TotalTestFailures: 100,
			TotalDuration:     time.Hour,
			Thresholds: []*Threshold{
				{
					Savings: 1,
				},
				{
					Value:                rts.Affectedness{Distance: 10},
					DistanceChangeRecall: 0.99,
					RankChangeRecall:     0.99,
					ChangeRecall:         0.99,
					TestRecall:           0.99,
					Savings:              0.25,
				},
				{
					Value:                rts.Affectedness{Distance: 40},
					DistanceChangeRecall: 1,
					RankChangeRecall:     1,
					ChangeRecall:         1,
					TestRecall:           1,
					Savings:              0.5,
				},
			},
		}

		buf := &bytes.Buffer{}
		r.Print(buf, 0)
		So(buf.String(), ShouldEqual, `
ChangeRecall | Savings | TestRecall | Distance, ChangeRecall | Rank, ChangeRecall
---------------------------------------------------------------------------------
  0.00%      | 100.00% |   0.00%    |  0.000              0% | 0               0%
 99.00%      |  25.00% |  99.00%    | 10.000             99% | 0              99%
100.00%      |  50.00% | 100.00%    | 40.000            100% | 0             100%

based on 100 rejections, 100 test failures, 1h0m0s testing time
`[1:])
	})
}
