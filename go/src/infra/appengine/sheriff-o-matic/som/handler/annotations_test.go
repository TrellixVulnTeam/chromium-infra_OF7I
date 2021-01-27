package handler

import (
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"infra/appengine/sheriff-o-matic/som/model"
	monorailv3 "infra/monorailv2/api/v3/api_proto"

	"go.chromium.org/luci/appengine/gaetesting"
	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/clock/testclock"
	"go.chromium.org/luci/common/logging/gologger"
	"go.chromium.org/luci/gae/service/datastore"
	"go.chromium.org/luci/server/auth/authtest"
	"go.chromium.org/luci/server/auth/xsrf"
	"go.chromium.org/luci/server/router"
	"google.golang.org/grpc"

	. "github.com/smartystreets/goconvey/convey"
)

func TestFilterAnnotations(t *testing.T) {
	Convey("Test filter annotation", t, func() {
		activeKeys := map[string]interface{}{
			"alert_1": nil,
			"alert_2": nil,
			"alert_3": nil,
		}

		annotations := []*model.Annotation{
			{
				Key:     "alert_1",
				GroupID: "group_1",
			},
			{
				Key:     "alert_2",
				GroupID: "group_2",
			},
			{
				Key:     "group_2",
				GroupID: "",
			},
			{
				Key:     "group_3",
				GroupID: "",
			},
			{
				Key:     "group_1",
				GroupID: "",
			},
		}
		result := filterAnnotations(annotations, activeKeys)
		So(len(result), ShouldEqual, 4)
		So(result[0].Key, ShouldEqual, "alert_1")
		So(result[1].Key, ShouldEqual, "alert_2")
		So(result[2].Key, ShouldEqual, "group_2")
		So(result[3].Key, ShouldEqual, "group_1")
	})
}

func TestFilterDuplicateBugs(t *testing.T) {
	Convey("Test filter annotation", t, func() {
		bugs := []model.MonorailBug{
			{
				BugID:     "bug_1",
				ProjectID: "project_1",
			},
			{
				BugID:     "bug_2",
				ProjectID: "project_2",
			},
			{
				BugID:     "bug_1",
				ProjectID: "project_1",
			},
			{
				BugID:     "bug_3",
				ProjectID: "project_3",
			},
		}

		result := filterDuplicateBugs(bugs)
		So(len(result), ShouldEqual, 3)
		So(result[0].BugID, ShouldEqual, "bug_1")
		So(result[1].BugID, ShouldEqual, "bug_2")
		So(result[2].BugID, ShouldEqual, "bug_3")
	})
}

func TestCreateProjectChunksMapping(t *testing.T) {
	Convey("Test create project chunk mapping", t, func() {
		bugs := []model.MonorailBug{
			{
				BugID:     "bug_1",
				ProjectID: "project_1",
			},
			{
				BugID:     "bug_2",
				ProjectID: "project_2",
			},
			{
				BugID:     "bug_3",
				ProjectID: "project_1",
			},
			{
				BugID:     "bug_4",
				ProjectID: "project_3",
			},
			{
				BugID:     "bug_5",
				ProjectID: "project_1",
			},
		}

		result := createProjectChunksMapping(bugs, 100)
		So(
			result,
			ShouldResemble,
			map[string][][]string{
				"project_1": {{"bug_1", "bug_3", "bug_5"}},
				"project_2": {{"bug_2"}},
				"project_3": {{"bug_4"}},
			},
		)

		result = createProjectChunksMapping(bugs, 2)
		So(
			result,
			ShouldResemble,
			map[string][][]string{
				"project_1": {{"bug_1", "bug_3"}, {"bug_5"}},
				"project_2": {{"bug_2"}},
				"project_3": {{"bug_4"}},
			},
		)
	})
}

func TestBreakToChunk(t *testing.T) {
	Convey("Test break bug ids to chunk", t, func() {
		bugIDs := []string{"bug1", "bug2", "bug3", "bug4", "bug5"}
		chunks := breakToChunks(bugIDs, 1)
		So(chunks, ShouldResemble, [][]string{{"bug1"}, {"bug2"}, {"bug3"}, {"bug4"}, {"bug5"}})
		chunks = breakToChunks(bugIDs, 3)
		So(chunks, ShouldResemble, [][]string{{"bug1", "bug2", "bug3"}, {"bug4", "bug5"}})
		chunks = breakToChunks(bugIDs, 5)
		So(chunks, ShouldResemble, [][]string{{"bug1", "bug2", "bug3", "bug4", "bug5"}})
		chunks = breakToChunks(bugIDs, 6)
		So(chunks, ShouldResemble, [][]string{{"bug1", "bug2", "bug3", "bug4", "bug5"}})
	})
}

