// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package rules

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"go.chromium.org/luci/server/span"

	"infra/appengine/weetbix/internal/bugs"
	"infra/appengine/weetbix/internal/clustering"
	"infra/appengine/weetbix/internal/testutil"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

const testProject = "myproject"

func TestSpan(t *testing.T) {
	Convey(`With Spanner Test Database`, t, func() {
		ctx := testutil.SpannerTestContext(t)
		Convey(`Read`, func() {
			validateAndClearTimes := func(rules []*FailureAssociationRule) {
				for _, r := range rules {
					So(r.CreationTime, ShouldNotBeZeroValue)
					So(r.LastUpdated, ShouldNotBeZeroValue)
					So(r.LastUpdated, ShouldEqual, r.CreationTime)
					// CreationTime and LastUpdated are set by implementation.
					// Ignore their values when comparing to expected rules.
					r.LastUpdated = time.Time{}
					r.CreationTime = time.Time{}
				}
			}
			Convey(`Empty`, func() {
				setRules(ctx, nil)

				rules, err := ReadActive(span.Single(ctx), testProject)
				So(err, ShouldBeNil)
				validateAndClearTimes(rules)
				So(rules, ShouldResemble, []*FailureAssociationRule{})
			})
			Convey(`Multiple`, func() {
				rulesToCreate := []*FailureAssociationRule{
					newRule(0),
					newRule(1),
					newRule(2),
				}
				rulesToCreate[1].IsActive = false
				setRules(ctx, rulesToCreate)

				rules, err := ReadActive(span.Single(ctx), testProject)
				So(err, ShouldBeNil)
				validateAndClearTimes(rules)
				So(rules, ShouldResemble, []*FailureAssociationRule{
					newRule(0),
					newRule(2),
				})
			})
		})
		Convey(`Create`, func() {
			testCreate := func(bc *FailureAssociationRule) error {
				_, err := span.ReadWriteTransaction(ctx, func(ctx context.Context) error {
					return Create(ctx, bc)
				})
				return err
			}
			r := newRule(100)
			Convey(`Valid`, func() {
				Convey(`With Source Cluster`, func() {
					So(r.SourceCluster.Algorithm, ShouldNotBeEmpty)
					So(r.SourceCluster.ID, ShouldNotBeNil)
					err := testCreate(r)
					So(err, ShouldBeNil)
				})
				Convey(`Without Source Cluster`, func() {
					// E.g. in case of a manually created rule.
					r.SourceCluster = clustering.ClusterID{}
					err := testCreate(r)
					So(err, ShouldBeNil)
				})
				// Create followed by read already tested as part of Read tests.
			})
			Convey(`With invalid Project`, func() {
				Convey(`Missing`, func() {
					r.Project = ""
					err := testCreate(r)
					So(err, ShouldErrLike, "project must be valid")
				})
				Convey(`Invalid`, func() {
					r.Project = "!"
					err := testCreate(r)
					So(err, ShouldErrLike, "project must be valid")
				})
			})
			Convey(`With invalid Rule Definition`, func() {
				r.RuleDefinition = "invalid"
				err := testCreate(r)
				So(err, ShouldErrLike, "rule definition is not valid")
			})
			Convey(`With invalid Bug`, func() {
				r.Bug.System = ""
				err := testCreate(r)
				So(err, ShouldErrLike, "bug is not valid")
			})
			Convey(`With invalid Source Cluster`, func() {
				So(r.SourceCluster.ID, ShouldNotBeNil)
				r.SourceCluster.Algorithm = ""
				err := testCreate(r)
				So(err, ShouldErrLike, "source cluster ID is not valid")
			})
		})
	})
}

func newRule(uniqifier int) *FailureAssociationRule {
	ruleIDBytes := sha256.Sum256([]byte(fmt.Sprintf("rule-id%v", uniqifier)))
	return &FailureAssociationRule{
		Project:        testProject,
		RuleID:         hex.EncodeToString(ruleIDBytes[0:16]),
		RuleDefinition: "reason LIKE \"%exit code 5%\" AND test LIKE \"tast.arc.%\"",
		Bug:            bugs.BugID{System: "monorail", ID: fmt.Sprintf("project/%v", uniqifier)},
		IsActive:       true,
		SourceCluster: clustering.ClusterID{
			Algorithm: fmt.Sprintf("clusteralg%v", uniqifier),
			ID:        hex.EncodeToString([]byte(fmt.Sprintf("id%v", uniqifier))),
		},
	}
}

// setRules replaces the set of stored rules to match the given set.
func setRules(ctx context.Context, rs []*FailureAssociationRule) {
	testutil.MustApply(ctx,
		spanner.Delete("FailureAssociationRules", spanner.AllKeys()))
	// Insert some FailureAssociationRules.
	_, err := span.ReadWriteTransaction(ctx, func(ctx context.Context) error {
		for _, bc := range rs {
			if err := Create(ctx, bc); err != nil {
				return err
			}
		}
		return nil
	})
	So(err, ShouldBeNil)
}
