// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package rules

import (
	"context"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/luci/common/api/gerrit"

	cpb "infra/appengine/cr-audit-commits/app/proto"
)

func TestTBRRules(t *testing.T) {
	Convey("TBR Rules", t, func() {
		ctx := context.Background()
		rc := &RelevantCommit{
			CommitHash:       "7b12c0de1",
			Status:           AuditScheduled,
			CommitTime:       time.Date(2017, time.August, 25, 15, 0, 0, 0, time.UTC),
			CommitterAccount: "jdoe@sample.com",
			AuthorAccount:    "jdoe@sample.com",
			CommitMessage:    "Revert security fix, TBR=someone",
		}
		cfg := &RefConfig{
			BaseRepoURL: "https://a.googlesource.com/a.git",
			GerritURL:   "https://a-review.googlesource.com/",
			BranchName:  "master",
		}
		ap := &AuditParams{
			TriggeringAccount: "jdoe@sample.com",
			RepoCfg:           cfg,
		}
		reviewedcl := &gerrit.Change{
			ChangeID:        "tbrchangeid1",
			ChangeNumber:    4321,
			CurrentRevision: "7b12c0de1",
			Owner: gerrit.AccountInfo{
				AccountID: 1337,
			},

			Labels: map[string]gerrit.LabelInfo{
				"Code-Review": {
					All: []gerrit.VoteInfo{
						{
							AccountInfo: gerrit.AccountInfo{
								AccountID: 1337,
							},
							Value: 1,
						},
						{
							AccountInfo: gerrit.AccountInfo{
								AccountID: 4001,
							},
							Value: 1,
						},
					},
					Values: map[string]string{
						"-1": "No",
						" 0": "Whatever",
						"+1": "Yes",
					},
				},
			},
		}
		notreviewedcl := &gerrit.Change{
			ChangeID:        "tbrchangeid2",
			ChangeNumber:    4322,
			CurrentRevision: "7b12c0de2",
			Owner: gerrit.AccountInfo{
				AccountID: 1337,
			},
			Labels: map[string]gerrit.LabelInfo{
				"Code-Review": {
					All: []gerrit.VoteInfo{
						{
							AccountInfo: gerrit.AccountInfo{
								AccountID: 1337,
							},
							Value: 1,
						},
					},
					Values: map[string]string{
						"-1": "No",
						" 0": "Whatever",
						"+1": "Yes",
					},
				},
			},
		}
		botCommitCl := &gerrit.Change{
			ChangeID:        "botcommit123",
			ChangeNumber:    4322,
			CurrentRevision: "7b12c0de3",
			Owner: gerrit.AccountInfo{
				AccountID: 1337,
			},
			Labels: map[string]gerrit.LabelInfo{
				"Bot-Commit": {
					All: []gerrit.VoteInfo{
						{
							AccountInfo: gerrit.AccountInfo{
								AccountID: 1551,
							},
							Value: 1,
						},
					},
					Values: map[string]string{
						" 0": "Whatever",
						"+1": "Yes",
					},
				},
			},
		}
		q := map[string][]*gerrit.Change{
			"commit:7b12c0de1": {reviewedcl},
			"commit:7b12c0de2": {notreviewedcl},
			"commit:7b12c0de3": {botCommitCl},
		}
		testClients := &Clients{}
		testClients.gerrit = &mockGerritClient{q: q}

		expectedStatus := RuleInvalid
		var gerritCalls []string

		Convey("Pass", func() {
			expectedStatus = RulePassed
		})

		Convey("Non-Pass", func() {
			Convey("Fail", func() {
				rc.CommitHash = "7b12c0de2"
				expectedStatus = RuleFailed
			})

			Convey("Pending - no notification", func() {
				rc.CommitHash = "7b12c0de2"
				rc.CommitTime = time.Now().Add(-23 * time.Hour)
				expectedStatus = RulePending
			})

			Convey("Pending - send notification", func() {
				rc.CommitHash = "7b12c0de2"
				rc.CommitTime = time.Now().Add(-25 * time.Hour)
				rc.Result = append(rc.Result, RuleResult{
					RuleName:         "ChangeReviewed",
					RuleResultStatus: RulePending,
				})
				expectedStatus = RulePending
				gerritCalls = []string{"SetReview"}
			})
		})

		Convey("Skip", func() {
			expectedStatus = RuleSkipped
			rc.AuthorAccount = "robot2@example.com"
		})

		c := ChangeReviewed{
			&cpb.ChangeReviewed{
				Robots: []string{"robot1@example.com", "robot2@example.com"},
			},
		}
		rr, _ := c.Run(ctx, ap, rc, testClients)
		So(rr.RuleResultStatus, ShouldEqual, expectedStatus)
		So(testClients.gerrit.(*mockGerritClient).calls, ShouldResemble, gerritCalls)
	})
}
