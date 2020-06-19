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

	"infra/appengine/cr-audit-commits/app/rules"
	"infra/monorail"
)

func TestNotifier(t *testing.T) {

	Convey("ViolationNotifier handler test", t, func() {
		ctx := memory.UseWithAppID(context.Background(), "cr-audit-commits-test")

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
					Notification: rules.CommentOrFileMonorailIssue{
						Components: []string{"Tools>Test>Findit>Autorevert"},
						Labels:     []string{"CommitLog-Audit-Violation"},
					},
				}},
			}
			// TODO: Do not mutate global state.
			rules.GetRuleMap()["old-repo"] = cfg
			repoState := &rules.RepoState{
				RepoURL:            "https://old.googlesource.com/old.git/+/master",
				LastKnownCommit:    "123456",
				LastRelevantCommit: "999999",
			}
			ds.Put(ctx, repoState)

			Convey("No audits", func() {
				testClients.Monorail = rules.MockMonorailClient{
					E: fmt.Errorf("Monorail was called even though there were no failed audits"),
				}
				err := notifyAboutViolations(ctx, cfg, repoState, testClients)
				So(err, ShouldBeNil)
			})
			Convey("No failed audits", func() {
				rsk := ds.KeyForObj(ctx, repoState)
				testClients.Monorail = rules.MockMonorailClient{
					E: fmt.Errorf("Monorail was called even though there were no failed audits"),
				}
				rc := &rules.RelevantCommit{
					RepoStateKey: rsk,
					CommitHash:   "600dc0de",
					Status:       rules.AuditCompleted,
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

				err = notifyAboutViolations(ctx, cfg, repoState, testClients)
				So(err, ShouldBeNil)
				rc = &rules.RelevantCommit{
					RepoStateKey: rsk,
					CommitHash:   "600dc0de",
				}
				err = ds.Get(ctx, rc)
				So(err, ShouldBeNil)
				So(rc.GetNotificationState("rules"), ShouldEqual, "")
				So(rc.NotifiedAll, ShouldBeFalse)
			})
			Convey("Failed audits - bug only", func() {
				rsk := ds.KeyForObj(ctx, repoState)
				testClients.Monorail = rules.MockMonorailClient{
					Il: &monorail.IssuesListResponse{},
					Ii: &monorail.InsertIssueResponse{
						Issue: &monorail.Issue{
							Id: 12345,
						},
					},
				}
				rc := &rules.RelevantCommit{
					RepoStateKey: rsk,
					CommitHash:   "badc0de",
					Status:       rules.AuditCompletedWithActionRequired,
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

				err = notifyAboutViolations(ctx, cfg, repoState, testClients)
				So(err, ShouldBeNil)
				rc = &rules.RelevantCommit{
					RepoStateKey: rsk,
					CommitHash:   "badc0de",
				}
				err = ds.Get(ctx, rc)
				So(err, ShouldBeNil)
				So(rc.GetNotificationState("rules"), ShouldEqual, "BUG=12345")
				So(rc.NotifiedAll, ShouldBeTrue)

			})
			Convey("Exceeded retries", func() {
				rsk := ds.KeyForObj(ctx, repoState)
				testClients.Monorail = rules.MockMonorailClient{
					Ii: &monorail.InsertIssueResponse{
						Issue: &monorail.Issue{
							Id: 12345,
						},
					},
				}
				rc := &rules.RelevantCommit{
					RepoStateKey:     rsk,
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

				err = notifyAboutViolations(ctx, cfg, repoState, testClients)
				So(err, ShouldBeNil)
				rc = &rules.RelevantCommit{
					RepoStateKey: rsk,
					CommitHash:   "b00b00",
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
					Notification: rules.CommentOnBugToAcknowledgeMerge{},
				}},
				Metadata: "MilestoneNumber:70",
			}
			// TODO: Do not mutate global state.
			rules.GetRuleMap()["old-repo-ack"] = cfg
			repoState := &rules.RepoState{
				RepoURL:            "https://old.googlesource.com/old-ack.git/+/master",
				LastKnownCommit:    "123456",
				LastRelevantCommit: "999999",
			}
			ds.Put(ctx, repoState)
			rsk := ds.KeyForObj(ctx, repoState)
			rc := &rules.RelevantCommit{
				RepoStateKey: rsk,
				CommitHash:   "badc0de",
				Status:       rules.AuditCompletedWithActionRequired,
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

			err = notifyAboutViolations(ctx, cfg, repoState, testClients)
			So(err, ShouldBeNil)

			rc = &rules.RelevantCommit{
				RepoStateKey: rsk,
				CommitHash:   "badc0de",
			}
			err = ds.Get(ctx, rc)
			So(rc.GetNotificationState("rulesAck"), ShouldEqual, "Comment posted on BUG(S)=8675389")
			So(rc.NotifiedAll, ShouldBeTrue)
		})
	})
}
