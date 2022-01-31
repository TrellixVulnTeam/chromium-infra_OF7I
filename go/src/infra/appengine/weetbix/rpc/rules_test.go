// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package rpc

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"infra/appengine/weetbix/internal/bugs"
	"infra/appengine/weetbix/internal/clustering"
	"infra/appengine/weetbix/internal/clustering/algorithms/testname"
	"infra/appengine/weetbix/internal/clustering/rules"
	"infra/appengine/weetbix/internal/config"
	configpb "infra/appengine/weetbix/internal/config/proto"
	"infra/appengine/weetbix/internal/testutil"
	pb "infra/appengine/weetbix/proto/v1"

	. "github.com/smartystreets/goconvey/convey"

	"go.chromium.org/luci/gae/impl/memory"
	"go.chromium.org/luci/grpc/appstatus"
	"go.chromium.org/luci/server/auth"
	"go.chromium.org/luci/server/auth/authtest"
	"go.chromium.org/luci/server/secrets"
	"go.chromium.org/luci/server/secrets/testsecrets"
	"go.chromium.org/luci/server/span"

	. "go.chromium.org/luci/common/testing/assertions"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestRules(t *testing.T) {
	Convey("With Server", t, func() {
		ctx := testutil.SpannerTestContext(t)

		// For user identification and XSRF Tokens.
		ctx = authtest.MockAuthConfig(ctx)
		ctx = auth.WithState(ctx, &authtest.FakeState{
			Identity: "user:someone@example.com",
		})
		ctx = secrets.Use(ctx, &testsecrets.Store{})

		// Provides datastore implementation needed for project config.
		ctx = memory.Use(ctx)

		srv := &Rules{}

		ruleOne := rules.NewRule(0).
			WithProject(testProject).
			WithBug(bugs.BugID{System: "monorail", ID: "monorailproject/151"}).
			Build()
		ruleTwo := rules.NewRule(1).
			WithProject("otherproject").
			WithBug(bugs.BugID{System: "monorail", ID: "monorailproject/333"}).
			Build()
		err := rules.SetRulesForTesting(ctx, []*rules.FailureAssociationRule{
			ruleOne,
			ruleTwo,
		})
		So(err, ShouldBeNil)

		cfg := &configpb.ProjectConfig{
			Monorail: &configpb.MonorailProject{
				Project:          "monorailproject",
				DisplayPrefix:    "mybug.com",
				MonorailHostname: "monorailhost.com",
			},
		}
		err = config.SetTestProjectConfig(ctx, map[string]*configpb.ProjectConfig{
			"testproject": cfg,
		})
		So(err, ShouldBeNil)

		Convey("Get", func() {
			Convey("Exists", func() {
				request := &pb.GetRuleRequest{
					Name: fmt.Sprintf("projects/%s/rules/%s", ruleOne.Project, ruleOne.RuleID),
				}

				rule, err := srv.Get(ctx, request)
				So(err, ShouldBeNil)
				So(rule, ShouldResembleProto, createRulePB(ruleOne, cfg))

				// Also verify createRule works as expected, so we do not need
				// to test that again in later tests.
				So(rule, ShouldResembleProto, &pb.Rule{
					Name:           fmt.Sprintf("projects/%s/rules/%s", ruleOne.Project, ruleOne.RuleID),
					Project:        ruleOne.Project,
					RuleId:         ruleOne.RuleID,
					RuleDefinition: ruleOne.RuleDefinition,
					Bug: &pb.AssociatedBug{
						System:   "monorail",
						Id:       "monorailproject/151",
						LinkText: "mybug.com/151",
						Url:      "https://monorailhost.com/p/monorailproject/issues/detail?id=151",
					},
					IsActive: true,
					SourceCluster: &pb.ClusterId{
						Algorithm: ruleOne.SourceCluster.Algorithm,
						Id:        ruleOne.SourceCluster.ID,
					},
					CreateTime:     timestamppb.New(ruleOne.CreationTime),
					CreateUser:     ruleOne.CreationUser,
					LastUpdateTime: timestamppb.New(ruleOne.LastUpdated),
					LastUpdateUser: ruleOne.LastUpdatedUser,
					Etag:           ruleETag(ruleOne),
				})
			})
			Convey("Not Exists", func() {
				ruleID := strings.Repeat("00", 16)
				request := &pb.GetRuleRequest{
					Name: fmt.Sprintf("projects/%s/rules/%s", ruleOne.Project, ruleID),
				}

				rule, err := srv.Get(ctx, request)
				st, ok := appstatus.Get(err)
				So(ok, ShouldBeTrue)
				So(st.Code(), ShouldEqual, codes.NotFound)
				So(rule, ShouldBeNil)
			})
		})
		Convey("List", func() {
			request := &pb.ListRulesRequest{
				Parent: fmt.Sprintf("projects/%s", testProject),
			}
			Convey("Non-Empty", func() {
				rs := []*rules.FailureAssociationRule{
					ruleOne,
					rules.NewRule(2).WithProject(testProject).Build(),
					rules.NewRule(3).WithProject(testProject).Build(),
					rules.NewRule(4).WithProject(testProject).Build(),
					// In other project.
					ruleTwo,
				}
				err := rules.SetRulesForTesting(ctx, rs)
				So(err, ShouldBeNil)

				response, err := srv.List(ctx, request)
				So(err, ShouldBeNil)

				expected := &pb.ListRulesResponse{
					Rules: []*pb.Rule{
						createRulePB(rs[0], cfg),
						createRulePB(rs[1], cfg),
						createRulePB(rs[2], cfg),
						createRulePB(rs[3], cfg),
					},
				}
				sort.Slice(expected.Rules, func(i, j int) bool {
					return expected.Rules[i].RuleId < expected.Rules[j].RuleId
				})
				sort.Slice(response.Rules, func(i, j int) bool {
					return response.Rules[i].RuleId < response.Rules[j].RuleId
				})
				So(response, ShouldResembleProto, expected)
			})
			Convey("Empty", func() {
				err := rules.SetRulesForTesting(ctx, nil)
				So(err, ShouldBeNil)

				response, err := srv.List(ctx, request)
				So(err, ShouldBeNil)

				expected := &pb.ListRulesResponse{}
				So(response, ShouldResembleProto, expected)
			})
		})
		Convey("Update", func() {
			request := &pb.UpdateRuleRequest{
				Rule: &pb.Rule{
					Name:           fmt.Sprintf("projects/%s/rules/%s", ruleOne.Project, ruleOne.RuleID),
					RuleDefinition: `test = "updated"`,
					Bug: &pb.AssociatedBug{
						System: "monorail",
						Id:     "monorailproject/2",
					},
					IsActive: false,
				},
				UpdateMask: &fieldmaskpb.FieldMask{
					// On the client side, we use JSON equivalents, i.e. ruleDefinition,
					// isActive.
					Paths: []string{"rule_definition", "bug", "is_active"},
				},
				Etag: ruleETag(ruleOne),
			}

			Convey("Success", func() {
				rule, err := srv.Update(ctx, request)
				So(err, ShouldBeNil)

				storedRule, err := rules.Read(span.Single(ctx), testProject, ruleOne.RuleID)
				So(err, ShouldBeNil)

				So(storedRule.LastUpdated, ShouldNotEqual, ruleOne.LastUpdated)

				expectedRule := rules.NewRule(0).
					WithProject(testProject).
					WithRuleDefinition(`test = "updated"`).
					WithBug(bugs.BugID{System: "monorail", ID: "monorailproject/2"}).
					WithActive(false).
					// Accept whatever the new last updated time is.
					WithLastUpdated(storedRule.LastUpdated).
					WithLastUpdatedUser("someone@example.com").
					Build()

				// Verify the rule was correctly updated in the database.
				So(storedRule, ShouldResemble, expectedRule)

				// Verify the returned rule matches what was expected.
				So(rule, ShouldResembleProto, createRulePB(expectedRule, cfg))
			})
			Convey("Concurrent Modification", func() {
				_, err := srv.Update(ctx, request)
				So(err, ShouldBeNil)

				// Attempt the same modification again without
				// requerying.
				rule, err := srv.Update(ctx, request)
				So(rule, ShouldBeNil)
				st, _ := appstatus.Get(err)
				So(st.Code(), ShouldEqual, codes.Aborted)
			})
			Convey("Rule does not exist", func() {
				ruleID := strings.Repeat("00", 16)
				request.Rule.Name = fmt.Sprintf("projects/%s/rules/%s", ruleOne.Project, ruleID)

				rule, err := srv.Update(ctx, request)
				So(rule, ShouldBeNil)
				st, _ := appstatus.Get(err)
				So(st.Code(), ShouldEqual, codes.NotFound)
			})
			Convey("Validation error", func() {
				Convey("Invalid bug monorail project", func() {
					request.Rule.Bug.Id = "otherproject/2"

					rule, err := srv.Update(ctx, request)
					So(rule, ShouldBeNil)
					st, _ := appstatus.Get(err)
					So(st.Code(), ShouldEqual, codes.InvalidArgument)
					So(st.Message(), ShouldEqual, "bug not in expected monorail project (monorailproject)")
				})
				Convey("Re-use of same bug", func() {
					// Use the same bug as another rule.
					request.Rule.Bug = &pb.AssociatedBug{
						System: ruleTwo.BugID.System,
						Id:     ruleTwo.BugID.ID,
					}

					rule, err := srv.Update(ctx, request)
					So(rule, ShouldBeNil)
					st, _ := appstatus.Get(err)
					So(st.Code(), ShouldEqual, codes.InvalidArgument)
					So(st.Message(), ShouldStartWith, "bug already used by another failure association rule")
				})
				Convey("Invalid rule definition", func() {
					// Use an invalid failure association rule.
					request.Rule.RuleDefinition = ""

					rule, err := srv.Update(ctx, request)
					So(rule, ShouldBeNil)
					st, _ := appstatus.Get(err)
					So(st.Code(), ShouldEqual, codes.InvalidArgument)
					So(st.Message(), ShouldStartWith, "rule definition is not valid")
				})
			})
		})
		Convey("Create", func() {
			request := &pb.CreateRuleRequest{
				Parent: fmt.Sprintf("projects/%s", testProject),
				Rule: &pb.Rule{
					RuleDefinition: `test = "create"`,
					Bug: &pb.AssociatedBug{
						System: "monorail",
						Id:     "monorailproject/2",
					},
					IsActive: false,
					SourceCluster: &pb.ClusterId{
						Algorithm: testname.AlgorithmName,
						Id:        strings.Repeat("aa", 16),
					},
				},
			}

			Convey("Success", func() {
				rule, err := srv.Create(ctx, request)
				So(err, ShouldBeNil)

				storedRule, err := rules.Read(span.Single(ctx), testProject, rule.RuleId)
				So(err, ShouldBeNil)

				expectedRule := rules.NewRule(0).
					WithProject(testProject).
					// Accept the randomly generated rule ID.
					WithRuleID(rule.RuleId).
					WithRuleDefinition(`test = "create"`).
					WithBug(bugs.BugID{System: "monorail", ID: "monorailproject/2"}).
					WithActive(false).
					// Accept whatever CreationTime was assigned, as it
					// is determined by Spanner commit time.
					// Rule spanner data access code tests already validate
					// this is populated correctly.
					WithCreationTime(storedRule.CreationTime).
					WithCreationUser("someone@example.com").
					// LastUpdated time should be the same as Creation Time.
					WithLastUpdated(storedRule.CreationTime).
					WithLastUpdatedUser("someone@example.com").
					WithSourceCluster(clustering.ClusterID{
						Algorithm: testname.AlgorithmName,
						ID:        strings.Repeat("aa", 16),
					}).
					Build()

				// Verify the rule was correctly created in the database.
				So(storedRule, ShouldResemble, expectedRule)

				// Verify the returned rule matches our expectations.
				So(rule, ShouldResembleProto, createRulePB(expectedRule, cfg))
			})
			Convey("Validation error", func() {
				Convey("Invalid bug monorail project", func() {
					request.Rule.Bug.Id = "otherproject/2"

					rule, err := srv.Create(ctx, request)
					So(rule, ShouldBeNil)
					st, _ := appstatus.Get(err)
					So(st.Code(), ShouldEqual, codes.InvalidArgument)
					So(st.Message(), ShouldEqual, "bug not in expected monorail project (monorailproject)")
				})
				Convey("Re-use of same bug", func() {
					// Use the same bug as another rule.
					request.Rule.Bug = &pb.AssociatedBug{
						System: ruleTwo.BugID.System,
						Id:     ruleTwo.BugID.ID,
					}

					rule, err := srv.Create(ctx, request)
					So(rule, ShouldBeNil)
					st, _ := appstatus.Get(err)
					So(st.Code(), ShouldEqual, codes.InvalidArgument)
					So(st.Message(), ShouldStartWith, "bug already used by another failure association rule")
				})
				Convey("Invalid rule definition", func() {
					// Use an invalid failure association rule.
					request.Rule.RuleDefinition = ""

					rule, err := srv.Create(ctx, request)
					So(rule, ShouldBeNil)
					st, _ := appstatus.Get(err)
					So(st.Code(), ShouldEqual, codes.InvalidArgument)
					So(st.Message(), ShouldStartWith, "rule definition is not valid")
				})
			})
		})
	})
}
