// Copyright 2017 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"testing"
	"time"

	"github.com/golang/protobuf/jsonpb"
	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/luci/common/data/stringset"
	ds "go.chromium.org/luci/gae/service/datastore"

	"infra/qscheduler/qslib/tutils"
	admin "infra/tricium/api/admin/v1"
	tricium "infra/tricium/api/v1"
	"infra/tricium/appengine/common/gerrit"
	"infra/tricium/appengine/common/track"
	"infra/tricium/appengine/common/triciumtest"

	. "go.chromium.org/luci/common/testing/assertions"
)

func TestWorkerDoneRequest(t *testing.T) {
	Convey("Worker done request with successful worker", t, func() {
		ctx := triciumtest.Context()

		simple := "Simple"
		simpleUbuntu := "Simple_Ubuntu"

		workflowProvider := &mockWorkflowProvider{
			Workflow: &admin.Workflow{
				Workers: []*admin.Worker{
					{
						Name:     simpleUbuntu,
						Needs:    tricium.Data_FILES,
						Provides: tricium.Data_RESULTS,
					},
				},
			},
		}

		// Add pending workflow run.
		request := &track.AnalyzeRequest{}
		request.GitRef = "refs/changes/88/508788/7"
		So(ds.Put(ctx, request), ShouldBeNil)
		requestKey := ds.KeyForObj(ctx, request)
		run := &track.WorkflowRun{ID: 1, Parent: requestKey}
		So(ds.Put(ctx, run), ShouldBeNil)
		runKey := ds.KeyForObj(ctx, run)
		So(ds.Put(ctx, &track.WorkflowRunResult{
			ID:     1,
			Parent: runKey,
			State:  tricium.State_PENDING,
		}), ShouldBeNil)

		// Mark workflow as launched.
		So(workflowLaunched(ctx, &admin.WorkflowLaunchedRequest{
			RunId: request.ID,
		}, workflowProvider), ShouldBeNil)

		// Mark worker as launched.
		So(workerLaunched(ctx, &admin.WorkerLaunchedRequest{
			RunId:  request.ID,
			Worker: simpleUbuntu,
		}), ShouldBeNil)

		// Mark worker as done.
		So(workerDone(ctx, &admin.WorkerDoneRequest{
			RunId:             request.ID,
			Worker:            simpleUbuntu,
			Provides:          tricium.Data_RESULTS,
			State:             tricium.State_SUCCESS,
			BuildbucketOutput: `{"comments": [{"message": "foo"}, {"message": "bar"}]}`,
		}), ShouldBeNil)

		functionKey := ds.NewKey(ctx, "FunctionRun", simple, 0, runKey)

		Convey("Marks worker as done", func() {
			workerKey := ds.NewKey(ctx, "WorkerRun", simpleUbuntu, 0, functionKey)
			wr := &track.WorkerRunResult{ID: 1, Parent: workerKey}
			So(ds.Get(ctx, wr), ShouldBeNil)
			So(wr.State, ShouldEqual, tricium.State_SUCCESS)
		})

		Convey("Marks function as done and adds no comments", func() {
			fr := &track.FunctionRunResult{ID: 1, Parent: functionKey}
			So(ds.Get(ctx, fr), ShouldBeNil)
			So(fr.State, ShouldEqual, tricium.State_SUCCESS)
		})

		Convey("Marks workflow as done and adds comments", func() {
			wr := &track.WorkflowRunResult{ID: 1, Parent: runKey}
			So(ds.Get(ctx, wr), ShouldBeNil)
			So(wr.State, ShouldEqual, tricium.State_SUCCESS)
			So(wr.NumComments, ShouldEqual, 2)
		})

		Convey("Marks request as done", func() {
			ar := &track.AnalyzeRequestResult{ID: 1, Parent: requestKey}
			So(ds.Get(ctx, ar), ShouldBeNil)
			So(ar.State, ShouldEqual, tricium.State_SUCCESS)
		})
	})
}

