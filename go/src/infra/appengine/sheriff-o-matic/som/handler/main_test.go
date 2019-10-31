// Copyright 2016 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package handler

import (
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
	"infra/monitoring/messages"

	"golang.org/x/net/context"

	"github.com/julienschmidt/httprouter"

	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/appengine/gaetesting"
	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/clock/testclock"
	"go.chromium.org/luci/common/logging/gologger"
	"go.chromium.org/luci/server/auth/authtest"
	"go.chromium.org/luci/server/auth/xsrf"
	"go.chromium.org/luci/server/router"

	. "github.com/smartystreets/goconvey/convey"
)

var _ = fmt.Printf

func TestMain(t *testing.T) {
	t.Parallel()

	Convey("main", t, func() {
		c := gaetesting.TestingContext()
		c = authtest.MockAuthConfig(c)
		c = gologger.StdConfig.Use(c)

		cl := testclock.New(testclock.TestRecentTimeUTC)
		c = clock.Set(c, cl)

		w := httptest.NewRecorder()

		monorailMux := http.NewServeMux()
		monorailServer := httptest.NewServer(monorailMux)
		defer monorailServer.Close()
		tok, err := xsrf.Token(c)
		So(err, ShouldBeNil)
		Convey("/api/v1", func() {
			alertIdx := datastore.IndexDefinition{
				Kind:     "AlertJSON",
				Ancestor: true,
				SortBy: []datastore.IndexColumn{
					{
						Property: "Resolved",
					},
					{
						Property:   "Date",
						Descending: false,
					},
				},
			}
			revisionSummaryIdx := datastore.IndexDefinition{
				Kind:     "RevisionSummaryJSON",
				Ancestor: true,
				SortBy: []datastore.IndexColumn{
					{
						Property:   "Date",
						Descending: false,
					},
				},
			}
			indexes := []*datastore.IndexDefinition{&alertIdx, &revisionSummaryIdx}
			datastore.GetTestable(c).AddIndexes(indexes...)

			Convey("GetTrees", func() {
				Convey("no trees yet", func() {
					trees, err := GetTrees(c)

					So(err, ShouldBeNil)
					So(string(trees), ShouldEqual, "[]")
				})

				tree := &model.Tree{
					Name:        "oak",
					DisplayName: "Oak",
				}
				So(datastore.Put(c, tree), ShouldBeNil)
				datastore.GetTestable(c).CatchupIndexes()

				Convey("basic tree", func() {
					trees, err := GetTrees(c)

					So(err, ShouldBeNil)
					So(string(trees), ShouldEqual, `[{"name":"oak","display_name":"Oak","bb_project_filter":""}]`)
				})
			})

			Convey("/alerts", func() {
				contents, _ := json.Marshal(&messages.Alert{
					Key: "test",
				})
				alertJSON := &model.AlertJSON{
					ID:       "test",
					Tree:     datastore.MakeKey(c, "Tree", "chromeos"),
					Resolved: false,
					Date:     time.Unix(1, 0).UTC(),
					Contents: []byte(contents),
				}
				contents2, _ := json.Marshal(&messages.Alert{
					Key: "test2",
				})
				oldResolvedJSON := &model.AlertJSON{
					ID:       "test2",
					Tree:     datastore.MakeKey(c, "Tree", "chromeos"),
					Resolved: true,
					Date:     time.Unix(1, 0).UTC(),
					Contents: []byte(contents2),
				}
				contents3, _ := json.Marshal(&messages.Alert{
					Key: "test3",
				})
				newResolvedJSON := &model.AlertJSON{
					ID:       "test3",
					Tree:     datastore.MakeKey(c, "Tree", "chromeos"),
					Resolved: true,
					Date:     clock.Now(c),
					Contents: []byte(contents3),
				}
				oldRevisionSummaryJSON := &model.RevisionSummaryJSON{
					ID:       "rev1",
					Tree:     datastore.MakeKey(c, "Tree", "chromeos"),
					Date:     time.Unix(1, 0).UTC(),
					Contents: []byte(contents),
				}
				newRevisionSummaryJSON := &model.RevisionSummaryJSON{
					ID:       "rev2",
					Tree:     datastore.MakeKey(c, "Tree", "chromeos"),
					Date:     clock.Now(c),
					Contents: []byte(contents),
				}

				Convey("GET", func() {
					Convey("no alerts yet", func() {
						GetAlertsHandler(&router.Context{
							Context: c,
							Writer:  w,
							Request: makeGetRequest(),
							Params:  makeParams("tree", "chromeos"),
						})

						_, err := ioutil.ReadAll(w.Body)
						So(err, ShouldBeNil)
						So(w.Code, ShouldEqual, 200)
					})

					So(datastore.Put(c, alertJSON), ShouldBeNil)
					So(datastore.Put(c, oldRevisionSummaryJSON), ShouldBeNil)
					So(datastore.Put(c, newRevisionSummaryJSON), ShouldBeNil)
					datastore.GetTestable(c).CatchupIndexes()

					Convey("basic alerts", func() {
						GetAlertsHandler(&router.Context{
							Context: c,
							Writer:  w,
							Request: makeGetRequest(),
							Params:  makeParams("tree", "chromeos"),
						})

						r, err := ioutil.ReadAll(w.Body)
						So(err, ShouldBeNil)
						So(w.Code, ShouldEqual, 200)
						summary := &messages.AlertsSummary{}
						err = json.Unmarshal(r, &summary)
						So(err, ShouldBeNil)
						So(summary.Alerts, ShouldHaveLength, 1)
						So(summary.Alerts[0].Key, ShouldEqual, "test")
						So(summary.Resolved, ShouldHaveLength, 0)
						// TODO(seanmccullough): Remove all of the POST /alerts handling
						// code and tests except for whatever chromeos needs.
					})

					So(datastore.Put(c, oldResolvedJSON), ShouldBeNil)
					So(datastore.Put(c, newResolvedJSON), ShouldBeNil)

					Convey("resolved alerts", func() {
						GetAlertsHandler(&router.Context{
							Context: c,
							Writer:  w,
							Request: makeGetRequest(),
							Params:  makeParams("tree", "chromeos"),
						})

						r, err := ioutil.ReadAll(w.Body)
						So(err, ShouldBeNil)
						So(w.Code, ShouldEqual, 200)
						summary := &messages.AlertsSummary{}
						err = json.Unmarshal(r, &summary)
						So(err, ShouldBeNil)
						So(summary.Alerts, ShouldHaveLength, 1)
						So(summary.Alerts[0].Key, ShouldEqual, "test")
						So(summary.Resolved, ShouldHaveLength, 1)
						So(summary.Resolved[0].Key, ShouldEqual, "test3")
						// TODO(seanmccullough): Remove all of the POST /alerts handling
						// code and tests except for whatever chromeos needs.
					})
				})
			})

			Convey("/unresolved", func() {
				contents, _ := json.Marshal(&messages.Alert{
					Key: "test",
				})
				alertJSON := &model.AlertJSON{
					ID:       "test",
					Tree:     datastore.MakeKey(c, "Tree", "chromeos"),
					Resolved: false,
					Date:     time.Unix(1, 0).UTC(),
					Contents: []byte(contents),
				}
				contents2, _ := json.Marshal(&messages.Alert{
					Key: "test2",
				})
				oldResolvedJSON := &model.AlertJSON{
					ID:       "test2",
					Tree:     datastore.MakeKey(c, "Tree", "chromeos"),
					Resolved: true,
					Date:     time.Unix(1, 0).UTC(),
					Contents: []byte(contents2),
				}
				contents3, _ := json.Marshal(&messages.Alert{
					Key: "test3",
				})
				newResolvedJSON := &model.AlertJSON{
					ID:       "test3",
					Tree:     datastore.MakeKey(c, "Tree", "chromeos"),
					Resolved: true,
					Date:     clock.Now(c),
					Contents: []byte(contents3),
				}
				oldRevisionSummaryJSON := &model.RevisionSummaryJSON{
					ID:       "rev1",
					Tree:     datastore.MakeKey(c, "Tree", "chromeos"),
					Date:     time.Unix(1, 0).UTC(),
					Contents: []byte(contents),
				}
				newRevisionSummaryJSON := &model.RevisionSummaryJSON{
					ID:       "rev2",
					Tree:     datastore.MakeKey(c, "Tree", "chromeos"),
					Date:     clock.Now(c),
					Contents: []byte(contents),
				}

				Convey("GET", func() {
					Convey("no alerts yet", func() {
						GetUnresolvedAlertsHandler(&router.Context{
							Context: c,
							Writer:  w,
							Request: makeGetRequest(),
							Params:  makeParams("tree", "chromeos"),
						})

						_, err := ioutil.ReadAll(w.Body)
						So(err, ShouldBeNil)
						So(w.Code, ShouldEqual, 200)
					})

					So(datastore.Put(c, alertJSON), ShouldBeNil)
					So(datastore.Put(c, oldRevisionSummaryJSON), ShouldBeNil)
					So(datastore.Put(c, newRevisionSummaryJSON), ShouldBeNil)
					So(datastore.Put(c, oldResolvedJSON), ShouldBeNil)
					So(datastore.Put(c, newResolvedJSON), ShouldBeNil)
					datastore.GetTestable(c).CatchupIndexes()

					Convey("basic alerts", func() {
						GetUnresolvedAlertsHandler(&router.Context{
							Context: c,
							Writer:  w,
							Request: makeGetRequest(),
							Params:  makeParams("tree", "chromeos"),
						})

						r, err := ioutil.ReadAll(w.Body)
						So(err, ShouldBeNil)
						So(w.Code, ShouldEqual, 200)
						summary := &messages.AlertsSummary{}
						err = json.Unmarshal(r, &summary)
						So(err, ShouldBeNil)
						So(summary.Alerts, ShouldHaveLength, 1)
						So(summary.Alerts[0].Key, ShouldEqual, "test")
						So(summary.Resolved, ShouldBeNil)
					})
				})
			})

			Convey("/resolved", func() {
				contents, _ := json.Marshal(&messages.Alert{
					Key: "test",
				})
				alertJSON := &model.AlertJSON{
					ID:       "test",
					Tree:     datastore.MakeKey(c, "Tree", "chromeos"),
					Resolved: false,
					Date:     time.Unix(1, 0).UTC(),
					Contents: []byte(contents),
				}
				contents2, _ := json.Marshal(&messages.Alert{
					Key: "test2",
				})
				oldResolvedJSON := &model.AlertJSON{
					ID:       "test2",
					Tree:     datastore.MakeKey(c, "Tree", "chromeos"),
					Resolved: true,
					Date:     time.Unix(1, 0).UTC(),
					Contents: []byte(contents2),
				}
				contents3, _ := json.Marshal(&messages.Alert{
					Key: "test3",
				})
				newResolvedJSON := &model.AlertJSON{
					ID:       "test3",
					Tree:     datastore.MakeKey(c, "Tree", "chromeos"),
					Resolved: true,
					Date:     clock.Now(c),
					Contents: []byte(contents3),
				}
				oldRevisionSummaryJSON := &model.RevisionSummaryJSON{
					ID:       "rev1",
					Tree:     datastore.MakeKey(c, "Tree", "chromeos"),
					Date:     time.Unix(1, 0).UTC(),
					Contents: []byte(contents),
				}
				newRevisionSummaryJSON := &model.RevisionSummaryJSON{
					ID:       "rev2",
					Tree:     datastore.MakeKey(c, "Tree", "chromeos"),
					Date:     clock.Now(c),
					Contents: []byte(contents),
				}

				Convey("GET", func() {
					Convey("no alerts yet", func() {
						GetResolvedAlertsHandler(&router.Context{
							Context: c,
							Writer:  w,
							Request: makeGetRequest(),
							Params:  makeParams("tree", "chromeos"),
						})

						_, err := ioutil.ReadAll(w.Body)
						So(err, ShouldBeNil)
						So(w.Code, ShouldEqual, 200)
					})

					So(datastore.Put(c, alertJSON), ShouldBeNil)
					So(datastore.Put(c, oldRevisionSummaryJSON), ShouldBeNil)
					So(datastore.Put(c, newRevisionSummaryJSON), ShouldBeNil)
					So(datastore.Put(c, oldResolvedJSON), ShouldBeNil)
					So(datastore.Put(c, newResolvedJSON), ShouldBeNil)
					datastore.GetTestable(c).CatchupIndexes()

					Convey("resolved alerts", func() {
						GetResolvedAlertsHandler(&router.Context{
							Context: c,
							Writer:  w,
							Request: makeGetRequest(),
							Params:  makeParams("tree", "chromeos"),
						})

						r, err := ioutil.ReadAll(w.Body)
						So(err, ShouldBeNil)
						So(w.Code, ShouldEqual, 200)
						summary := &messages.AlertsSummary{}
						err = json.Unmarshal(r, &summary)
						So(err, ShouldBeNil)
						So(summary.Alerts, ShouldBeNil)
						So(summary.Resolved, ShouldHaveLength, 1)
						So(summary.Resolved[0].Key, ShouldEqual, "test3")
						// TODO(seanmccullough): Remove all of the POST /alerts handling
						// code and tests except for whatever chromeos needs.
					})
				})
			})
		})

		Convey("cron", func() {
			Convey("flushOldAnnotations", func() {
				getAllAnns := func() []*model.Annotation {
					anns := []*model.Annotation{}
					So(datastore.GetAll(c, datastore.NewQuery("Annotation"), &anns), ShouldBeNil)
					return anns
				}

				ann := &model.Annotation{
					KeyDigest:        fmt.Sprintf("%x", sha1.Sum([]byte("foobar"))),
					Key:              "foobar",
					ModificationTime: datastore.RoundTime(cl.Now()),
				}
				So(datastore.Put(c, ann), ShouldBeNil)
				datastore.GetTestable(c).CatchupIndexes()

				Convey("current not deleted", func() {
					num, err := flushOldAnnotations(c)
					So(err, ShouldBeNil)
					So(num, ShouldEqual, 0)
					So(getAllAnns(), ShouldResemble, []*model.Annotation{ann})
				})

				ann.ModificationTime = cl.Now().Add(-(annotationExpiration + time.Hour))
				So(datastore.Put(c, ann), ShouldBeNil)
				datastore.GetTestable(c).CatchupIndexes()

				Convey("old deleted", func() {
					num, err := flushOldAnnotations(c)
					So(err, ShouldBeNil)
					So(num, ShouldEqual, 1)
					So(getAllAnns(), ShouldResemble, []*model.Annotation{})
				})

				datastore.GetTestable(c).CatchupIndexes()
				q := datastore.NewQuery("Annotation")
				anns := []*model.Annotation{}
				datastore.GetTestable(c).CatchupIndexes()
				datastore.GetAll(c, q, &anns)
				datastore.Delete(c, anns)
				anns = []*model.Annotation{
					{
						KeyDigest:        fmt.Sprintf("%x", sha1.Sum([]byte("foobar2"))),
						Key:              "foobar2",
						ModificationTime: datastore.RoundTime(cl.Now()),
					},
					{
						KeyDigest:        fmt.Sprintf("%x", sha1.Sum([]byte("foobar"))),
						Key:              "foobar",
						ModificationTime: datastore.RoundTime(cl.Now().Add(-(annotationExpiration + time.Hour))),
					},
				}
				So(datastore.Put(c, anns), ShouldBeNil)
				datastore.GetTestable(c).CatchupIndexes()

				Convey("only delete old", func() {
					num, err := flushOldAnnotations(c)
					So(err, ShouldBeNil)
					So(num, ShouldEqual, 1)
					So(getAllAnns(), ShouldResemble, anns[:1])
				})

				Convey("handler", func() {
					ctx := &router.Context{
						Context: c,
						Writer:  w,
						Request: makePostRequest(""),
						Params:  makeParams("annKey", "foobar", "action", "add"),
					}

					FlushOldAnnotationsHandler(ctx)
				})
			})

			Convey("clientmon", func() {
				body := &eCatcherReq{XSRFToken: tok}
				bodyBytes, err := json.Marshal(body)
				So(err, ShouldBeNil)
				ctx := &router.Context{
					Context: c,
					Writer:  w,
					Request: makePostRequest(string(bodyBytes)),
					Params:  makeParams("xsrf_token", tok),
				}

				PostClientMonHandler(ctx)
				So(w.Code, ShouldEqual, 200)
			})

			Convey("treelogo", func() {
				ctx := &router.Context{
					Context: c,
					Writer:  w,
					Request: makeGetRequest(),
					Params:  makeParams("tree", "chromium"),
				}

				getTreeLogo(ctx, "", &noopSigner{})
				So(w.Code, ShouldEqual, 302)
			})

			Convey("treelogo fail", func() {
				ctx := &router.Context{
					Context: c,
					Writer:  w,
					Request: makeGetRequest(),
					Params:  makeParams("tree", "chromium"),
				}

				getTreeLogo(ctx, "", &noopSigner{fmt.Errorf("fail")})
				So(w.Code, ShouldEqual, 500)
			})
		})
	})
}

type noopSigner struct {
	err error
}

func (n *noopSigner) SignBytes(c context.Context, b []byte) (string, []byte, error) {
	return string(b), b, n.err
}

func makeGetRequest(queryParams ...string) *http.Request {
	if len(queryParams)%2 != 0 {
		return nil
	}
	params := make([]string, len(queryParams)/2)
	for i := range params {
		params[i] = fmt.Sprintf("%s=%s", queryParams[2*i], queryParams[2*i+1])
	}
	paramsStr := strings.Join(params, "&")
	url := fmt.Sprintf("/doesntmatter?%s", paramsStr)
	req, _ := http.NewRequest("GET", url, nil)
	return req
}

func makePostRequest(body string) *http.Request {
	req, _ := http.NewRequest("POST", "/doesntmatter", strings.NewReader(body))
	return req
}

func makeParams(items ...string) httprouter.Params {
	if len(items)%2 != 0 {
		return nil
	}

	params := make([]httprouter.Param, len(items)/2)
	for i := range params {
		params[i] = httprouter.Param{
			Key:   items[2*i],
			Value: items[2*i+1],
		}
	}

	return params
}
