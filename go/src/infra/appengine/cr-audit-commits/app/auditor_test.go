// Copyright 2017 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"context"

	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/gae/impl/memory"
	ds "go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/common/proto/git"
	gitilespb "go.chromium.org/luci/common/proto/gitiles"
	"go.chromium.org/luci/server/router"

	"infra/appengine/cr-audit-commits/app/rules"
)

type errorRule struct{}

// GetName returns the name of the rule.
func (rule errorRule) GetName() string {
	return "Dummy Rule"
}

// Run return errors if the commit hasn't been audited.
func (rule errorRule) Run(c context.Context, ap *rules.AuditParams, rc *rules.RelevantCommit, cs *rules.Clients) (*rules.RuleResult, error) {
	if rc.Status == rules.AuditScheduled {
		return nil, fmt.Errorf("error rule")
	}
	return &rules.RuleResult{
		RuleName:         "Dummy rule",
		RuleResultStatus: rules.RuleFailed,
		Message:          "",
		MetaData:         "",
	}, fmt.Errorf("error rule")
}

func dummyNotifier(ctx context.Context, cfg *rules.RefConfig, rc *rules.RelevantCommit, cs *rules.Clients, state string) (string, error) {
	return "NotificationSent", nil
}

func TestAuditor(t *testing.T) {

	Convey("CommitScanner handler test", t, func() {
		ctx := memory.Use(context.Background())

		auditorPath := "/_task/auditor"

		withTestingContext := func(c *router.Context, next router.Handler) {
			c.Context = ctx
			ds.GetTestable(ctx).CatchupIndexes()
			next(c)
		}

		r := router.New()
		r.GET(auditorPath, router.NewMiddlewareChain(withTestingContext), Auditor)
		srv := httptest.NewServer(r)
		client := &http.Client{}
		auditorTestClients = &rules.Clients{}
		Convey("Unknown Ref", func() {
			resp, err := client.Get(srv.URL + auditorPath + "?refUrl=unknown")
			So(err, ShouldBeNil)
			So(resp.StatusCode, ShouldEqual, 400)

		})
		Convey("Dummy Repo", func() {
			// TODO: Do not mutate global state.
			rules.GetRuleMap()["dummy-repo"] = &rules.RefConfig{
				BaseRepoURL:    "https://dummy.googlesource.com/dummy.git",
				GerritURL:      "https://dummy-review.googlesource.com",
				BranchName:     "refs/heads/master",
				StartingCommit: "000000",
				Rules: map[string]rules.AccountRules{"rules": {
					Account: "dummy@test.com",
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
					NotificationFunction: dummyNotifier,
				}},
			}
			escapedRepoURL := url.QueryEscape("https://dummy.googlesource.com/dummy.git/+/refs/heads/master")
			gitilesMockClient := gitilespb.NewMockGitilesClient(gomock.NewController(t))
			auditorTestClients.GitilesFactory = func(host string, httpClient *http.Client) (gitilespb.GitilesClient, error) {
				return gitilesMockClient, nil
			}
			Convey("Test scanning", func() {
				ds.Put(ctx, &rules.RepoState{
					RepoURL:            "https://dummy.googlesource.com/dummy.git/+/refs/heads/master",
					ConfigName:         "dummy-repo",
					LastKnownCommit:    "123456",
					LastRelevantCommit: "999999",
				})

				Convey("No revisions", func() {
					gitilesMockClient.EXPECT().Log(gomock.Any(), &gitilespb.LogRequest{
						Project:            "dummy",
						Committish:         "refs/heads/master",
						ExcludeAncestorsOf: "123456",
						PageSize:           6000,
					}).Return(&gitilespb.LogResponse{
						Log: []*git.Commit{},
					}, nil)
					resp, err := client.Get(srv.URL + auditorPath + "?refUrl=" + escapedRepoURL)
					So(err, ShouldBeNil)
					So(resp.StatusCode, ShouldEqual, 200)
					rs := &rules.RepoState{RepoURL: "https://dummy.googlesource.com/dummy.git/+/refs/heads/master"}
					err = ds.Get(ctx, rs)
					So(err, ShouldBeNil)
					So(rs.LastKnownCommit, ShouldEqual, "123456")
					So(rs.LastRelevantCommit, ShouldEqual, "999999")
				})
				Convey("No interesting revisions", func() {
					gitilesMockClient.EXPECT().Log(gomock.Any(), &gitilespb.LogRequest{
						Project:            "dummy",
						Committish:         "refs/heads/master",
						ExcludeAncestorsOf: "123456",
						PageSize:           6000,
					}).Return(&gitilespb.LogResponse{
						Log: []*git.Commit{{Id: "abcdef000123123"}},
					}, nil)
					resp, err := client.Get(srv.URL + auditorPath + "?refUrl=" + escapedRepoURL)
					So(err, ShouldBeNil)
					So(resp.StatusCode, ShouldEqual, 200)
					rs := &rules.RepoState{RepoURL: "https://dummy.googlesource.com/dummy.git/+/refs/heads/master"}
					err = ds.Get(ctx, rs)
					So(err, ShouldBeNil)
					So(rs.LastKnownCommit, ShouldEqual, "abcdef000123123")
					So(rs.LastRelevantCommit, ShouldEqual, "999999")
				})
				Convey("Interesting revisions", func() {
					gitilesMockClient.EXPECT().Log(gomock.Any(), &gitilespb.LogRequest{
						Project:            "dummy",
						Committish:         "refs/heads/master",
						ExcludeAncestorsOf: "123456",
						PageSize:           6000,
					}).Return(&gitilespb.LogResponse{
						Log: []*git.Commit{
							{Id: "deadbeef"},
							{
								Id: "c001c0de",
								Author: &git.Commit_User{
									Email: "dummy@test.com",
									Time:  rules.MustGitilesTime("Sun Sep 03 00:56:34 2017"),
								},
								Committer: &git.Commit_User{
									Email: "dummy@test.com",
									Time:  rules.MustGitilesTime("Sun Sep 03 00:56:34 2017"),
								},
							},
						},
					}, nil)
					resp, err := client.Get(srv.URL + auditorPath + "?refUrl=" + escapedRepoURL)
					So(err, ShouldBeNil)
					So(resp.StatusCode, ShouldEqual, 200)
					rs := &rules.RepoState{RepoURL: "https://dummy.googlesource.com/dummy.git/+/refs/heads/master"}
					err = ds.Get(ctx, rs)
					So(err, ShouldBeNil)
					So(rs.LastKnownCommit, ShouldEqual, "deadbeef")
					So(rs.LastRelevantCommit, ShouldEqual, "c001c0de")
					rc := &rules.RelevantCommit{
						RepoStateKey: ds.KeyForObj(ctx, rs),
						CommitHash:   "c001c0de",
					}
					err = ds.Get(ctx, rc)
					So(err, ShouldBeNil)
					So(rc.PreviousRelevantCommit, ShouldEqual, "999999")
				})
			})
			Convey("Test auditing", func() {
				repoState := &rules.RepoState{
					ConfigName:         "dummy-repo",
					RepoURL:            "https://dummy.googlesource.com/dummy.git/+/refs/heads/master",
					LastKnownCommit:    "222222",
					LastRelevantCommit: "222222",
				}
				err := ds.Put(ctx, repoState)
				rsk := ds.KeyForObj(ctx, repoState)

				So(err, ShouldBeNil)
				gitilesMockClient.EXPECT().Log(gomock.Any(), &gitilespb.LogRequest{
					Project:            "dummy",
					Committish:         "refs/heads/master",
					ExcludeAncestorsOf: "222222",
					PageSize:           6000,
				}).Return(&gitilespb.LogResponse{
					Log: []*git.Commit{},
				}, nil)

				Convey("No commits", func() {
					resp, err := client.Get(srv.URL + auditorPath + "?refUrl=" + escapedRepoURL)
					So(err, ShouldBeNil)
					So(resp.StatusCode, ShouldEqual, 200)
				})
				Convey("With commits", func() {
					for i := 0; i < 10; i++ {
						rc := &rules.RelevantCommit{
							RepoStateKey:  rsk,
							CommitHash:    fmt.Sprintf("%02d%02d%02d", i, i, i),
							Status:        rules.AuditScheduled,
							AuthorAccount: "dummy@test.com",
						}
						err := ds.Put(ctx, rc)
						So(err, ShouldBeNil)
					}
					Convey("All pass", func() {
						resp, err := client.Get(srv.URL + auditorPath + "?refUrl=" + escapedRepoURL)
						So(err, ShouldBeNil)
						So(resp.StatusCode, ShouldEqual, 200)
						for i := 0; i < 10; i++ {
							rc := &rules.RelevantCommit{
								RepoStateKey: rsk,
								CommitHash:   fmt.Sprintf("%02d%02d%02d", i, i, i),
							}
							err := ds.Get(ctx, rc)
							So(err, ShouldBeNil)
							So(rc.Status, ShouldEqual, rules.AuditCompleted)
						}
					})
					Convey("Some fail", func() {
						// TODO: Do not depend on global state.
						dummyRuleTmp := rules.GetRuleMap()["dummy-repo"].Rules["rules"].Rules[0].(rules.DummyRule)
						dummyRuleTmp.Result.RuleResultStatus = rules.RuleFailed
						resp, err := client.Get(srv.URL + auditorPath + "?refUrl=" + escapedRepoURL)
						So(err, ShouldBeNil)
						So(resp.StatusCode, ShouldEqual, 200)
						for i := 0; i < 10; i++ {
							rc := &rules.RelevantCommit{
								RepoStateKey: rsk,
								CommitHash:   fmt.Sprintf("%02d%02d%02d", i, i, i),
							}
							err := ds.Get(ctx, rc)
							So(err, ShouldBeNil)
							So(rc.Status, ShouldEqual, rules.AuditCompletedWithActionRequired)
						}
					})
					Convey("Some error", func() {
						// TODO: Do not mutate global state.
						rules.GetRuleMap()["dummy-repo"].Rules["rules"].Rules[0] = errorRule{}
						resp, err := client.Get(srv.URL + auditorPath + "?refUrl=" + escapedRepoURL)
						So(err, ShouldBeNil)
						So(resp.StatusCode, ShouldEqual, 200)
						for i := 0; i < 10; i++ {
							rc := &rules.RelevantCommit{
								RepoStateKey: rsk,
								CommitHash:   fmt.Sprintf("%02d%02d%02d", i, i, i),
							}
							err := ds.Get(ctx, rc)
							So(err, ShouldBeNil)
							So(rc.Status, ShouldEqual, rules.AuditScheduled)
							So(rc.Retries, ShouldEqual, 1)
						}
					})

				})
			})
			srv.Close()
		})
	})
}
