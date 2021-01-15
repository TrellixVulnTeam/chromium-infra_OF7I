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
				"Bot-Commit": {
					Values: map[string]string{
						" 0": "Whatever",
						"+1": "Yes",
					},
				},
			},
		}

		selfReviewedCl := &gerrit.Change{
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
				"Bot-Commit": {
					Values: map[string]string{
						" 0": "Whatever",
						"+1": "Yes",
					},
				},
			},
		}

		botCommitCl := &gerrit.Change{
			ChangeID:        "botcommit123",
			ChangeNumber:    4323,
			CurrentRevision: "7b12c0de3",
			Owner: gerrit.AccountInfo{
				AccountID: 1337,
			},
			Labels: map[string]gerrit.LabelInfo{
				"Code-Review": {
					Values: map[string]string{
						"-1": "No",
						" 0": "Whatever",
						"+1": "Yes",
					},
				},
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

		botCommitAndSelfCl := &gerrit.Change{
			ChangeID:        "botcommit456",
			ChangeNumber:    4324,
			CurrentRevision: "7b12c0de4",
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

		selfReviewedOsCl := &gerrit.Change{
			ChangeID:        "tbrchangeid3",
			ChangeNumber:    4325,
			CurrentRevision: "7b12c0de5",
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
							Value: 2,
						},
					},
					Values: map[string]string{
						"-1": "No",
						" 0": "Whatever",
						"+1": "Yes",
						"+2": "Super Yes",
					},
				},
				"Bot-Commit": {
					Values: map[string]string{
						" 0": "Whatever",
						"+1": "Yes",
					},
				},
			},
		}

		reviewedOsCl := &gerrit.Change{
			ChangeID:        "tbrchangeid4",
			ChangeNumber:    4326,
			CurrentRevision: "7b12c0de6",
			Owner: gerrit.AccountInfo{
				AccountID: 1337,
			},
			Labels: map[string]gerrit.LabelInfo{
				"Code-Review": {
					All: []gerrit.VoteInfo{
						{
							AccountInfo: gerrit.AccountInfo{
								AccountID: 4001,
							},
							Value: 2,
						},
					},
					Values: map[string]string{
						"-1": "No",
						" 0": "Whatever",
						"+1": "Yes",
						"+2": "Super Yes",
					},
				},
				"Bot-Commit": {
					Values: map[string]string{
						" 0": "Whatever",
						"+1": "Yes",
					},
				},
			},
		}

		q := map[string][]*gerrit.Change{
			"commit:7b12c0de1": {reviewedcl},
			"commit:7b12c0de2": {selfReviewedCl},
			"commit:7b12c0de3": {botCommitCl},
			"commit:7b12c0de4": {botCommitAndSelfCl},
			"commit:7b12c0de5": {selfReviewedOsCl},
			"commit:7b12c0de6": {reviewedOsCl},
		}

		testClients := &Clients{}
		testClients.gerrit = &mockGerritClient{q: q}

		const msgFail = "Please review"
		expectedStatus := RuleInvalid
		msg := ""
		c := ChangeReviewed{
			&cpb.ChangeReviewed{
				Robots:  []string{"robot1@example.com", "robot2@example.com"},
				Message: msgFail,
			},
		}

		Convey("Pass", func() {
			expectedStatus = RulePassed
		})

		Convey("Non-Pass", func() {
			rc.CommitHash = "7b12c0de2"
			expectedStatus = RuleFailed
			msg = msgFail
			Convey("Fail", func() {
			})

			Convey("Pending - no notification", func() {
				rc.CommitTime = time.Now().Add(-23 * time.Hour)
			})

			Convey("Pending - send notification", func() {
				rc.CommitTime = time.Now().Add(-25 * time.Hour)
				rc.Result = append(rc.Result, RuleResult{
					RuleName:         "ChangeReviewed",
					RuleResultStatus: RulePending,
				})
			})

			Convey("Default message", func() {
				c.Message = ""
				msg = chromeTBRMessage
			})
		})

		Convey("Skip", func() {
			expectedStatus = RuleSkipped
			rc.AuthorAccount = "robot2@example.com"
		})

		Convey("Bot-Commit", func() {
			rc.CommitHash = "7b12c0de3"
			expectedStatus = RulePassed
		})

		Convey("Bot-Commit and self CR", func() {
			rc.CommitHash = "7b12c0de4"
			expectedStatus = RulePassed
		})

		Convey("Self-reviewed OS CL fail", func() {
			rc.CommitHash = "7b12c0de5"
			expectedStatus = RuleFailed
			msg = msgFail
		})

		Convey("Reviewed OS CL pass", func() {
			rc.CommitHash = "7b12c0de6"
			expectedStatus = RulePassed
		})

		rr, _ := c.Run(ctx, ap, rc, testClients)
		So(rr.RuleResultStatus, ShouldEqual, expectedStatus)
		So(rr.Message, ShouldEqual, msg)
		So(testClients.gerrit.(*mockGerritClient).calls, ShouldResemble, []string(nil))
	})
}