func TestMakeAnnotationResponse(t *testing.T) {
	Convey("Test make annotation response successful", t, func() {
		annotations := &model.Annotation{
			Bugs: []model.MonorailBug{
				{BugID: "123", ProjectID: "chromium"},
				{BugID: "456", ProjectID: "chromium"},
			},
		}
		meta := map[string]*monorailv3.Issue{
			"123": {
				Name: "projects/chromium/issues/123",
				Status: &monorailv3.Issue_StatusValue{
					Status: "Assigned",
				},
				Summary: "Sum1",
			},
			"456": {
				Name: "projects/chromium/issues/456",
				Status: &monorailv3.Issue_StatusValue{
					Status: "Fixed",
				},
				Summary: "Sum2",
			},
		}
		expected := &AnnotationResponse{
			Annotation: *annotations,
			BugData: map[string]MonorailBugData{
				"123": {
					BugID:     "123",
					ProjectID: "chromium",
					Summary:   "Sum1",
					Status:    "Assigned",
				},
				"456": {
					BugID:     "456",
					ProjectID: "chromium",
					Summary:   "Sum2",
					Status:    "Fixed",
				},
			},
		}
		actual, err := makeAnnotationResponse(annotations, meta)
		So(err, ShouldBeNil)
		So(actual, ShouldResemble, expected)
	})
	Convey("Test make annotation response failed", t, func() {
		annotations := &model.Annotation{
			Bugs: []model.MonorailBug{
				{BugID: "123", ProjectID: "chromium"},
			},
		}
		meta := map[string]*monorailv3.Issue{
			"123": {
				Name: "invalid",
			},
		}
		_, err := makeAnnotationResponse(annotations, meta)
		So(err, ShouldNotBeNil)
	})
}

type FakeIC struct{}

func (ic FakeIC) SearchIssues(c context.Context, req *monorailv3.SearchIssuesRequest, ops ...grpc.CallOption) (*monorailv3.SearchIssuesResponse, error) {
	if req.Projects[0] == "projects/chromium" {
		return &monorailv3.SearchIssuesResponse{
			Issues: []*monorailv3.Issue{
				{
					Name: "projects/chromium/issues/333",
					Status: &monorailv3.Issue_StatusValue{
						Status: "Untriaged",
					},
				},
				{
					Name: "projects/chromium/issues/444",
					Status: &monorailv3.Issue_StatusValue{
						Status: "Untriaged",
					},
				},
			},
		}, nil
	}
	if req.Projects[0] == "projects/fuchsia" {
		return &monorailv3.SearchIssuesResponse{
			Issues: []*monorailv3.Issue{
				{
					Name: "projects/fuchsia/issues/555",
					Status: &monorailv3.Issue_StatusValue{
						Status: "Untriaged",
					},
				},
				{
					Name: "projects/fuchsia/issues/666",
					Status: &monorailv3.Issue_StatusValue{
						Status: "Untriaged",
					},
				},
			},
		}, nil
	}
	return nil, nil
}

func (ic FakeIC) MakeIssue(c context.Context, req *monorailv3.MakeIssueRequest, opts ...grpc.CallOption) (*monorailv3.Issue, error) {
	projectRes := req.Parent
	return &monorailv3.Issue{
		Name:    fmt.Sprintf("%s/issues/123", projectRes),
		Summary: req.Issue.Summary,
		Status:  req.Issue.Status,
		Labels:  req.Issue.Labels,
		CcUsers: req.Issue.CcUsers,
	}, nil
}

