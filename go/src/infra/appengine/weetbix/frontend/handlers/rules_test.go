// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"infra/appengine/weetbix/internal/bugs"
	"infra/appengine/weetbix/internal/clustering/rules"
	"infra/appengine/weetbix/internal/config"
	configpb "infra/appengine/weetbix/internal/config/proto"
	"infra/appengine/weetbix/internal/testutil"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/luci/gae/impl/memory"
	"go.chromium.org/luci/server/auth"
	"go.chromium.org/luci/server/auth/authtest"
	"go.chromium.org/luci/server/auth/xsrf"
	"go.chromium.org/luci/server/secrets"
	"go.chromium.org/luci/server/secrets/testsecrets"
	"go.chromium.org/luci/server/span"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

const testProject = "testproject"

func TestRules(t *testing.T) {
	Convey("With Router", t, func() {
		ctx := testutil.SpannerTestContext(t)

		// For user identification and XSRF Tokens.
		ctx = authtest.MockAuthConfig(ctx)
		ctx = auth.WithState(ctx, &authtest.FakeState{
			Identity: "user:someone@example.com",
		})
		ctx = secrets.Use(ctx, &testsecrets.Store{})

		// Provides datastore implementation needed for project config.
		ctx = memory.Use(ctx)

		router := routerForTesting(ctx)

		ruleOne := rules.NewRule(0).
			WithProject(testProject).
			WithBug(bugs.BugID{System: "monorail", ID: "monorailproject/151"}).
			Build()
		ruleTwo := rules.NewRule(1).
			WithProject("otherproject").
			WithBug(bugs.BugID{System: "monorail", ID: "monorailproject/333"}).
			Build()
		rules.SetRulesForTesting(ctx, []*rules.FailureAssociationRule{
			ruleOne,
			ruleTwo,
		})

		config.SetTestProjectConfig(ctx, map[string]*configpb.ProjectConfig{
			"testproject": {
				Monorail: &configpb.MonorailProject{
					Project:          "monorailproject",
					DisplayPrefix:    "mybug.com",
					MonorailHostname: "monorailhost.com",
				},
			},
		})

		Convey("Get", func() {
			get := func(project string, ruleID string) *http.Response {
				url := fmt.Sprintf("/api/projects/%s/rules/%s", testProject, ruleID)
				request, err := http.NewRequest("GET", url, nil)
				So(err, ShouldBeNil)

				response := httptest.NewRecorder()
				router.ServeHTTP(response, request)
				return response.Result()
			}
			Convey("Exists", func() {
				response := get(ruleOne.Project, ruleOne.RuleID)

				So(response.StatusCode, ShouldEqual, 200)
				So(response.Header.Get("ETag"), ShouldEqual, ruleETag(ruleOne))

				b, err := io.ReadAll(response.Body)
				So(err, ShouldBeNil)

				var responseBody *rule
				So(json.Unmarshal(b, &responseBody), ShouldBeNil)
				So(responseBody.FailureAssociationRule, ShouldResemble, *ruleOne)
				So(responseBody.BugLink, ShouldResemble, &BugLink{
					Name: "mybug.com/151",
					URL:  "https://monorailhost.com/p/monorailproject/issues/detail?id=151",
				})
			})
			Convey("Not Exists", func() {
				ruleID := strings.Repeat("00", 16)
				response := get(ruleOne.Project, ruleID)

				So(response.StatusCode, ShouldEqual, 404)
			})
		})
		Convey("Patch", func() {
			patch := func(body *ruleUpdateRequest) *http.Response {
				b, err := json.Marshal(body)
				So(err, ShouldBeNil)

				url := fmt.Sprintf("/api/projects/%s/rules/%s", testProject, ruleOne.RuleID)
				request, err := http.NewRequest("PATCH", url, bytes.NewReader(b))
				So(err, ShouldBeNil)
				request.Header.Add("If-Match", ruleETag(ruleOne))

				response := httptest.NewRecorder()
				router.ServeHTTP(response, request)
				return response.Result()
			}

			tok, err := xsrf.Token(ctx)
			So(err, ShouldBeNil)

			request := &ruleUpdateRequest{
				Rule: &rules.FailureAssociationRule{
					RuleDefinition: `test = "updated"`,
					BugID: bugs.BugID{
						System: "monorail",
						ID:     "monorailproject/2",
					},
					IsActive: false,
				},
				UpdateMask: &fieldmaskpb.FieldMask{
					Paths: []string{"ruleDefinition", "bugId", "isActive"},
				},
				XSRFToken: tok,
			}

			Convey("Success", func() {
				response := patch(request)

				So(response.StatusCode, ShouldEqual, 200)

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

				So(response.Header.Get("ETag"), ShouldEqual, ruleETag(expectedRule))

				// Verify the returned rule matches what was expected.
				b, err := io.ReadAll(response.Body)
				So(err, ShouldBeNil)
				var responseBody *rule
				So(json.Unmarshal(b, &responseBody), ShouldBeNil)
				So(responseBody.FailureAssociationRule, ShouldResemble, *expectedRule)
				So(responseBody.BugLink, ShouldResemble, &BugLink{
					Name: "mybug.com/2",
					URL:  "https://monorailhost.com/p/monorailproject/issues/detail?id=2",
				})
			})
			Convey("Concurrent Modification", func() {
				response := patch(request)
				So(response.StatusCode, ShouldEqual, 200)

				// Attempt a new modification without
				// requerying.
				response = patch(request)
				// 409 = Status Conflict.
				So(response.StatusCode, ShouldEqual, 409)
			})
			Convey("Validation error", func() {
				Convey("Invalid bug monorail project", func() {
					request.Rule.BugID.ID = "otherproject/2"
					response := patch(request)

					So(response.StatusCode, ShouldEqual, 400)
					b, err := io.ReadAll(response.Body)
					So(err, ShouldBeNil)
					So(string(b), ShouldEqual, "Validation error: bug not in expected monorail project (monorailproject).\n")
				})
				Convey("Re-use of same bug", func() {
					// Use the same bug as another rule.
					request.Rule.BugID = ruleTwo.BugID
					response := patch(request)

					So(response.StatusCode, ShouldEqual, 400)
					b, err := io.ReadAll(response.Body)
					So(err, ShouldBeNil)
					So(string(b), ShouldContainSubstring, "Validation error: bug already used by another failure association rule")
				})
				Convey("Invalid rule definition", func() {
					// Use an invalid failure association rule.
					request.Rule.RuleDefinition = ""
					response := patch(request)

					So(response.StatusCode, ShouldEqual, 400)
					b, err := io.ReadAll(response.Body)
					So(err, ShouldBeNil)
					So(string(b), ShouldContainSubstring, "Validation error: rule definition is not valid")
				})
			})
			Convey("XSRF Token missing", func() {
				request.XSRFToken = ""
				response := patch(request)

				So(response.StatusCode, ShouldEqual, 400)
			})
		})
	})
}
