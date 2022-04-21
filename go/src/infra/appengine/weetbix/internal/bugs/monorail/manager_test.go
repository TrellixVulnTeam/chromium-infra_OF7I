// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package monorail

import (
	"context"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
	"google.golang.org/genproto/protobuf/field_mask"

	"infra/appengine/weetbix/internal/bugs"
	"infra/appengine/weetbix/internal/clustering"
	mpb "infra/monorailv2/api/v3/api_proto"
)

func NewCreateRequest() *bugs.CreateRequest {
	cluster := &bugs.CreateRequest{
		Description: &clustering.ClusterDescription{
			Title:       "ClusterID",
			Description: "Tests are failing with reason: Some failure reason.",
		},
	}
	return cluster
}

func TestManager(t *testing.T) {
	t.Parallel()

	Convey("With Bug Manager", t, func() {
		ctx := context.Background()
		f := &FakeIssuesStore{
			NextID:            100,
			PriorityFieldName: "projects/chromium/fieldDefs/11",
		}
		user := AutomationUsers[0]
		cl, err := NewClient(UseFakeIssuesClient(ctx, f, user), "myhost")
		So(err, ShouldBeNil)
		monorailCfgs := ChromiumTestConfig()
		bm := NewBugManager(cl, "chops-weetbix-test", "luciproject", monorailCfgs)

		Convey("Create", func() {
			c := NewCreateRequest()
			c.Impact = ChromiumLowP1Impact()

			Convey("With reason-based failure cluster", func() {
				reason := `Expected equality of these values:
					"Expected_Value"
					my_expr.evaluate(123)
						Which is: "Unexpected_Value"`
				c.Description.Title = reason
				c.Description.Description = "A cluster of failures has been found with reason: " + reason

				bug, err := bm.Create(ctx, c)
				So(err, ShouldBeNil)
				So(bug, ShouldEqual, "chromium/100")
				So(len(f.Issues), ShouldEqual, 1)
				issue := f.Issues[0]

				So(issue.Issue, ShouldResembleProto, &mpb.Issue{
					Name:     "projects/chromium/issues/100",
					Summary:  "Tests are failing: Expected equality of these values: \"Expected_Value\" my_expr.evaluate(123) Which is: \"Unexpected_Value\"",
					Reporter: AutomationUsers[0],
					Owner:    &mpb.Issue_UserValue{User: ChromiumDefaultAssignee},
					State:    mpb.IssueContentState_ACTIVE,
					Status:   &mpb.Issue_StatusValue{Status: "Untriaged"},
					FieldValues: []*mpb.FieldValue{
						{
							// Type field.
							Field: "projects/chromium/fieldDefs/10",
							Value: "Bug",
						},
						{
							// Priority field.
							Field: "projects/chromium/fieldDefs/11",
							Value: "1",
						},
					},
					Labels: []*mpb.Issue_LabelValue{{
						Label: "Restrict-View-Google",
					}, {
						Label: "Weetbix-Auto-Filed",
					}},
				})
				So(len(issue.Comments), ShouldEqual, 2)
				So(issue.Comments[0].Content, ShouldContainSubstring, reason)
				So(issue.Comments[0].Content, ShouldNotContainSubstring, "ClusterIDShouldNotAppearInOutput")
				// Link to cluster page should appear in output.
				So(issue.Comments[1].Content, ShouldContainSubstring, "https://chops-weetbix-test.appspot.com/b/chromium/100")
				So(issue.NotifyCount, ShouldEqual, 1)
			})
			Convey("With test name failure cluster", func() {
				c.Description.Title = "ninja://:blink_web_tests/media/my-suite/my-test.html"
				c.Description.Description = "A test is failing " + c.Description.Title

				bug, err := bm.Create(ctx, c)
				So(err, ShouldBeNil)
				So(bug, ShouldEqual, "chromium/100")
				So(len(f.Issues), ShouldEqual, 1)
				issue := f.Issues[0]

				So(issue.Issue, ShouldResembleProto, &mpb.Issue{
					Name:     "projects/chromium/issues/100",
					Summary:  "Tests are failing: ninja://:blink_web_tests/media/my-suite/my-test.html",
					Reporter: AutomationUsers[0],
					Owner:    &mpb.Issue_UserValue{User: ChromiumDefaultAssignee},
					State:    mpb.IssueContentState_ACTIVE,
					Status:   &mpb.Issue_StatusValue{Status: "Untriaged"},
					FieldValues: []*mpb.FieldValue{
						{
							// Type field.
							Field: "projects/chromium/fieldDefs/10",
							Value: "Bug",
						},
						{
							// Priority field.
							Field: "projects/chromium/fieldDefs/11",
							Value: "1",
						},
					},
					Labels: []*mpb.Issue_LabelValue{{
						Label: "Restrict-View-Google",
					}, {
						Label: "Weetbix-Auto-Filed",
					}},
				})
				So(len(issue.Comments), ShouldEqual, 2)
				So(issue.Comments[0].Content, ShouldContainSubstring, "ninja://:blink_web_tests/media/my-suite/my-test.html")
				// Link to cluster page should appear in output.
				So(issue.Comments[1].Content, ShouldContainSubstring, "https://chops-weetbix-test.appspot.com/b/chromium/100")
				So(issue.NotifyCount, ShouldEqual, 1)
			})
			Convey("Does nothing if in simulation mode", func() {
				bm.Simulate = true
				_, err := bm.Create(ctx, c)
				So(err, ShouldEqual, bugs.ErrCreateSimulated)
				So(len(f.Issues), ShouldEqual, 0)
			})
		})
		Convey("Update", func() {
			c := NewCreateRequest()
			c.Impact = ChromiumP2Impact()
			bug, err := bm.Create(ctx, c)
			So(err, ShouldBeNil)
			So(bug, ShouldEqual, "chromium/100")
			So(len(f.Issues), ShouldEqual, 1)
			So(ChromiumTestIssuePriority(f.Issues[0].Issue), ShouldEqual, "2")

			bugToUpdate := &bugs.BugToUpdate{
				BugName: bug,
				Impact:  c.Impact,
			}
			bugsToUpdate := []*bugs.BugToUpdate{bugToUpdate}
			updateDoesNothing := func() {
				originalIssues := CopyIssuesStore(f)
				err := bm.Update(ctx, bugsToUpdate)
				So(err, ShouldBeNil)
				So(f, ShouldResembleIssuesStore, originalIssues)
			}
			// Create a monorail client that interacts with monorail
			// as an end-user. This is needed as we distinguish user
			// updates to the bug from system updates.
			user := "users/100"
			usercl, err := NewClient(UseFakeIssuesClient(ctx, f, user), "myhost")
			So(err, ShouldBeNil)

			Convey("If impact unchanged, does nothing", func() {
				updateDoesNothing()
			})
			Convey("If impact changed", func() {
				bugToUpdate.Impact = ChromiumP3Impact()
				Convey("Does not reduce priority if impact within hysteresis range", func() {
					bugToUpdate.Impact = ChromiumHighP3Impact()

					updateDoesNothing()
				})
				Convey("Reduces priority in response to reduced impact", func() {
					bugToUpdate.Impact = ChromiumP3Impact()
					originalNotifyCount := f.Issues[0].NotifyCount
					err := bm.Update(ctx, bugsToUpdate)
					So(err, ShouldBeNil)
					So(ChromiumTestIssuePriority(f.Issues[0].Issue), ShouldEqual, "3")

					So(f.Issues[0].Comments, ShouldHaveLength, 3)
					So(f.Issues[0].Comments[2].Content, ShouldEqual,
						"Because:\n"+
							"- Test Runs Failed (1-day) < 9, and\n"+
							"- Test Results Failed (1-day) < 90\n"+
							"Weetbix has decreased the bug priority from 2 to 3.")

					// Does not notify.
					So(f.Issues[0].NotifyCount, ShouldEqual, originalNotifyCount)

					// Verify repeated update has no effect.
					updateDoesNothing()
				})
				Convey("Does not increase priority if impact within hysteresis range", func() {
					bugToUpdate.Impact = ChromiumLowP1Impact()

					updateDoesNothing()
				})
				Convey("Increases priority in response to increased impact (single-step)", func() {
					bugToUpdate.Impact = ChromiumP1Impact()

					originalNotifyCount := f.Issues[0].NotifyCount
					err := bm.Update(ctx, bugsToUpdate)
					So(err, ShouldBeNil)
					So(ChromiumTestIssuePriority(f.Issues[0].Issue), ShouldEqual, "1")

					So(f.Issues[0].Comments, ShouldHaveLength, 3)
					So(f.Issues[0].Comments[2].Content, ShouldEqual,
						"Because:\n"+
							"- Test Results Failed (1-day) >= 550\n"+
							"Weetbix has increased the bug priority from 2 to 1.")

					// Notified the increase.
					So(f.Issues[0].NotifyCount, ShouldEqual, originalNotifyCount+1)

					// Verify repeated update has no effect.
					updateDoesNothing()
				})
				Convey("Increases priority in response to increased impact (multi-step)", func() {
					bugToUpdate.Impact = ChromiumP0Impact()

					originalNotifyCount := f.Issues[0].NotifyCount
					err := bm.Update(ctx, bugsToUpdate)
					So(err, ShouldBeNil)
					So(ChromiumTestIssuePriority(f.Issues[0].Issue), ShouldEqual, "0")

					expectedComment := "Because:\n" +
						"- Test Results Failed (1-day) >= 1000\n" +
						"Weetbix has increased the bug priority from 2 to 0."
					So(f.Issues[0].Comments, ShouldHaveLength, 3)
					So(f.Issues[0].Comments[2].Content, ShouldEqual, expectedComment)

					// Notified the increase.
					So(f.Issues[0].NotifyCount, ShouldEqual, originalNotifyCount+1)

					// Verify repeated update has no effect.
				})
				Convey("Does not adjust priority if priority manually set", func() {
					updateReq := updateBugPriorityRequest(f.Issues[0].Issue.Name, "0")
					err = usercl.ModifyIssues(ctx, updateReq)
					So(err, ShouldBeNil)
					So(ChromiumTestIssuePriority(f.Issues[0].Issue), ShouldEqual, "0")

					// Check the update sets the label.
					expectedIssue := CopyIssue(f.Issues[0].Issue)
					expectedIssue.Labels = append(expectedIssue.Labels, &mpb.Issue_LabelValue{
						Label: manualPriorityLabel,
					})
					SortLabels(expectedIssue.Labels)

					So(f.Issues[0].NotifyCount, ShouldEqual, 1)
					err = bm.Update(ctx, bugsToUpdate)
					So(err, ShouldBeNil)
					So(f.Issues[0].Issue, ShouldResembleProto, expectedIssue)

					// Does not notify.
					So(f.Issues[0].NotifyCount, ShouldEqual, 1)

					// Check repeated update does nothing more.
					updateDoesNothing()

					Convey("Unless manual priority cleared", func() {
						updateReq := removeLabelRequest(f.Issues[0].Issue.Name, manualPriorityLabel)
						err = usercl.ModifyIssues(ctx, updateReq)
						So(err, ShouldBeNil)
						So(hasLabel(f.Issues[0].Issue, manualPriorityLabel), ShouldBeFalse)

						err := bm.Update(ctx, bugsToUpdate)
						So(err, ShouldBeNil)
						So(ChromiumTestIssuePriority(f.Issues[0].Issue), ShouldEqual, "3")

						// Verify repeated update has no effect.
						updateDoesNothing()
					})
				})
				Convey("Does nothing if in simulation mode", func() {
					bm.Simulate = true
					updateDoesNothing()
				})
				Convey("Does nothing if Restrict-View-Google is unset", func() {
					// This requirement comes from security review, see crbug.com/1245877.
					updateReq := removeLabelRequest(f.Issues[0].Issue.Name, restrictViewLabel)
					err = usercl.ModifyIssues(ctx, updateReq)
					So(err, ShouldBeNil)
					So(hasLabel(f.Issues[0].Issue, restrictViewLabel), ShouldBeFalse)

					updateDoesNothing()
				})
			})
			Convey("If impact falls below lowest priority threshold", func() {
				bugToUpdate.Impact = ChromiumClosureImpact()
				Convey("Update leaves bug open if impact within hysteresis range", func() {
					bugToUpdate.Impact = ChromiumClosureHighImpact()

					// Update may reduce the priority from P1 to P3, but the
					// issue should be left open. This is because hysteresis on
					// priority and issue verified state is applied separately.
					err := bm.Update(ctx, bugsToUpdate)
					So(err, ShouldBeNil)
					So(f.Issues[0].Issue.Status.Status, ShouldEqual, UntriagedStatus)
				})
				Convey("Update closes bug", func() {
					err := bm.Update(ctx, bugsToUpdate)
					So(err, ShouldBeNil)
					So(f.Issues[0].Issue.Status.Status, ShouldEqual, VerifiedStatus)

					expectedComment := "Because:\n" +
						"- Test Results Failed (1-day) < 45, and\n" +
						"- Test Results Failed (3-day) < 272, and\n" +
						"- Test Results Failed (7-day) < 636\n" +
						"Weetbix is marking the issue verified."
					So(f.Issues[0].Comments, ShouldHaveLength, 3)
					So(f.Issues[0].Comments[2].Content, ShouldEqual, expectedComment)

					// Verify repeated update has no effect.
					updateDoesNothing()

					Convey("Does not reopen bug if impact within hysteresis range", func() {
						bugToUpdate.Impact = ChromiumP3LowImpact()

						updateDoesNothing()
					})

					Convey("If impact increases, bug is re-opened with correct priority", func() {
						bugToUpdate.Impact = ChromiumP3Impact()
						Convey("Issue has owner", func() {
							// Update issue owner.
							updateReq := updateOwnerRequest(f.Issues[0].Issue.Name, "users/100")
							err = usercl.ModifyIssues(ctx, updateReq)
							So(err, ShouldBeNil)
							So(f.Issues[0].Issue.Owner.GetUser(), ShouldEqual, "users/100")

							// Issue should return to "Assigned" status.
							err := bm.Update(ctx, bugsToUpdate)
							So(err, ShouldBeNil)
							So(f.Issues[0].Issue.Status.Status, ShouldEqual, AssignedStatus)
							So(ChromiumTestIssuePriority(f.Issues[0].Issue), ShouldEqual, "3")

							expectedComment := "Because:\n" +
								"- Test Results Failed (1-day) >= 55\n" +
								"Weetbix has re-opened the bug.\n\n" +
								"Because:\n" +
								"- Test Runs Failed (1-day) < 9, and\n" +
								"- Test Results Failed (1-day) < 90\n" +
								"Weetbix has decreased the bug priority from 2 to 3."
							So(f.Issues[0].Comments, ShouldHaveLength, 5)
							So(f.Issues[0].Comments[4].Content, ShouldEqual, expectedComment)

							// Verify repeated update has no effect.
							updateDoesNothing()
						})
						Convey("Issue has no owner", func() {
							// Remove owner.
							updateReq := updateOwnerRequest(f.Issues[0].Issue.Name, "")
							err = usercl.ModifyIssues(ctx, updateReq)
							So(err, ShouldBeNil)
							So(f.Issues[0].Issue.Owner.GetUser(), ShouldEqual, "")

							// Issue should return to "Untriaged" status.
							err := bm.Update(ctx, bugsToUpdate)
							So(err, ShouldBeNil)
							So(f.Issues[0].Issue.Status.Status, ShouldEqual, UntriagedStatus)
							So(ChromiumTestIssuePriority(f.Issues[0].Issue), ShouldEqual, "3")

							expectedComment := "Because:\n" +
								"- Test Results Failed (1-day) >= 55\n" +
								"Weetbix has re-opened the bug.\n\n" +
								"Because:\n" +
								"- Test Runs Failed (1-day) < 9, and\n" +
								"- Test Results Failed (1-day) < 90\n" +
								"Weetbix has decreased the bug priority from 2 to 3."
							So(f.Issues[0].Comments, ShouldHaveLength, 5)
							So(f.Issues[0].Comments[4].Content, ShouldEqual, expectedComment)

							// Verify repeated update has no effect.
							updateDoesNothing()
						})
					})
				})
			})
		})
	})
}