func TestRecipeWorkerDoneRequest(t *testing.T) {
	Convey("Worker done request with successful worker", t, func() {
		ctx := triciumtest.Context()

		simple := "Simple"
		simpleUbuntu := "Simple_Ubuntu"

		workflowProvider := &mockWorkflowProvider{
			Workflow: &admin.Workflow{
				Workers: []*admin.Worker{
					{
						Name:     simpleUbuntu,
						Provides: tricium.Data_RESULTS,
					},
				},
			},
		}

		// Add pending workflow run.
		request := &track.AnalyzeRequest{}
		request.GitRef = "refs/changes/88/508788/7"
		So(ds.Put(ctx, request), ShouldBeNil)
		requestKey := ds.KeyForObj(ctx, request)
		run := &track.WorkflowRun{ID: 1, Parent: requestKey}
		So(ds.Put(ctx, run), ShouldBeNil)
		runKey := ds.KeyForObj(ctx, run)
		So(ds.Put(ctx, &track.WorkflowRunResult{
			ID:     1,
			Parent: runKey,
			State:  tricium.State_PENDING,
		}), ShouldBeNil)

		// Mark workflow as launched.
		So(workflowLaunched(ctx, &admin.WorkflowLaunchedRequest{
			RunId: request.ID,
		}, workflowProvider), ShouldBeNil)

		// Mark worker as launched.
		So(workerLaunched(ctx, &admin.WorkerLaunchedRequest{
			RunId:  request.ID,
			Worker: simpleUbuntu,
		}), ShouldBeNil)

		// Mark worker as done.
		So(workerDone(ctx, &admin.WorkerDoneRequest{
			RunId:             request.ID,
			Worker:            simpleUbuntu,
			Provides:          tricium.Data_GIT_FILE_DETAILS,
			State:             tricium.State_SUCCESS,
			BuildbucketOutput: `{"comments": []}`,
		}), ShouldBeNil)

		functionKey := ds.NewKey(ctx, "FunctionRun", simple, 0, runKey)

		Convey("Marks worker as done", func() {
			workerKey := ds.NewKey(ctx, "WorkerRun", simpleUbuntu, 0, functionKey)
			wr := &track.WorkerRunResult{ID: 1, Parent: workerKey}
			So(ds.Get(ctx, wr), ShouldBeNil)
			So(wr.State, ShouldEqual, tricium.State_SUCCESS)
		})
	})
}

func TestAbortedWorkerDoneRequest(t *testing.T) {
	Convey("Worker done request with an aborted worker", t, func() {
		// This test is similar to the case above, except that one of
		// the workers is aborted, so the function is considered
		// failed, and thus the workflow run is failed.
		ctx := triciumtest.Context()

		simple := "Simple"
		simpleUbuntu := "Simple_Ubuntu"

		workflowProvider := &mockWorkflowProvider{
			Workflow: &admin.Workflow{
				Workers: []*admin.Worker{
					{
						Name:     simpleUbuntu,
						Needs:    tricium.Data_GIT_FILE_DETAILS,
						Provides: tricium.Data_RESULTS,
					},
				},
			},
		}

		// Add pending run entry.
		request := &track.AnalyzeRequest{}
		request.GitRef = "refs/changes/88/508788/7"
		So(ds.Put(ctx, request), ShouldBeNil)
		requestKey := ds.KeyForObj(ctx, request)
		run := &track.WorkflowRun{ID: 1, Parent: requestKey}
		So(ds.Put(ctx, run), ShouldBeNil)
		runKey := ds.KeyForObj(ctx, run)
		So(ds.Put(ctx, &track.WorkflowRunResult{
			ID:     1,
			Parent: runKey,
			State:  tricium.State_PENDING,
		}), ShouldBeNil)

		// Mark workflow as launched.
		So(workflowLaunched(ctx, &admin.WorkflowLaunchedRequest{
			RunId: request.ID,
		}, workflowProvider), ShouldBeNil)

		// Mark worker as launched.
		So(workerLaunched(ctx, &admin.WorkerLaunchedRequest{
			RunId:  request.ID,
			Worker: simpleUbuntu,
		}), ShouldBeNil)

		// Mark worker as aborted.
		So(workerDone(ctx, &admin.WorkerDoneRequest{
			RunId:             request.ID,
			Worker:            simpleUbuntu,
			State:             tricium.State_ABORTED,
			BuildbucketOutput: `{"comments": []}`,
		}), ShouldBeNil)

		functionKey := ds.NewKey(ctx, "FunctionRun", simple, 0, runKey)

		Convey("WorkerRun is marked as aborted", func() {
			workerKey := ds.NewKey(ctx, "WorkerRun", simpleUbuntu, 0, functionKey)
			wr := &track.WorkerRunResult{ID: 1, Parent: workerKey}
			So(ds.Get(ctx, wr), ShouldBeNil)
			So(wr.State, ShouldEqual, tricium.State_ABORTED)
		})

		Convey("FunctionRun is failed, with no comments", func() {
			fr := &track.FunctionRunResult{ID: 1, Parent: functionKey}
			So(ds.Get(ctx, fr), ShouldBeNil)
			So(fr.State, ShouldEqual, tricium.State_FAILURE)
		})

		Convey("WorkflowRun is marked as failed", func() {
			wr := &track.WorkflowRunResult{ID: 1, Parent: runKey}
			So(ds.Get(ctx, wr), ShouldBeNil)
			So(wr.State, ShouldEqual, tricium.State_FAILURE)
		})

		Convey("AnalyzeRequest is marked as failed", func() {
			ar := &track.AnalyzeRequestResult{ID: 1, Parent: requestKey}
			So(ds.Get(ctx, ar), ShouldBeNil)
			So(ar.State, ShouldEqual, tricium.State_FAILURE)
		})
	})
}

