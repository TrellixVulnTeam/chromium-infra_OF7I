// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package rpc

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
	"go.chromium.org/luci/gae/impl/memory"
	"go.chromium.org/luci/server/auth"
	"go.chromium.org/luci/server/auth/authtest"
	"go.chromium.org/luci/server/secrets"
	"go.chromium.org/luci/server/secrets/testsecrets"
	"go.chromium.org/luci/server/span"
	"google.golang.org/grpc/codes"
	grpcStatus "google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"infra/appengine/weetbix/internal/bugs"
	"infra/appengine/weetbix/internal/clustering"
	"infra/appengine/weetbix/internal/clustering/algorithms/testname"
	"infra/appengine/weetbix/internal/clustering/rules"
	"infra/appengine/weetbix/internal/config"
	configpb "infra/appengine/weetbix/internal/config/proto"
	"infra/appengine/weetbix/internal/testutil"
	pb "infra/appengine/weetbix/proto/v1"
)

func TestRules(t *testing.T) {
	Convey("With Server", t, func() {
		ctx := testutil.SpannerTestContext(t)

		// For user identification.
		ctx = authtest.MockAuthConfig(ctx)
		ctx = auth.WithState(ctx, &authtest.FakeState{
			Identity:       "user:someone@example.com",
			IdentityGroups: []string{"weetbix-access"},
		})
		ctx = secrets.Use(ctx, &testsecrets.Store{})

		// Provides datastore implementation needed for project config.
		ctx = memory.Use(ctx)

		srv := NewRulesSever()

		ruleManagedBuilder := rules.NewRule(0).
			WithProject(testProject).
			WithBug(bugs.BugID{System: "monorail", ID: "monorailproject/111"})
		ruleManaged := ruleManagedBuilder.Build()
		ruleTwoProject := rules.NewRule(1).
			WithProject(testProject).
			WithBug(bugs.BugID{System: "monorail", ID: "monorailproject/222"}).
			WithBugManaged(false).
			Build()
		ruleTwoProjectOther := rules.NewRule(2).
			WithProject("otherproject").
			WithBug(bugs.BugID{System: "monorail", ID: "monorailproject/222"}).
			Build()
		ruleUnmanagedOther := rules.NewRule(3).
			WithProject("otherproject").
			WithBug(bugs.BugID{System: "monorail", ID: "monorailproject/444"}).
			WithBugManaged(false).
			Build()
		ruleManagedOther := rules.NewRule(4).
			WithProject("otherproject").
			WithBug(bugs.BugID{System: "monorail", ID: "monorailproject/555"}).
			WithBugManaged(true).
			Build()
		ruleBuganizer := rules.NewRule(5).
			WithProject(testProject).
			WithBug(bugs.BugID{System: "buganizer", ID: "666"}).
			Build()

		err := rules.SetRulesForTesting(ctx, []*rules.FailureAssociationRule{
			ruleManaged,
			ruleTwoProject,
			ruleTwoProjectOther,
			ruleUnmanagedOther,
			ruleManagedOther,
			ruleBuganizer,
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

		Convey("Unauthorised requests are rejected", func() {
			// Ensure no access to weetbix-access.
			ctx = auth.WithState(ctx, &authtest.FakeState{
				Identity: "user:someone@example.com",
				// Not a member of weetbix-access.
				IdentityGroups: []string{"other-group"},
			})

			// Make some request (the request should not matter, as
			// a common decorator is used for all requests.)
			request := &pb.GetRuleRequest{
				Name: fmt.Sprintf("projects/%s/rules/%s", ruleManaged.Project, ruleManaged.RuleID),
			}

			rule, err := srv.Get(ctx, request)
			st, _ := grpcStatus.FromError(err)
			So(st.Code(), ShouldEqual, codes.PermissionDenied)
			So(st.Message(), ShouldEqual, "not a member of weetbix-access")
			So(rule, ShouldBeNil)
		})
		Convey("Get", func() {
			Convey("Rule exists", func() {
				Convey("Read rule with Monorail bug", func() {
					request := &pb.GetRuleRequest{
						Name: fmt.Sprintf("projects/%s/rules/%s", ruleManaged.Project, ruleManaged.RuleID),
					}

					rule, err := srv.Get(ctx, request)
					So(err, ShouldBeNil)
					So(rule, ShouldResembleProto, createRulePB(ruleManaged, cfg))

					// Also verify createRulePB works as expected, so we do not need
					// to test that again in later tests.
					So(rule, ShouldResembleProto, &pb.Rule{
						Name:           fmt.Sprintf("projects/%s/rules/%s", ruleManaged.Project, ruleManaged.RuleID),
						Project:        ruleManaged.Project,
						RuleId:         ruleManaged.RuleID,
						RuleDefinition: ruleManaged.RuleDefinition,
						Bug: &pb.AssociatedBug{
							System:   "monorail",
							Id:       "monorailproject/111",
							LinkText: "mybug.com/111",
							Url:      "https://monorailhost.com/p/monorailproject/issues/detail?id=111",
						},
						IsActive:      true,
						IsManagingBug: true,
						SourceCluster: &pb.ClusterId{
							Algorithm: ruleManaged.SourceCluster.Algorithm,
							Id:        ruleManaged.SourceCluster.ID,
						},
						CreateTime:              timestamppb.New(ruleManaged.CreationTime),
						CreateUser:              ruleManaged.CreationUser,
						LastUpdateTime:          timestamppb.New(ruleManaged.LastUpdated),
						LastUpdateUser:          ruleManaged.LastUpdatedUser,
						PredicateLastUpdateTime: timestamppb.New(ruleManaged.PredicateLastUpdated),
						Etag:                    ruleETag(ruleManaged),
					})
				})
				Convey("Read rule with Buganizer bug", func() {
					request := &pb.GetRuleRequest{
						Name: fmt.Sprintf("projects/%s/rules/%s", ruleBuganizer.Project, ruleBuganizer.RuleID),
					}

					rule, err := srv.Get(ctx, request)
					So(err, ShouldBeNil)
					So(rule, ShouldResembleProto, createRulePB(ruleBuganizer, cfg))

					// Also verify createRulePB works as expected, so we do not need
					// to test that again in later tests.
					So(rule, ShouldResembleProto, &pb.Rule{
						Name:           fmt.Sprintf("projects/%s/rules/%s", ruleBuganizer.Project, ruleBuganizer.RuleID),
						Project:        ruleBuganizer.Project,
						RuleId:         ruleBuganizer.RuleID,
						RuleDefinition: ruleBuganizer.RuleDefinition,
						Bug: &pb.AssociatedBug{
							System:   "buganizer",
							Id:       "666",
							LinkText: "b/666",
							Url:      "https://issuetracker.google.com/issues/666",
						},
						IsActive:      true,
						IsManagingBug: true,
						SourceCluster: &pb.ClusterId{
							Algorithm: ruleBuganizer.SourceCluster.Algorithm,
							Id:        ruleBuganizer.SourceCluster.ID,
						},
						CreateTime:              timestamppb.New(ruleBuganizer.CreationTime),
						CreateUser:              ruleBuganizer.CreationUser,
						LastUpdateTime:          timestamppb.New(ruleBuganizer.LastUpdated),
						LastUpdateUser:          ruleBuganizer.LastUpdatedUser,
						PredicateLastUpdateTime: timestamppb.New(ruleBuganizer.PredicateLastUpdated),
						Etag:                    ruleETag(ruleBuganizer),
					})
				})
			})
			Convey("Rule does not exist", func() {
				ruleID := strings.Repeat("00", 16)
				request := &pb.GetRuleRequest{
					Name: fmt.Sprintf("projects/%s/rules/%s", ruleManaged.Project, ruleID),
				}

				rule, err := srv.Get(ctx, request)
				st, _ := grpcStatus.FromError(err)
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
					ruleManaged,
					ruleBuganizer,
					rules.NewRule(2).WithProject(testProject).Build(),
					rules.NewRule(3).WithProject(testProject).Build(),
					rules.NewRule(4).WithProject(testProject).Build(),
					// In other project.
					ruleManagedOther,
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
						createRulePB(rs[4], cfg),
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
			Convey("With project not configured", func() {
				request := &pb.ListRulesRequest{
					Parent: "projects/not-exists",
				}

				// Run
				response, err := srv.List(ctx, request)

				// Verify
				st, _ := grpcStatus.FromError(err)
				So(st.Code(), ShouldEqual, codes.FailedPrecondition)
				So(st.Message(), ShouldEqual, "project does not exist in Weetbix")
				So(response, ShouldBeNil)
			})
		})
		Convey("Update", func() {
			request := &pb.UpdateRuleRequest{
				Rule: &pb.Rule{
					Name:           fmt.Sprintf("projects/%s/rules/%s", ruleManaged.Project, ruleManaged.RuleID),
					RuleDefinition: `test = "updated"`,
					Bug: &pb.AssociatedBug{
						System: "monorail",
						Id:     "monorailproject/2",
					},
					IsManagingBug: false,
					IsActive:      false,
				},
				UpdateMask: &fieldmaskpb.FieldMask{
					// On the client side, we use JSON equivalents, i.e. ruleDefinition,
					// bug, isActive, isManagingBug.
					Paths: []string{"rule_definition", "bug", "is_active", "is_managing_bug"},
				},
				Etag: ruleETag(ruleManaged),
			}

			Convey("Success", func() {
				Convey("Predicate updated", func() {
					rule, err := srv.Update(ctx, request)
					So(err, ShouldBeNil)

					storedRule, err := rules.Read(span.Single(ctx), testProject, ruleManaged.RuleID)
					So(err, ShouldBeNil)

					So(storedRule.LastUpdated, ShouldNotEqual, ruleManaged.LastUpdated)

					expectedRule := ruleManagedBuilder.
						WithRuleDefinition(`test = "updated"`).
						WithBug(bugs.BugID{System: "monorail", ID: "monorailproject/2"}).
						WithActive(false).
						WithBugManaged(false).
						// Accept whatever the new last updated time is.
						WithLastUpdated(storedRule.LastUpdated).
						WithLastUpdatedUser("someone@example.com").
						// The predicate last updated time should be the same as
						// the last updated time.
						WithPredicateLastUpdated(storedRule.LastUpdated).
						Build()

					// Verify the rule was correctly updated in the database.
					So(storedRule, ShouldResemble, expectedRule)

					// Verify the returned rule matches what was expected.
					So(rule, ShouldResembleProto, createRulePB(expectedRule, cfg))
				})
				Convey("Predicate not updated", func() {
					request.UpdateMask.Paths = []string{"bug"}
					request.Rule.Bug = &pb.AssociatedBug{
						System: "buganizer",
						Id:     "99999999",
					}

					rule, err := srv.Update(ctx, request)
					So(err, ShouldBeNil)

					storedRule, err := rules.Read(span.Single(ctx), testProject, ruleManaged.RuleID)
					So(err, ShouldBeNil)

					// Check the rule was updated, but that predicate last
					// updated time was NOT updated.
					So(storedRule.LastUpdated, ShouldNotEqual, ruleManaged.LastUpdated)

					expectedRule := ruleManagedBuilder.
						WithBug(bugs.BugID{System: "buganizer", ID: "99999999"}).
						// Accept whatever the new last updated time is.
						WithLastUpdated(storedRule.LastUpdated).
						WithLastUpdatedUser("someone@example.com").
						Build()

					// Verify the rule was correctly updated in the database.
					So(storedRule, ShouldResemble, expectedRule)

					// Verify the returned rule matches what was expected.
					So(rule, ShouldResembleProto, createRulePB(expectedRule, cfg))
				})
				Convey("Re-use of bug managed by another project", func() {
					request.UpdateMask.Paths = []string{"bug"}
					request.Rule.Bug = &pb.AssociatedBug{
						System: ruleManagedOther.BugID.System,
						Id:     ruleManagedOther.BugID.ID,
					}

					rule, err := srv.Update(ctx, request)
					So(err, ShouldBeNil)

					storedRule, err := rules.Read(span.Single(ctx), testProject, ruleManaged.RuleID)
					So(err, ShouldBeNil)

					// Check the rule was updated.
					So(storedRule.LastUpdated, ShouldNotEqual, ruleManaged.LastUpdated)

					expectedRule := ruleManagedBuilder.
						// Verify the bug was updated, but that IsManagingBug
						// was silently set to false, because ruleManagedOther
						// already controls the bug.
						WithBug(ruleManagedOther.BugID).
						WithBugManaged(false).
						// Accept whatever the new last updated time is.
						WithLastUpdated(storedRule.LastUpdated).
						WithLastUpdatedUser("someone@example.com").
						Build()

					// Verify the rule was correctly updated in the database.
					So(storedRule, ShouldResemble, expectedRule)

					// Verify the returned rule matches what was expected.
					So(rule, ShouldResembleProto, createRulePB(expectedRule, cfg))
				})
			})
			Convey("Concurrent Modification", func() {
				_, err := srv.Update(ctx, request)
				So(err, ShouldBeNil)

				// Attempt the same modification again without
				// requerying.
				rule, err := srv.Update(ctx, request)
				So(rule, ShouldBeNil)
				st, _ := grpcStatus.FromError(err)
				So(st.Code(), ShouldEqual, codes.Aborted)
			})
			Convey("Rule does not exist", func() {
				ruleID := strings.Repeat("00", 16)
				request.Rule.Name = fmt.Sprintf("projects/%s/rules/%s", testProject, ruleID)

				rule, err := srv.Update(ctx, request)
				So(rule, ShouldBeNil)
				st, _ := grpcStatus.FromError(err)
				So(st.Code(), ShouldEqual, codes.NotFound)
			})
			Convey("Validation error", func() {
				Convey("Invalid bug monorail project", func() {
					request.Rule.Bug.Id = "otherproject/2"

					rule, err := srv.Update(ctx, request)
					So(rule, ShouldBeNil)
					st, _ := grpcStatus.FromError(err)
					So(st.Code(), ShouldEqual, codes.InvalidArgument)
					So(st.Message(), ShouldEqual, "bug not in expected monorail project (monorailproject)")
				})
				Convey("Re-use of same bug in same project", func() {
					// Use the same bug as another rule.
					request.Rule.Bug = &pb.AssociatedBug{
						System: ruleTwoProject.BugID.System,
						Id:     ruleTwoProject.BugID.ID,
					}

					rule, err := srv.Update(ctx, request)
					So(rule, ShouldBeNil)
					st, _ := grpcStatus.FromError(err)
					So(st.Code(), ShouldEqual, codes.InvalidArgument)
					So(st.Message(), ShouldEqual,
						fmt.Sprintf("bug already used by a rule in the same project (%s/%s)",
							ruleTwoProject.Project, ruleTwoProject.RuleID))
				})
				Convey("Bug managed by another rule", func() {
					// Select a bug already managed by another rule.
					request.Rule.Bug = &pb.AssociatedBug{
						System: ruleManagedOther.BugID.System,
						Id:     ruleManagedOther.BugID.ID,
					}
					// Request we manage this bug.
					request.Rule.IsManagingBug = true
					request.UpdateMask.Paths = []string{"bug", "is_managing_bug"}

					rule, err := srv.Update(ctx, request)
					So(rule, ShouldBeNil)
					st, _ := grpcStatus.FromError(err)
					So(st.Code(), ShouldEqual, codes.InvalidArgument)
					So(st.Message(), ShouldEqual,
						fmt.Sprintf("bug already managed by a rule in another project (%s/%s)",
							ruleManagedOther.Project, ruleManagedOther.RuleID))
				})
				Convey("Invalid rule definition", func() {
					// Use an invalid failure association rule.
					request.Rule.RuleDefinition = ""

					rule, err := srv.Update(ctx, request)
					So(rule, ShouldBeNil)
					st, _ := grpcStatus.FromError(err)
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
					IsActive:      false,
					IsManagingBug: true,
					SourceCluster: &pb.ClusterId{
						Algorithm: testname.AlgorithmName,
						Id:        strings.Repeat("aa", 16),
					},
				},
			}

			Convey("Success", func() {
				expectedRuleBuilder := rules.NewRule(0).
					WithProject(testProject).
					WithRuleDefinition(`test = "create"`).
					WithActive(false).
					WithBugManaged(true).
					WithCreationUser("someone@example.com").
					WithLastUpdatedUser("someone@example.com").
					WithSourceCluster(clustering.ClusterID{
						Algorithm: testname.AlgorithmName,
						ID:        strings.Repeat("aa", 16),
					})

				Convey("Bug not managed by another rule", func() {
					// Re-use the same bug as a rule in another project,
					// where the other rule is not managing the bug.
					request.Rule.Bug = &pb.AssociatedBug{
						System: ruleUnmanagedOther.BugID.System,
						Id:     ruleUnmanagedOther.BugID.ID,
					}

					rule, err := srv.Create(ctx, request)
					So(err, ShouldBeNil)

					storedRule, err := rules.Read(span.Single(ctx), testProject, rule.RuleId)
					So(err, ShouldBeNil)

					expectedRule := expectedRuleBuilder.
						// Accept the randomly generated rule ID.
						WithRuleID(rule.RuleId).
						WithBug(ruleUnmanagedOther.BugID).
						// Accept whatever CreationTime was assigned, as it
						// is determined by Spanner commit time.
						// Rule spanner data access code tests already validate
						// this is populated correctly.
						WithCreationTime(storedRule.CreationTime).
						WithLastUpdated(storedRule.CreationTime).
						WithPredicateLastUpdated(storedRule.CreationTime).
						Build()

					// Verify the rule was correctly created in the database.
					So(storedRule, ShouldResemble, expectedRuleBuilder.Build())

					// Verify the returned rule matches our expectations.
					So(rule, ShouldResembleProto, createRulePB(expectedRule, cfg))
				})
				Convey("Bug managed by another rule", func() {
					// Re-use the same bug as a rule in another project,
					// where that rule is managing the bug.
					request.Rule.Bug = &pb.AssociatedBug{
						System: ruleManagedOther.BugID.System,
						Id:     ruleManagedOther.BugID.ID,
					}

					rule, err := srv.Create(ctx, request)
					So(err, ShouldBeNil)

					storedRule, err := rules.Read(span.Single(ctx), testProject, rule.RuleId)
					So(err, ShouldBeNil)

					expectedRule := expectedRuleBuilder.
						// Accept the randomly generated rule ID.
						WithRuleID(rule.RuleId).
						WithBug(ruleManagedOther.BugID).
						// Because another rule is managing the bug, this rule
						// should be silenlty stopped from managing the bug.
						WithBugManaged(false).
						// Accept whatever CreationTime was assigned.
						WithCreationTime(storedRule.CreationTime).
						WithLastUpdated(storedRule.CreationTime).
						WithPredicateLastUpdated(storedRule.CreationTime).
						Build()

					// Verify the rule was correctly created in the database.
					So(storedRule, ShouldResemble, expectedRuleBuilder.Build())

					// Verify the returned rule matches our expectations.
					So(rule, ShouldResembleProto, createRulePB(expectedRule, cfg))
				})
				Convey("Buganizer", func() {
					request.Rule.Bug = &pb.AssociatedBug{
						System: "buganizer",
						Id:     "1111111111",
					}

					rule, err := srv.Create(ctx, request)
					So(err, ShouldBeNil)

					storedRule, err := rules.Read(span.Single(ctx), testProject, rule.RuleId)
					So(err, ShouldBeNil)

					expectedRule := expectedRuleBuilder.
						// Accept the randomly generated rule ID.
						WithRuleID(rule.RuleId).
						WithBug(bugs.BugID{System: "buganizer", ID: "1111111111"}).
						// Accept whatever CreationTime was assigned, as it
						// is determined by Spanner commit time.
						// Rule spanner data access code tests already validate
						// this is populated correctly.
						WithCreationTime(storedRule.CreationTime).
						WithLastUpdated(storedRule.CreationTime).
						WithPredicateLastUpdated(storedRule.CreationTime).
						Build()

					// Verify the rule was correctly created in the database.
					So(storedRule, ShouldResemble, expectedRuleBuilder.Build())

					// Verify the returned rule matches our expectations.
					So(rule, ShouldResembleProto, createRulePB(expectedRule, cfg))
				})
			})
			Convey("Validation error", func() {
				Convey("Invalid bug monorail project", func() {
					request.Rule.Bug.Id = "otherproject/2"

					rule, err := srv.Create(ctx, request)
					So(rule, ShouldBeNil)
					st, _ := grpcStatus.FromError(err)
					So(st.Code(), ShouldEqual, codes.InvalidArgument)
					So(st.Message(), ShouldEqual,
						"bug not in expected monorail project (monorailproject)")
				})
				Convey("Re-use of same bug in same project", func() {
					// Use the same bug as another rule, in the same project.
					request.Rule.Bug = &pb.AssociatedBug{
						System: ruleTwoProject.BugID.System,
						Id:     ruleTwoProject.BugID.ID,
					}

					rule, err := srv.Create(ctx, request)
					So(rule, ShouldBeNil)
					st, _ := grpcStatus.FromError(err)
					So(st.Code(), ShouldEqual, codes.InvalidArgument)
					So(st.Message(), ShouldEqual,
						fmt.Sprintf("bug already used by a rule in the same project (%s/%s)",
							ruleTwoProject.Project, ruleTwoProject.RuleID))
				})
				Convey("Invalid rule definition", func() {
					// Use an invalid failure association rule.
					request.Rule.RuleDefinition = ""

					rule, err := srv.Create(ctx, request)
					So(rule, ShouldBeNil)
					st, _ := grpcStatus.FromError(err)
					So(st.Code(), ShouldEqual, codes.InvalidArgument)
					So(st.Message(), ShouldStartWith, "rule definition is not valid")
				})
			})
		})
		Convey("LookupBug", func() {
			Convey("Exists None", func() {
				request := &pb.LookupBugRequest{
					System: "monorail",
					Id:     "notexists/1",
				}

				response, err := srv.LookupBug(ctx, request)
				So(err, ShouldBeNil)
				So(response, ShouldResembleProto, &pb.LookupBugResponse{
					Rules: []string{},
				})
			})
			Convey("Exists One", func() {
				request := &pb.LookupBugRequest{
					System: ruleManaged.BugID.System,
					Id:     ruleManaged.BugID.ID,
				}

				response, err := srv.LookupBug(ctx, request)
				So(err, ShouldBeNil)
				So(response, ShouldResembleProto, &pb.LookupBugResponse{
					Rules: []string{
						fmt.Sprintf("projects/%s/rules/%s",
							ruleManaged.Project, ruleManaged.RuleID),
					},
				})
			})
			Convey("Exists Many", func() {
				request := &pb.LookupBugRequest{
					System: ruleTwoProject.BugID.System,
					Id:     ruleTwoProject.BugID.ID,
				}

				response, err := srv.LookupBug(ctx, request)
				So(err, ShouldBeNil)
				So(response, ShouldResembleProto, &pb.LookupBugResponse{
					Rules: []string{
						// Rules are returned alphabetically by project.
						fmt.Sprintf("projects/otherproject/rules/%s", ruleTwoProjectOther.RuleID),
						fmt.Sprintf("projects/testproject/rules/%s", ruleTwoProject.RuleID),
					},
				})
			})
		})
	})
}