func TestAnnotations(t *testing.T) {
	newContext := func() (context.Context, testclock.TestClock) {
		c := gaetesting.TestingContext()
		c = authtest.MockAuthConfig(c)
		c = gologger.StdConfig.Use(c)

		cl := testclock.New(testclock.TestRecentTimeUTC)
		c = clock.Set(c, cl)
		return c, cl
	}
	Convey("/annotations", t, func() {

		w := httptest.NewRecorder()
		c, cl := newContext()
		tok, err := xsrf.Token(c)
		So(err, ShouldBeNil)

		ah := &AnnotationHandler{
			Bqh:                 &BugQueueHandler{},
			MonorailIssueClient: FakeIC{},
		}

		Convey("GET", func() {
			Convey("no annotations yet", func() {
				ah.GetAnnotationsHandler(&router.Context{
					Context: c,
					Writer:  w,
					Request: makeGetRequest(),
				}, nil)

				r, err := ioutil.ReadAll(w.Body)
				So(err, ShouldBeNil)
				body := string(r)
				So(w.Code, ShouldEqual, 200)
				So(body, ShouldEqual, "[]")
			})

			ann := &model.Annotation{
				KeyDigest:        fmt.Sprintf("%x", sha1.Sum([]byte("foobar"))),
				Key:              "foobar",
				Bugs:             []model.MonorailBug{{BugID: "111", ProjectID: "fuchsia"}, {BugID: "222", ProjectID: "chromium"}},
				SnoozeTime:       123123,
				ModificationTime: datastore.RoundTime(clock.Now(c).Add(4 * time.Hour)),
			}

			So(datastorePutAnnotation(c, ann), ShouldBeNil)
			datastore.GetTestable(c).CatchupIndexes()

			Convey("basic annotation", func() {
				ah.GetAnnotationsHandler(&router.Context{
					Context: c,
					Writer:  w,
					Request: makeGetRequest(),
				}, map[string]interface{}{ann.Key: nil})

				r, err := ioutil.ReadAll(w.Body)
				So(err, ShouldBeNil)
				body := string(r)
				So(w.Code, ShouldEqual, 200)
				rslt := []*model.Annotation{}
				So(json.NewDecoder(strings.NewReader(body)).Decode(&rslt), ShouldBeNil)
				So(rslt, ShouldHaveLength, 1)
				So(rslt[0], ShouldResemble, ann)
			})

			Convey("basic annotation, alert no longer active", func() {
				ah.GetAnnotationsHandler(&router.Context{
					Context: c,
					Writer:  w,
					Request: makeGetRequest(),
				}, nil)

				r, err := ioutil.ReadAll(w.Body)
				So(err, ShouldBeNil)
				body := string(r)
				So(w.Code, ShouldEqual, 200)
				rslt := []*model.Annotation{}
				So(json.NewDecoder(strings.NewReader(body)).Decode(&rslt), ShouldBeNil)
				So(rslt, ShouldHaveLength, 0)
			})
		})

		addXSRFToken := func(data map[string]interface{}, tok string) string {
			change, err := json.Marshal(map[string]interface{}{
				"xsrf_token": tok,
				"data":       data,
			})
			So(err, ShouldBeNil)
			return string(change)
		}

		Convey("POST", func() {
			Convey("invalid action", func() {
				ah.PostAnnotationsHandler(&router.Context{
					Context: c,
					Writer:  w,
					Request: makePostRequest(""),
					Params:  makeParams("action", "lolwut"),
				})

				So(w.Code, ShouldEqual, 400)
			})

			Convey("invalid json", func() {
				ah.PostAnnotationsHandler(&router.Context{
					Context: c,
					Writer:  w,
					Request: makePostRequest("invalid json"),
					Params:  makeParams("annKey", "foobar", "action", "add"),
				})

				So(w.Code, ShouldEqual, http.StatusBadRequest)
			})

			ann := &model.Annotation{
				Tree:             datastore.MakeKey(c, "Tree", "tree.unknown"),
				Key:              "foobar",
				KeyDigest:        fmt.Sprintf("%x", sha1.Sum([]byte("foobar"))),
				ModificationTime: datastore.RoundTime(clock.Now(c)),
			}
			cl.Add(time.Hour)

			Convey("add, bad xsrf token", func() {
				ah.PostAnnotationsHandler(&router.Context{
					Context: c,
					Writer:  w,
					Request: makePostRequest(addXSRFToken(map[string]interface{}{
						"snoozeTime": 123123,
					}, "no good token")),
					Params: makeParams("annKey", "foobar", "action", "add"),
				})

				So(w.Code, ShouldEqual, http.StatusForbidden)
			})

			Convey("add", func() {
				ann = &model.Annotation{
					Tree:             datastore.MakeKey(c, "Tree", "tree.unknown"),
					Key:              "foobar",
					KeyDigest:        fmt.Sprintf("%x", sha1.Sum([]byte("foobar"))),
					ModificationTime: datastore.RoundTime(clock.Now(c)),
				}
				change := map[string]interface{}{}
				Convey("snoozeTime", func() {
					ah.PostAnnotationsHandler(&router.Context{
						Context: c,
						Writer:  w,
						Request: makePostRequest(addXSRFToken(map[string]interface{}{
							"snoozeTime": 123123,
							"key":        "foobar",
						}, tok)),
						Params: makeParams("action", "add", "tree", "tree.unknown"),
					})

					So(w.Code, ShouldEqual, 200)
					So(datastoreGetAnnotation(c, ann), ShouldBeNil)
					So(ann.SnoozeTime, ShouldEqual, 123123)
				})

				Convey("bugs", func() {
					change["bugs"] = []model.MonorailBug{{BugID: "123123", ProjectID: "chromium"}}
					change["key"] = "foobar"
					ah.PostAnnotationsHandler(&router.Context{
						Context: c,
						Writer:  w,
						Request: makePostRequest(addXSRFToken(change, tok)),
						Params:  makeParams("action", "add", "tree", "tree.unknown"),
					})

					So(w.Code, ShouldEqual, 200)

					So(datastoreGetAnnotation(c, ann), ShouldBeNil)
					So(ann.Bugs, ShouldResemble, []model.MonorailBug{{BugID: "123123", ProjectID: "chromium"}})
				})
			})

			Convey("remove", func() {
				Convey("can't remove non-existent annotation", func() {
					ah.PostAnnotationsHandler(&router.Context{
						Context: c,
						Writer:  w,
						Request: makePostRequest(addXSRFToken(map[string]interface{}{"key": "foobar"}, tok)),
						Params:  makeParams("action", "remove", "tree", "tree.unknown"),
					})

					So(w.Code, ShouldEqual, 404)
				})

				ann.SnoozeTime = 123
				So(datastorePutAnnotation(c, ann), ShouldBeNil)

				Convey("basic", func() {
					So(ann.SnoozeTime, ShouldEqual, 123)

					ah.PostAnnotationsHandler(&router.Context{
						Context: c,
						Writer:  w,
						Request: makePostRequest(addXSRFToken(map[string]interface{}{
							"key":        "foobar",
							"snoozeTime": true,
						}, tok)),
						Params: makeParams("action", "remove", "tree", "tree.unknown"),
					})

					So(w.Code, ShouldEqual, 200)
					So(datastoreGetAnnotation(c, ann), ShouldBeNil)
					So(ann.SnoozeTime, ShouldEqual, 0)
				})
			})
		})

		Convey("refreshAnnotations", func() {
			Convey("handler", func() {
				c, _ := newContext()

				ah.RefreshAnnotationsHandler(&router.Context{
					Context: c,
					Writer:  w,
					Request: makeGetRequest(),
				})

				b, err := ioutil.ReadAll(w.Body)
				So(err, ShouldBeNil)
				So(w.Code, ShouldEqual, 200)
				So(string(b), ShouldEqual, "{}")
			})

			ann := &model.Annotation{
				KeyDigest: fmt.Sprintf("%x", sha1.Sum([]byte("foobar"))),
				Key:       "foobar",
				Bugs:      []model.MonorailBug{{BugID: "333", ProjectID: "chromium"}, {BugID: "444", ProjectID: "chromium"}},
			}

			ann1 := &model.Annotation{
				KeyDigest: fmt.Sprintf("%x", sha1.Sum([]byte("foobar1"))),
				Key:       "foobar1",
				Bugs:      []model.MonorailBug{{BugID: "555", ProjectID: "fuchsia"}, {BugID: "666", ProjectID: "fuchsia"}},
			}

			So(datastorePutAnnotation(c, ann), ShouldBeNil)
			So(datastorePutAnnotation(c, ann1), ShouldBeNil)
			datastore.GetTestable(c).CatchupIndexes()

			Convey("query alerts which have multiple bugs", func() {
				ah.RefreshAnnotationsHandler(&router.Context{
					Context: c,
					Writer:  w,
					Request: makeGetRequest(),
				})

				b, err := ioutil.ReadAll(w.Body)
				So(err, ShouldBeNil)
				So(w.Code, ShouldEqual, 200)

				var result map[string]interface{}
				json.Unmarshal(b, &result)
				expected := map[string]interface{}{
					"333": map[string]interface{}{
						"name":   "projects/chromium/issues/333",
						"status": map[string]interface{}{"status": "Untriaged"},
					},
					"444": map[string]interface{}{
						"name":   "projects/chromium/issues/444",
						"status": map[string]interface{}{"status": "Untriaged"},
					},
					"555": map[string]interface{}{
						"name":   "projects/fuchsia/issues/555",
						"status": map[string]interface{}{"status": "Untriaged"},
					},
					"666": map[string]interface{}{
						"name":   "projects/fuchsia/issues/666",
						"status": map[string]interface{}{"status": "Untriaged"},
					},
				}
				So(result, ShouldResemble, expected)
			})
		})
		Convey("file bug", func() {
			c, _ := newContext()
			ah.FileBugHandler(&router.Context{
				Context: c,
				Writer:  w,
				Request: makePostRequest(addXSRFToken(map[string]interface{}{
					"ProjectId":   "chromium",
					"Description": "des",
					"Labels":      []string{"label1"},
					"Cc":          []string{"abc@google.com"},
					"Summary":     "my summary",
				}, tok)),
			})

			b, err := ioutil.ReadAll(w.Body)
			So(err, ShouldBeNil)
			So(w.Code, ShouldEqual, 200)
			res := &monorailv3.Issue{}
			err = json.Unmarshal(b, res)
			So(res, ShouldResemble, &monorailv3.Issue{
				Name:    "projects/chromium/issues/123",
				Summary: "my summary",
				Status:  &monorailv3.Issue_StatusValue{Status: "Untriaged"},
				CcUsers: []*monorailv3.Issue_UserValue{
					{User: "users/abc@google.com"},
				},
				Labels: []*monorailv3.Issue_LabelValue{
					{Label: "label1"},
				},
			})
		})
	})
	Convey("Test process issue request", t, func() {
		c, _ := newContext()
		Convey("Test process issue request chromium", func() {
			req := &monorailv3.MakeIssueRequest{
				Parent: "projects/chromium",
				Issue: &monorailv3.Issue{
					Summary: "sum",
					Status: &monorailv3.Issue_StatusValue{
						Status: "Untriaged",
					},
				},
			}
			processIssueRequest(c, "chromium", req)
			expected := &monorailv3.MakeIssueRequest{
				Parent: "projects/chromium",
				Issue: &monorailv3.Issue{
					Summary: "sum",
					Status: &monorailv3.Issue_StatusValue{
						Status: "Untriaged",
					},
					FieldValues: []*monorailv3.FieldValue{
						{
							Field: "projects/chromium/fieldDefs/10",
							Value: "Bug",
						},
					},
				},
			}
			So(req, ShouldResemble, expected)
		})
		Convey("Test process issue request angleproject", func() {
			req := &monorailv3.MakeIssueRequest{
				Parent: "projects/angleproject",
				Issue: &monorailv3.Issue{
					Summary: "sum",
					Status: &monorailv3.Issue_StatusValue{
						Status: "Untriaged",
					},
					Labels: []*monorailv3.Issue_LabelValue{
						{Label: "File-From-SoM"},
						{Label: "Pri-3"},
					},
				},
			}
			processIssueRequest(c, "angleproject", req)
			expected := &monorailv3.MakeIssueRequest{
				Parent: "projects/angleproject",
				Issue: &monorailv3.Issue{
					Summary: "sum",
					Status: &monorailv3.Issue_StatusValue{
						Status: "New",
					},
					FieldValues: []*monorailv3.FieldValue{
						{
							Field: "projects/angleproject/fieldDefs/32",
							Value: "Low",
						},
						{
							Field: "projects/angleproject/fieldDefs/55",
							Value: "Defect",
						},
					},
					Labels: []*monorailv3.Issue_LabelValue{
						{Label: "File-From-SoM"},
					},
				},
			}
			So(req, ShouldResemble, expected)
		})
	})
}
