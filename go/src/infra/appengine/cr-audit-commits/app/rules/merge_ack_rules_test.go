// Copyright 2018 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package rules

import (
	"testing"

	"context"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/gae/impl/memory"
	"go.chromium.org/gae/service/datastore"

	"infra/monorail"
)

func TestMergeAckRules(t *testing.T) {

	Convey("Merge Acknowledgement rules work", t, func() {
		ctx := memory.Use(context.Background())
		rs := &RefState{
			RepoURL: "https://a.googlesource.com/a.git/+/master",
		}
		datastore.Put(ctx, rs)
		rc := &RelevantCommit{
			RefStateKey:   datastore.KeyForObj(ctx, rs),
			CommitHash:    "b07c0de",
			Status:        AuditScheduled,
			CommitMessage: "Acknowledging merges into a release branch",
		}
		cfg := &RefConfig{
			BaseRepoURL: "https://a.googlesource.com/a.git",
			GerritURL:   "https://a-review.googlesource.com/",
			BranchName:  "3325",
			Metadata:    "MilestoneNumber:65",
		}
		ap := &AuditParams{
			TriggeringAccount: "releasebot@sample.com",
			RepoCfg:           cfg,
		}
		testClients := &Clients{}
		testClients.Monorail = MockMonorailClient{
			Gi: &monorail.Issue{},
			Ii: &monorail.InsertIssueResponse{
				Issue: &monorail.Issue{},
			},
		}
		Convey("Change to commit has a valid bug", func() {
			testClients.Monorail = MockMonorailClient{
				Gi: &monorail.Issue{
					Id: 123456,
				},
			}
			rc.CommitMessage = "This change has a valid bug ID \nBug:123456"
			// Run rule
			rr, _ := AcknowledgeMerge{}.Run(ctx, ap, rc, testClients)
			So(rr.RuleResultStatus, ShouldEqual, NotificationRequired)
		})
		Convey("Change to commit has no bug", func() {
			testClients.Monorail = MockMonorailClient{
				Gi: &monorail.Issue{
					Id: 123456,
				},
			}
			rc.CommitMessage = "This change has no bug attached"
			// Run rule
			rr, _ := AcknowledgeMerge{}.Run(ctx, ap, rc, testClients)
			So(rr.RuleResultStatus, ShouldEqual, RuleSkipped)
		})
	})
}
