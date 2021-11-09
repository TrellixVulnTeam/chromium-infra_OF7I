// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cache

import (
	"infra/appengine/weetbix/internal/clustering/rules"
	"infra/appengine/weetbix/internal/testutil"
	"testing"
	"time"

	"go.chromium.org/luci/common/clock/testclock"
	"go.chromium.org/luci/server/caching"

	. "github.com/smartystreets/goconvey/convey"
)

var cache = caching.RegisterLRUCache(50)

func TestRulesCache(t *testing.T) {
	Convey(`With Spanner Test Database`, t, func() {
		ctx := testutil.SpannerTestContext(t)
		ctx, tc := testclock.UseTime(ctx, testclock.TestRecentTimeUTC)
		ctx = caching.WithEmptyProcessCache(ctx)

		rc := NewRulesCache(cache)
		rules.SetRulesForTesting(ctx, nil)

		test := func(expectedRules []*rules.FailureAssociationRule, expectedVersion time.Time) {
			// Tests the content of the cache is as expected.
			ruleset, err := rc.Ruleset(ctx, "myproject")
			So(err, ShouldBeNil)
			So(ruleset.RulesVersion, ShouldEqual, expectedVersion)

			actualUpdated := make(map[string]time.Time)
			expectedUpdated := make(map[string]time.Time)
			actualRule := make(map[string]string)
			expectedRule := make(map[string]string)

			for _, a := range ruleset.Rules {
				actualUpdated[a.RuleID] = a.LastUpdated
				So(a.Expr, ShouldNotBeNil)
				// Technically this only gets us back the original rule if it
				// was in normal formatting. But during testing, we use rules
				// with normalised formatting.
				actualRule[a.RuleID] = a.Expr.String()
			}
			for _, e := range expectedRules {
				if e.IsActive {
					expectedUpdated[e.RuleID] = e.LastUpdated
					expectedRule[e.RuleID] = e.RuleDefinition
				}
			}
			So(actualUpdated, ShouldResemble, expectedUpdated)
			So(actualRule, ShouldResemble, expectedRule)
		}

		Convey(`Initially Empty`, func() {
			err := rules.SetRulesForTesting(ctx, nil)
			So(err, ShouldBeNil)
			test(nil, rules.StartingEpoch)

			Convey(`Then Empty`, func() {
				// Test cache.
				test(nil, rules.StartingEpoch)

				tc.Add(refreshInterval)

				test(nil, rules.StartingEpoch)
				test(nil, rules.StartingEpoch)
			})
			Convey(`Then Non-Empty`, func() {
				// Spanner commit timestamps are in microsecond
				// (not nanosecond) granularity, and some Spanner timestamp
				// operators truncates to microseconds. For this
				// reason, we use microsecond resolution timestamps
				// when testing.
				reference := time.Date(2020, 1, 2, 3, 4, 5, 6000, time.UTC)

				rs := []*rules.FailureAssociationRule{
					rules.NewRule(100).WithLastUpdated(reference.Add(-1 * time.Hour)).Build(),
					rules.NewRule(101).WithActive(false).WithLastUpdated(reference).Build(),
				}
				err := rules.SetRulesForTesting(ctx, rs)
				So(err, ShouldBeNil)

				// Test cache is working and still returning the old value.
				tc.Add(refreshInterval / 2)
				test(nil, rules.StartingEpoch)

				tc.Add(refreshInterval)

				test(rs, reference)
				test(rs, reference)
			})
		})
		Convey(`Initially Non-Empty`, func() {
			reference := time.Date(2020, 1, 2, 3, 4, 5, 6000, time.UTC)

			rs := []*rules.FailureAssociationRule{
				rules.NewRule(100).WithLastUpdated(reference.Add(-1 * time.Hour)).Build(),
				rules.NewRule(101).WithActive(false).WithLastUpdated(reference).Build(),
			}
			err := rules.SetRulesForTesting(ctx, rs)
			So(err, ShouldBeNil)

			test(rs, reference)

			Convey(`Then Empty`, func() {
				// Mark all rules inactive.
				newRules := []*rules.FailureAssociationRule{
					rules.NewRule(100).WithActive(false).WithLastUpdated(reference.Add(time.Hour)).Build(),
					rules.NewRule(101).WithActive(false).WithLastUpdated(reference).Build(),
				}
				err := rules.SetRulesForTesting(ctx, newRules)
				So(err, ShouldBeNil)

				// Test cache is working and still returning the old value.
				tc.Add(refreshInterval / 2)
				test(rs, reference)

				tc.Add(refreshInterval)

				test(newRules, reference.Add(time.Hour))
				test(newRules, reference.Add(time.Hour))
			})
			Convey(`Then Non-Empty`, func() {
				// Make an existing rule inactive, make an existing inactive
				// rule active, and add an inactive and active new rule.
				newRules := []*rules.FailureAssociationRule{
					rules.NewRule(100).WithActive(false).WithLastUpdated(reference.Add(time.Hour)).Build(),
					rules.NewRule(101).WithLastUpdated(reference.Add(time.Hour)).Build(),
					rules.NewRule(102).WithLastUpdated(reference.Add(time.Hour)).Build(),
					rules.NewRule(103).WithActive(false).WithLastUpdated(reference.Add(2 * time.Hour)).Build(),
				}
				err := rules.SetRulesForTesting(ctx, newRules)
				So(err, ShouldBeNil)

				// Test cache is working and still returning the old value.
				tc.Add(refreshInterval / 2)
				test(rs, reference)

				tc.Add(refreshInterval)

				test(newRules, reference.Add(2*time.Hour))
				test(newRules, reference.Add(2*time.Hour))
			})
		})
	})
}