func updateOwnerRequest(name string, owner string) *mpb.ModifyIssuesRequest {
	return &mpb.ModifyIssuesRequest{
		Deltas: []*mpb.IssueDelta{
			{
				Issue: &mpb.Issue{
					Name: name,
					Owner: &mpb.Issue_UserValue{
						User: owner,
					},
				},
				UpdateMask: &field_mask.FieldMask{
					Paths: []string{"owner"},
				},
			},
		},
		CommentContent: "User comment.",
	}
}

func updateBugPriorityRequest(name string, priority string) *mpb.ModifyIssuesRequest {
	return &mpb.ModifyIssuesRequest{
		Deltas: []*mpb.IssueDelta{
			{
				Issue: &mpb.Issue{
					Name: name,
					FieldValues: []*mpb.FieldValue{
						{
							Field: "projects/chromium/fieldDefs/11",
							Value: priority,
						},
					},
				},
				UpdateMask: &field_mask.FieldMask{
					Paths: []string{"field_values"},
				},
			},
		},
		CommentContent: "User comment.",
	}
}

func removeLabelRequest(name string, label string) *mpb.ModifyIssuesRequest {
	return &mpb.ModifyIssuesRequest{
		Deltas: []*mpb.IssueDelta{
			{
				Issue: &mpb.Issue{
					Name: name,
				},
				UpdateMask:   &field_mask.FieldMask{},
				LabelsRemove: []string{label},
			},
		},
		CommentContent: "User comment.",
	}
}