func TestValidateWorkerDoneRequestRequest(t *testing.T) {
	Convey("Request with all parts is valid", t, func() {
		So(validateWorkerDoneRequest(&admin.WorkerDoneRequest{
			RunId:             1234,
			Worker:            "MyLint_Ubuntu",
			Provides:          tricium.Data_RESULTS,
			State:             tricium.State_SUCCESS,
			BuildbucketOutput: `{"comments": []}`,
		}), ShouldBeNil)
	})

	Convey("Specifying provides and state is optional", t, func() {
		So(validateWorkerDoneRequest(&admin.WorkerDoneRequest{
			RunId:             1234,
			Worker:            "MyLint_Ubuntu",
			BuildbucketOutput: `{"comments": []}`,
		}), ShouldBeNil)
	})

	Convey("Request with no run ID is invalid", t, func() {
		So(validateWorkerDoneRequest(&admin.WorkerDoneRequest{
			Worker:            "MyLint_Ubuntu",
			BuildbucketOutput: `{"comments": []}`,
		}), ShouldNotBeNil)
	})

	Convey("Request with no worker name invalid", t, func() {
		So(validateWorkerDoneRequest(&admin.WorkerDoneRequest{
			RunId:             1234,
			BuildbucketOutput: `{"comments": []}`,
		}), ShouldNotBeNil)
	})

	Convey("Request with no output is valid", t, func() {
		So(validateWorkerDoneRequest(&admin.WorkerDoneRequest{
			RunId:  1234,
			Worker: "MyLint_Ubuntu",
		}), ShouldBeNil)
	})

	Convey("Providing buildbucket output and not isolated output is OK", t, func() {
		So(validateWorkerDoneRequest(&admin.WorkerDoneRequest{
			RunId:             1234,
			Worker:            "MyLint_Ubuntu",
			BuildbucketOutput: `{"comments": []}`,
		}), ShouldBeNil)
	})

	Convey("Providing deprecated field isolated output is not OK", t, func() {
		So(validateWorkerDoneRequest(&admin.WorkerDoneRequest{
			RunId:              1234,
			Worker:             "MyLint_Ubuntu",
			IsolatedOutputHash: "12ab34cd",
		}), ShouldNotBeNil)
	})
}

