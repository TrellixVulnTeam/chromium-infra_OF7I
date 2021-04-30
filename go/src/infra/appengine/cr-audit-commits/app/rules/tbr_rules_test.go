// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package rules

import (
	"context"
	"fmt"
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

		reviewedCL := &gerrit.Change{
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

		selfReviewedCL := &gerrit.Change{
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

		botCommitCL := &gerrit.Change{
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

		botCommitAndSelfCL := &gerrit.Change{
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

		selfReviewedPlus2CL := &gerrit.Change{
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

		reviewedPlus2CL := &gerrit.Change{
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

		botOwnedCL := &gerrit.Change{
			ChangeID:        "tbrchangeid5",
			ChangeNumber:    4327,
			CurrentRevision: "7b12c0de7",
			Owner: gerrit.AccountInfo{
				AccountID: 567,
				Email:     "robot1@example.com",
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
			},
		}

		allCLs := []*gerrit.Change{
			reviewedCL,
			selfReviewedCL,
			botCommitCL,
			botCommitAndSelfCL,
			selfReviewedPlus2CL,
			reviewedPlus2CL,
			botOwnedCL,
		}
		q := map[string][]*gerrit.Change{}
		for _, cl := range allCLs {
			key := fmt.Sprintf("commit:%s", cl.CurrentRevision)
			q[key] = []*gerrit.Change{cl}
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
			rc.CommitHash = selfReviewedCL.CurrentRevision
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
			rc.CommitHash = botCommitCL.CurrentRevision
			expectedStatus = RulePassed
		})

		Convey("Bot-Commit and self CR", func() {
			rc.CommitHash = botCommitAndSelfCL.CurrentRevision
			expectedStatus = RulePassed
		})

		Convey("Self-reviewed +2 CL fail", func() {
			rc.CommitHash = selfReviewedCL.CurrentRevision
			expectedStatus = RuleFailed
			msg = msgFail
		})

		Convey("Reviewed +2 CL pass", func() {
			rc.CommitHash = reviewedPlus2CL.CurrentRevision
			expectedStatus = RulePassed
		})

		Convey("Bot-owned CL skip", func() {
			rc.CommitHash = botOwnedCL.CurrentRevision
			expectedStatus = RuleSkipped
		})

		rr, _ := c.Run(ctx, ap, rc, testClients)
		So(rr.RuleResultStatus.ToString(), ShouldEqual, expectedStatus.ToString())
		So(rr.Message, ShouldEqual, msg)
		So(testClients.gerrit.(*mockGerritClient).calls, ShouldResemble, []string(nil))
	})
}
