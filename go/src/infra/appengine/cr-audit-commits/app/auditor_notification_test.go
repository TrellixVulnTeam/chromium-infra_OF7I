// Copyright 2017 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"fmt"
	"testing"

	"context"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/gae/impl/memory"
	ds "go.chromium.org/gae/service/datastore"
	"go.chromium.org/gae/service/mail"
	"go.chromium.org/gae/service/user"

	"infra/appengine/cr-audit-commits/app/rules"
	"infra/monorail"
)

// sendEmailForFinditViolation is not actually used by any AccountRules its purpose
// is to illustrate how one would use SendEmailForViolation to notify about
// violations via email.
func sendEmailForFinditViolation(ctx context.Context, cfg *rules.RefConfig, rc *rules.RelevantCommit, cs *rules.Clients, state string) (string, error) {
	recipients := []string{"eng-team@dummy.com"}
	subject := "A policy violation was detected on commit %s"
	return rules.SendEmailForViolation(ctx, cfg, rc, cs, state, recipients, subject)
}

func TestNotifier(t *testing.T) {

	Convey("ViolationNotifier handler test", t, func() {
		ctx := memory.UseWithAppID(context.Background(), "cr-audit-commits-test")

		user.GetTestable(ctx).Login("notifier@cr-audit-commits-test.appspotmail.com", "", false)

		testClients := &rules.Clients{}
		Convey("Existing Repo", func() {
			cfg := &rules.RefConfig{
				BaseRepoURL:     "https://old.googlesource.com/old.git",
				GerritURL:       "https://old-review.googlesource.com",
				BranchName:      "master",
				StartingCommit:  "000000",
				MonorailAPIURL:  "https://monorail-fake.appspot.com/_ah/api/monorail/v1",
				MonorailProject: "fakeproject",
				NotifierEmail:   "notifier@cr-audit-commits-test.appspotmail.com",
				Rules: map[string]rules.AccountRules{"rules": {
					Account: "author@test.com",
					Rules: []rules.Rule{
						rules.DummyRule{
							Name: "Dummy rule",
							Result: &rules.RuleResult{
								RuleName:         "Dummy rule",
								RuleResultStatus: rules.RulePassed,
								Message:          "",
								MetaData:         "label:Random-Label",
							},
						},
					},
					NotificationFunction: rules.FileBugForFinditViolation,
				}},
			}
			rules.RuleMap["old-repo"] = cfg
			refState := &rules.RefState{
				RepoURL:            "https://old.googlesource.com/old.git/+/master",
				LastKnownCommit:    "123456",
				LastRelevantCommit: "999999",
			}
			ds.Put(ctx, refState)

			Convey("No audits", func() {
				testClients.Monorail = rules.MockMonorailClient{
					E: fmt.Errorf("Monorail was called even though there were no failed audits"),
				}
				err := notifyAboutViolations(ctx, cfg, refState, testClients)
				So(err, ShouldBeNil)
			})
			Convey("No failed audits", func() {
				rsk := ds.KeyForObj(ctx, refState)
				testClients.Monorail = rules.MockMonorailClient{
					E: fmt.Errorf("Monorail was called even though there were no failed audits"),
				}
				rc := &rules.RelevantCommit{
					RefStateKey: rsk,
					CommitHash:  "600dc0de",
					Status:      rules.AuditCompleted,
					Result: []rules.RuleResult{{
						RuleName:         "DummyRule",
						RuleResultStatus: rules.RulePassed,
						Message:          "",
						MetaData:         "",
					}},
					CommitterAccount: "committer@test.com",
					AuthorAccount:    "author@test.com",
					CommitMessage:    "This commit passed all audits.",
				}
				err := ds.Put(ctx, rc)
				So(err, ShouldBeNil)

				err = notifyAboutViolations(ctx, cfg, refState, testClients)
				So(err, ShouldBeNil)
				rc = &rules.RelevantCommit{
					RefStateKey: rsk,
					CommitHash:  "600dc0de",
				}
				err = ds.Get(ctx, rc)
				So(err, ShouldBeNil)
				So(rc.GetNotificationState("rules"), ShouldEqual, "")
				So(rc.NotifiedAll, ShouldBeFalse)
			})
			Convey("Failed audits - bug only", func() {
				rsk := ds.KeyForObj(ctx, refState)
				testClients.Monorail = rules.MockMonorailClient{
					Il: &monorail.IssuesListResponse{},
					Ii: &monorail.InsertIssueResponse{
						Issue: &monorail.Issue{
							Id: 12345,
						},
					},
				}
				rc := &rules.RelevantCommit{
					RefStateKey: rsk,
					CommitHash:  "badc0de",
					Status:      rules.AuditCompletedWithActionRequired,
					Result: []rules.RuleResult{{
						RuleName:         "DummyRule",
						RuleResultStatus: rules.RuleFailed,
						Message:          "This commit is bad",
						MetaData:         "",
					}},
					CommitterAccount: "committer@test.com",
					AuthorAccount:    "author@test.com",
					CommitMessage:    "This commit failed all audits.",
				}
				err := ds.Put(ctx, rc)
				So(err, ShouldBeNil)

				err = notifyAboutViolations(ctx, cfg, refState, testClients)
				So(err, ShouldBeNil)
				rc = &rules.RelevantCommit{
					RefStateKey: rsk,
					CommitHash:  "badc0de",
				}
				err = ds.Get(ctx, rc)
				So(err, ShouldBeNil)
				So(rc.GetNotificationState("rules"), ShouldEqual, "BUG=12345")
				So(rc.NotifiedAll, ShouldBeTrue)
				m := mail.GetTestable(ctx)
				So(m.SentMessages(), ShouldBeEmpty)

			})
			Convey("Exceeded retries", func() {
				rsk := ds.KeyForObj(ctx, refState)
				testClients.Monorail = rules.MockMonorailClient{
					Ii: &monorail.InsertIssueResponse{
						Issue: &monorail.Issue{
							Id: 12345,
						},
					},
				}
				rc := &rules.RelevantCommit{
					RefStateKey:      rsk,
					CommitHash:       "b00b00",
					Status:           rules.AuditFailed,
					Result:           []rules.RuleResult{},
					CommitterAccount: "committer@test.com",
					AuthorAccount:    "author@test.com",
					CommitMessage:    "This commit panicked and panicked",
					Retries:          rules.MaxRetriesPerCommit + 1,
				}
				err := ds.Put(ctx, rc)
				So(err, ShouldBeNil)

				err = notifyAboutViolations(ctx, cfg, refState, testClients)
				So(err, ShouldBeNil)
				rc = &rules.RelevantCommit{
					RefStateKey: rsk,
					CommitHash:  "b00b00",
				}
				err = ds.Get(ctx, rc)
				So(err, ShouldBeNil)
				So(rc.GetNotificationState("AuditFailure"), ShouldEqual, "BUG=12345")
				So(rc.NotifiedAll, ShouldBeTrue)
			})
		})
		Convey("Notification required audits - comment only", func() {
			testClients.Monorail = rules.MockMonorailClient{
				Gi: &monorail.Issue{
					Id: 8675389,
				},
			}
			cfg := &rules.RefConfig{
				BaseRepoURL:     "https://old.googlesource.com/old-ack.git",
				GerritURL:       "https://old-review.googlesource.com",
				BranchName:      "master",
				StartingCommit:  "000000",
				MonorailAPIURL:  "https://monorail-fake.appspot.com/_ah/api/monorail/v1",
				MonorailProject: "fakeproject",
				NotifierEmail:   "notifier@cr-audit-commits-test.appspotmail.com",
				Rules: map[string]rules.AccountRules{"rulesAck": {
					Account: "author@test.com",
					Rules: []rules.Rule{
						rules.DummyRule{
							Name: "Dummy rule",
							Result: &rules.RuleResult{
								RuleName:         "Dummy rule",
								RuleResultStatus: rules.NotificationRequired,
								Message:          "",
								MetaData:         "BugNumbers:8675389",
							},
						},
					},
					NotificationFunction: rules.CommentOnBugToAcknowledgeMerge,
				}},
				Metadata: "MilestoneNumber:70",
			}
			rules.RuleMap["old-repo-ack"] = cfg
			refState := &rules.RefState{
				RepoURL:            "https://old.googlesource.com/old-ack.git/+/master",
				LastKnownCommit:    "123456",
				LastRelevantCommit: "999999",
			}
			ds.Put(ctx, refState)
			rsk := ds.KeyForObj(ctx, refState)
			rc := &rules.RelevantCommit{
				RefStateKey: rsk,
				CommitHash:  "badc0de",
				Status:      rules.AuditCompletedWithActionRequired,
				Result: []rules.RuleResult{{
					RuleName:         "DummyRule",
					RuleResultStatus: rules.NotificationRequired,
					Message:          "This commit requires a notification",
					MetaData:         "BugNumbers:8675389",
				}},
				CommitterAccount: "committer@test.com",
				AuthorAccount:    "author@test.com",
				CommitMessage:    "This commit requires a notification.",
			}
			err := ds.Put(ctx, rc)
			So(err, ShouldBeNil)

			err = notifyAboutViolations(ctx, cfg, refState, testClients)
			So(err, ShouldBeNil)

			rc = &rules.RelevantCommit{
				RefStateKey: rsk,
				CommitHash:  "badc0de",
			}
			err = ds.Get(ctx, rc)
			So(rc.GetNotificationState("rulesAck"), ShouldEqual, "Comment posted on BUG(S)=8675389")
			So(rc.NotifiedAll, ShouldBeTrue)
			m := mail.GetTestable(ctx)
			So(m.SentMessages(), ShouldBeEmpty)
		})
		Convey("Failed audits - email only", func() {
			cfg := &rules.RefConfig{
				BaseRepoURL:     "https://old.googlesource.com/old-email.git",
				GerritURL:       "https://old-review.googlesource.com",
				BranchName:      "master",
				StartingCommit:  "000000",
				MonorailAPIURL:  "https://monorail-fake.appspot.com/_ah/api/monorail/v1",
				MonorailProject: "fakeproject",
				NotifierEmail:   "notifier@cr-audit-commits-test.appspotmail.com",
				Rules: map[string]rules.AccountRules{"rulesEmail": {
					Account: "author@test.com",
					Rules: []rules.Rule{
						rules.DummyRule{
							Name: "Dummy rule",
							Result: &rules.RuleResult{
								RuleName:         "Dummy rule",
								RuleResultStatus: rules.RulePassed,
								Message:          "",
								MetaData:         "",
							},
						},
					},
					NotificationFunction: sendEmailForFinditViolation,
				}},
			}
			rules.RuleMap["old-repo-email"] = cfg
			refState := &rules.RefState{
				RepoURL:            "https://old.googlesource.com/old-email.git/+/master",
				LastKnownCommit:    "123456",
				LastRelevantCommit: "999999",
			}
			ds.Put(ctx, refState)
			rsk := ds.KeyForObj(ctx, refState)
			rc := &rules.RelevantCommit{
				RefStateKey: rsk,
				CommitHash:  "badc0de",
				Status:      rules.AuditCompletedWithActionRequired,
				Result: []rules.RuleResult{{
					RuleName:         "DummyRule",
					RuleResultStatus: rules.RuleFailed,
					Message:          "This commit is bad",
					MetaData:         "",
				}},
				CommitterAccount: "committer@test.com",
				AuthorAccount:    "author@test.com",
				CommitMessage:    "This commit failed all audits.",
			}
			err := ds.Put(ctx, rc)
			So(err, ShouldBeNil)

			err = notifyAboutViolations(ctx, cfg, refState, testClients)
			So(err, ShouldBeNil)
			rc = &rules.RelevantCommit{
				RefStateKey: rsk,
				CommitHash:  "badc0de",
			}
			err = ds.Get(ctx, rc)
			So(err, ShouldBeNil)
			m := mail.GetTestable(ctx)
			So(rc.NotifiedAll, ShouldBeTrue)
			So(m.SentMessages()[0], ShouldResemble,
				&mail.TestMessage{
					Message: mail.Message{
						Sender:  "notifier@cr-audit-commits-test.appspotmail.com",
						To:      []string{"eng-team@dummy.com"},
						Subject: "A policy violation was detected on commit badc0de",
						Body:    "Here are the messages from the rules that were broken by this commit:\n\nThis commit is bad",
					}})

		})
	})
}