func TestCreateCommentSelection(t *testing.T) {
	changedLines := gerrit.ChangedLinesInfo{
		"dir/file.txt": {2, 5, 6},
	}
	mock := &gerrit.MockRestAPI{ChangedLines: changedLines}
	ctx := triciumtest.Context()
	req := track.AnalyzeRequest{
		GerritProject: "my-project",
		GerritChange:  "my-project~master~I8473b95934b5732ac55d26311a706c9c2bde9940",
	}
	Convey("createCommentSelection keeps unchanged lines if comment's ShowOnUnchangedLines is true", t, func() {
		commentJSON, err := (&jsonpb.Marshaler{}).MarshalToString(&tricium.Data_Comment{
			Message:              "Not in Change",
			Path:                 "dir/file.txt",
			StartLine:            10,
			EndLine:              11,
			ShowOnUnchangedLines: true,
		})
		comments := []*track.Comment{
			{
				Comment: []byte(commentJSON),
			},
		}
		selections := []*track.CommentSelection{
			{
				ID:       1,
				Parent:   nil,
				Included: true,
			},
		}
		result, err := createCommentSelections(ctx, mock, &req, comments)
		So(result, ShouldResemble, selections)
		So(err, ShouldBeNil)
	})

	Convey("createCommentSelection discards unchanged lines if comment's ShowOnUnchangedLines is false", t, func() {
		commentJSON, err := (&jsonpb.Marshaler{}).MarshalToString(&tricium.Data_Comment{
			Message:              "Not in Change",
			Path:                 "dir/file.txt",
			StartLine:            10,
			EndLine:              11,
			ShowOnUnchangedLines: false,
		})
		comments := []*track.Comment{
			{
				Comment: []byte(commentJSON),
			},
		}
		selections := []*track.CommentSelection{
			{
				ID:       1,
				Parent:   nil,
				Included: false,
			},
		}
		result, err := createCommentSelections(ctx, mock, &req, comments)
		So(result, ShouldResemble, selections)
		So(err, ShouldBeNil)
	})
}

func TestCreateAnalysisResults(t *testing.T) {
	Convey("Default objects", t, func() {
		wres := track.WorkerRunResult{}
		areq := track.AnalyzeRequest{}
		ares := track.AnalyzeRequestResult{}
		comments := []*track.Comment{}
		selections := []*track.CommentSelection{}

		areq.GitRef = "refs/changes/88/508788/102"
		result, err := createAnalysisResults(&wres, &areq, &ares, comments, selections)
		So(err, ShouldBeNil)
		So(result, ShouldNotBeNil)
		So(result.RevisionNumber, ShouldEqual, 102)
	})

	Convey("GitRef required", t, func() {
		wres := track.WorkerRunResult{}
		areq := track.AnalyzeRequest{}
		ares := track.AnalyzeRequestResult{}
		comments := []*track.Comment{}
		selections := []*track.CommentSelection{}

		result, err := createAnalysisResults(&wres, &areq, &ares, comments, selections)
		So(err, ShouldNotBeNil)
		So(result, ShouldBeNil)
	})

	Convey("All values", t, func() {
		wres := track.WorkerRunResult{}

		areq := track.AnalyzeRequest{}
		areq.GerritHost = "http://my-gerrit-review.com/my-project"
		areq.Project = "my-project"
		areq.GerritChange = "my-project~master~I8473b95934b5732ac55d26311a706c9c2bde9940"
		areq.GitURL = "http://the-git-url.com/my-project"
		areq.GitRef = "refs/changes/88/508788/7"
		areq.Received = time.Now()
		areq.Files = []tricium.Data_File{
			{Path: "dir/file.txt"},
			{Path: "dir/file2.txt"},
		}

		ares := track.AnalyzeRequestResult{}
		deletedFileCommentJSON, err := (&jsonpb.Marshaler{}).MarshalToString(&tricium.Data_Comment{
			Category: "L",
			Message:  "Line too long",
			Path:     "dir/deleted_file.txt",
		})
		inChangeCommentJSON, err := (&jsonpb.Marshaler{}).MarshalToString(&tricium.Data_Comment{
			Category:  "L",
			Message:   "Line too short",
			Path:      "dir/file.txt",
			StartLine: 2,
			EndLine:   3,
		})
		So(err, ShouldBeNil)
		comments := []*track.Comment{
			{
				UUID:         "1234",
				Parent:       nil,
				Platforms:    tricium.PlatformBitPosToMask(tricium.Platform_ANY),
				Analyzer:     "analyzerName",
				Category:     "analyzerName/categoryName",
				Comment:      []byte(deletedFileCommentJSON),
				CreationTime: time.Now(),
			},
			{
				UUID:         "1234",
				Parent:       nil,
				Platforms:    tricium.PlatformBitPosToMask(tricium.Platform_IOS) | tricium.PlatformBitPosToMask(tricium.Platform_WINDOWS),
				Analyzer:     "analyzerName",
				Category:     "analyzerName/categoryName",
				Comment:      []byte(inChangeCommentJSON),
				CreationTime: time.Now(),
			},
			{
				UUID:         "1234",
				Parent:       nil,
				Platforms:    tricium.PlatformBitPosToMask(tricium.Platform_OSX),
				Analyzer:     "notSelected",
				Category:     "notSelected/categoryName",
				Comment:      []byte(inChangeCommentJSON),
				CreationTime: time.Now(),
			},
		}
		selections := []*track.CommentSelection{
			{
				ID:       1,
				Parent:   nil,
				Included: true,
			},
			{
				ID:       1,
				Parent:   nil,
				Included: true,
			},
			{
				ID:       1,
				Parent:   nil,
				Included: false,
			},
		}

		result, err := createAnalysisResults(&wres, &areq, &ares, comments, selections)
		So(err, ShouldBeNil)
		So(result, ShouldNotBeNil)
		So(result.GerritRevision.Host, ShouldEqual, areq.GerritHost)
		So(result.GerritRevision.Project, ShouldEqual, areq.Project)
		So(result.GerritRevision.Change, ShouldEqual, areq.GerritChange)
		So(result.GerritRevision.GitUrl, ShouldEqual, areq.GitURL)
		So(result.GerritRevision.GitRef, ShouldEqual, areq.GitRef)
		So(result.RevisionNumber, ShouldEqual, 7)
		So(tutils.Timestamp(result.RequestedTime), ShouldEqual, areq.Received)
		So(len(result.Files), ShouldEqual, len(areq.Files))
		for i := 0; i < len(result.Files); i++ {
			So(result.Files[i], ShouldResembleProto, &areq.Files[i])
		}
		So(len(result.Comments), ShouldEqual, len(comments))
		for i, gcomment := range result.Comments {
			tcomment := tricium.Data_Comment{}
			err := jsonpb.UnmarshalString(string(comments[i].Comment), &tcomment)
			So(err, ShouldBeNil)
			So(&tcomment, ShouldResembleProto, gcomment.Comment)
			So(gcomment.Analyzer, ShouldEqual, comments[i].Analyzer)
			So(gcomment.CreatedTime, ShouldResembleProto, tutils.TimestampProto(comments[i].CreationTime))
			platforms, _ := tricium.GetPlatforms(comments[i].Platforms)
			So(gcomment.Platforms, ShouldResemble, platforms)
			So(gcomment.Selected, ShouldEqual, selections[i].Included)
		}
	})
}

func TestCommentFetchingFunctions(t *testing.T) {
	Convey("Test Environment", t, func() {
		ctx := triciumtest.Context()

		// Add a request with no Gerrit details; it will not be fetched.
		So(ds.Put(ctx, &track.AnalyzeRequest{ID: 11}), ShouldBeNil)
		// Add two requests for the same CL.
		So(ds.Put(ctx, &track.AnalyzeRequest{
			ID:           22,
			GitRef:       "refs/changes/99/99/1",
			GerritHost:   "example.com",
			GerritChange: "p~master~I2222",
		}), ShouldBeNil)
		So(ds.Put(ctx, &track.AnalyzeRequest{
			ID:           23,
			GitRef:       "refs/changes/99/99/2",
			GerritHost:   "example.com",
			GerritChange: "p~master~I2222",
		}), ShouldBeNil)
		// And one more request with the same change ID but different host.
		So(ds.Put(ctx, &track.AnalyzeRequest{
			ID:           33,
			GitRef:       "refs/changes/99/99/1",
			GerritHost:   "other.test",
			GerritChange: "p~master~I2222",
		}), ShouldBeNil)

		Convey("A non-existent change has no runs", func() {
			keys, err := fetchRequestKeysByChange(ctx, "none.test", "none~m~Iabcd")
			So(len(keys), ShouldEqual, 0)
			So(err, ShouldBeNil)
		})

		Convey("No runs match if there are no Gerrit details", func() {
			keys, err := fetchRequestKeysByChange(ctx, "none.test", "")
			So(keys, ShouldBeNil)
			So(err, ShouldNotBeNil)
		})

		Convey("Two keys are fetched for a change with two runs", func() {
			keys, err := fetchRequestKeysByChange(ctx, "example.com", "p~master~I2222")
			So(len(keys), ShouldEqual, 2)
			So(err, ShouldBeNil)
		})

		Convey("One key is fetched for a change with one runs", func() {
			keys, err := fetchRequestKeysByChange(ctx, "other.test", "p~master~I2222")
			So(len(keys), ShouldEqual, 1)
			So(err, ShouldBeNil)
		})

		// In addition to the runs, add some comments and comment feedback.
		// In this example, run 22 has comments, some of which have not useful feedback.
		// Run 23 (same CL, different run) has one comment with not useful feedback.
		run22WorkerKey := ds.MakeKey(
			ctx, "AnalyzeRequest", 22, "WorkflowRun", 1,
			"FunctionRun", "Foo", "WorkerRun", "Foo_UBUNTU")

		c1 := &track.Comment{Parent: run22WorkerKey, Category: "Foo/C1"}
		So(ds.Put(ctx, c1), ShouldBeNil)
		c1Key := ds.KeyForObj(ctx, c1)
		So(ds.Put(ctx, &track.CommentFeedback{Parent: c1Key, ID: 1, NotUsefulReports: 1}), ShouldBeNil)

		c2 := &track.Comment{Parent: run22WorkerKey, Category: "Foo/C2"}
		So(ds.Put(ctx, c2), ShouldBeNil)
		c2Key := ds.KeyForObj(ctx, c2)
		So(ds.Put(ctx, &track.CommentFeedback{Parent: c2Key, ID: 1, NotUsefulReports: 2}), ShouldBeNil)

		c3 := &track.Comment{Parent: run22WorkerKey, Category: "Foo/C3"}
		So(ds.Put(ctx, c3), ShouldBeNil)
		c3Key := ds.KeyForObj(ctx, c3)
		So(ds.Put(ctx, &track.CommentFeedback{Parent: c3Key, ID: 1, NotUsefulReports: 0}), ShouldBeNil)

		run23WorkerKey := ds.MakeKey(
			ctx, "AnalyzeRequest", 23, "WorkflowRun", 1,
			"FunctionRun", "Foo", "WorkerRun", "Foo_UBUNTU")

		c4 := &track.Comment{Parent: run23WorkerKey, Category: "Foo/C4"}
		So(ds.Put(ctx, c4), ShouldBeNil)
		c4Key := ds.KeyForObj(ctx, c4)
		So(ds.Put(ctx, &track.CommentFeedback{Parent: c4Key, ID: 1, NotUsefulReports: 1}), ShouldBeNil)

		Convey("No CommentFeedback keys fetched for empty input", func() {
			keys, err := fetchAllCommentFeedback(ctx, nil)
			So(len(keys), ShouldEqual, 0)
			So(err, ShouldBeNil)
		})

		Convey("No CommentFeedback keys fetched for run with no comments", func() {
			keys, err := fetchAllCommentFeedback(ctx, []*ds.Key{ds.MakeKey(ctx, "AnalyzeRequest", 33)})
			So(len(keys), ShouldEqual, 0)
			So(err, ShouldBeNil)
		})

		Convey("Two CommentFeedback keys fetched for run with two comments", func() {
			keys, err := fetchAllCommentFeedback(ctx, []*ds.Key{ds.MakeKey(ctx, "AnalyzeRequest", 22)})
			// Comments c1 and c2 have "not useful" feedback.
			So(len(keys), ShouldEqual, 2)
			So(err, ShouldBeNil)
		})

		Convey("One CommentFeedback key fetched for run with one comment", func() {
			keys, err := fetchAllCommentFeedback(ctx, []*ds.Key{ds.MakeKey(ctx, "AnalyzeRequest", 23)})
			So(len(keys), ShouldEqual, 1)
			So(err, ShouldBeNil)
		})

		Convey("Three CommentFeedback keys for both of those runs together", func() {
			keys, err := fetchAllCommentFeedback(ctx, []*ds.Key{
				ds.MakeKey(ctx, "AnalyzeRequest", 22),
				ds.MakeKey(ctx, "AnalyzeRequest", 23),
			})
			So(len(keys), ShouldEqual, 3)
			So(err, ShouldBeNil)
		})

		Convey("suppressedCategories returns all not useful categories for all patchsets", func() {
			categories := suppressedCategories(ctx, "example.com", "p~master~I2222")
			So(categories, ShouldResemble, stringset.NewFromSlice("Foo/C1", "Foo/C2", "Foo/C4"))
		})

		Convey("suppressedCategories returns an empty set for nonexistent CLs", func() {
			categories := suppressedCategories(ctx, "example.com", "p~master~I999")
			So(categories, ShouldBeEmpty)
		})
	})
}
