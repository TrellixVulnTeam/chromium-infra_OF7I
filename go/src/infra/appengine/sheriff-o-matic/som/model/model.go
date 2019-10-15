// Copyright 2016 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"infra/appengine/sheriff-o-matic/som/model/gen"

	"cloud.google.com/go/bigquery"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
	"google.golang.org/appengine"

	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/gae/service/info"
	"go.chromium.org/luci/common/bq"
	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/server/auth"
	"golang.org/x/net/context"
)

const (
	bqDatasetID = "events"
	bqTableID   = "annotations"
)

// Tree is a tree which sheriff-o-matic receives and groups alerts for.
type Tree struct {
	Name                       string   `gae:"$id" json:"name"`
	DisplayName                string   `json:"display_name"`
	AlertStreams               []string `json:"alert_streams,omitempty"`
	BugQueueLabel              string   `json:"bug_queue_label,omitempty"`
	HelpLink                   string   `json:"help_link,omitempty"`
	GerritProject              string   `json:"gerrit_project,omitempty"`
	GerritInstance             string   `json:"gerrit_instance,omitempty"`
	DefaultMonorailProjectName string   `json:"default_monorail_project_name,omitempty"`
}

// BuildBucketTree holds "tree" information about buildbucket builders.
type BuildBucketTree struct {
	TreeName     string         `json:"tree_name"`
	TreeBuilders []*TreeBuilder `json:"tree_builders"`
}

// TreeBuilder is a builder that belongs to a particular tree or trees.
type TreeBuilder struct {
	Project  string   // buildbucket project name
	Bucket   string   // buildbucket bucket name
	Builders []string // buildbucket builder name
}

// AlertJSON is the JSON blob of an alert for a tree.
type AlertJSON struct {
	ID           string         `gae:"$id" json:"-"`
	Tree         *datastore.Key `gae:"$parent"`
	Date         time.Time
	Contents     []byte `gae:",noindex"`
	Resolved     bool
	AutoResolved bool
	ResolvedDate time.Time
}

// RevisionSummaryJSON is the JSON blob of a RevisionSummary for a tree.
type RevisionSummaryJSON struct {
	ID       string         `gae:"$id" json:"-"`
	Tree     *datastore.Key `gae:"$parent"`
	Date     time.Time
	Contents []byte `gae:",noindex"`
}

// ResolveRequest is the format of the request to resolve alerts.
type ResolveRequest struct {
	Keys     []string `json:"keys"`
	Resolved bool     `json:"resolved"`
}

// ResolveResponse is the format of the response to resolve alerts.
type ResolveResponse struct {
	Tree     string   `json:"tree"`
	Keys     []string `json:"keys"`
	Resolved bool     `json:"resolved"`
}

// Annotation is any information sheriffs want to annotate an alert with. For
// example, a bug where the cause of the alert is being solved.
type Annotation struct {
	Tree             *datastore.Key `gae:"$parent"`
	KeyDigest        string         `gae:"$id"`
	Key              string         `gae:",noindex" json:"key"`
	Bugs             []MonorailBug  `gae:",noindex" json:"bugs"`
	Comments         []Comment      `gae:",noindex" json:"comments"`
	SnoozeTime       int            `json:"snoozeTime"`
	GroupID          string         `gae:",noindex" json:"group_id"`
	ModificationTime time.Time
}

// MonorailBug stores data to differentiate bugs by projects.
type MonorailBug struct {
	BugID     string `json:"id"`        // This should match monorail.Issue.id
	ProjectID string `json:"projectId"` // This should match monorail.Issue.projectId
}

// Comment is the format for the data in the Comments property of an Annotation
type Comment struct {
	Text string    `json:"text"`
	User string    `json:"user"`
	Time time.Time `json:"time"`
}

type annotationAdd struct {
	Time     int           `json:"snoozeTime"`
	Bugs     []MonorailBug `json:"bugs"`
	Comments []string      `json:"comments"`
	GroupID  string        `json:"group_id"`
}

type annotationRemove struct {
	Time     bool          `json:"snoozeTime"`
	Bugs     []MonorailBug `json:"bugs"`
	Comments []int         `json:"comments"`
	GroupID  bool          `json:"group_id"`
}

func appendToBugList(knownBugs []MonorailBug, newBug MonorailBug) []MonorailBug {
	for _, knownBug := range knownBugs {
		if knownBug.BugID == newBug.BugID && knownBug.ProjectID == newBug.ProjectID {
			return knownBugs
		}
	}
	return append(knownBugs, newBug)
}

func removeFromBugList(knownBugs []MonorailBug, bugToRemove MonorailBug) []MonorailBug {
	for i, knownBug := range knownBugs {
		if knownBug.BugID == bugToRemove.BugID && knownBug.ProjectID == bugToRemove.ProjectID {
			return append(knownBugs[:i], knownBugs[i+1:]...)
		}
	}
	return knownBugs
}

// Add adds some data to an annotation. Returns true if a refresh of annotation
// metadata (currently monorail data) is required, and any errors encountered.
func (a *Annotation) Add(c context.Context, r io.Reader) (bool, error) {
	change := &annotationAdd{}
	needRefresh := false

	err := json.NewDecoder(r).Decode(change)
	if err != nil {
		return needRefresh, err
	}

	modified := false

	if change.Time != 0 {
		a.SnoozeTime = change.Time
		modified = true
	}

	// Check if changed bugs are new, and append to annotation Bugs list.
	if change.Bugs != nil {
		oldBugsCount := len(a.Bugs)
		for _, newBug := range change.Bugs {
			a.Bugs = appendToBugList(a.Bugs, newBug)
		}
		if oldBugsCount != len(a.Bugs) {
			needRefresh = true
			modified = true
		}
	}

	user := auth.CurrentIdentity(c)
	commentTime := clock.Now(c)
	if change.Comments != nil {
		comments := make([]Comment, len(change.Comments))
		for i, c := range change.Comments {
			comments[i].Text = c
			comments[i].User = user.Email()
			comments[i].Time = commentTime
		}

		a.Comments = append(a.Comments, comments...)
		modified = true
	}

	if change.GroupID != "" {
		a.GroupID = change.GroupID
		modified = true
	}

	if modified {
		a.ModificationTime = clock.Now(c)
	}

	evt := createAnnotationEvent(c, a, gen.SOMAnnotationEvent_ADD)
	evt.User = user.Email()
	for _, changedBug := range change.Bugs {
		evt.Bugs = append(evt.Bugs, &gen.SOMAnnotationEvent_MonorailBug{
			BugId:     changedBug.BugID,
			ProjectId: changedBug.ProjectID,
		})
	}
	if ts, err := intToTimestamp(a.SnoozeTime); err != nil {
		evt.SnoozeTime = ts
	} else {
		logging.Errorf(c, "error getting timestamp proto: %v", err)
	}

	evt.GroupId = change.GroupID

	var ct *timestamp.Timestamp
	if ct, err = ptypes.TimestampProto(commentTime); err != nil {
		logging.Errorf(c, "error getting timestamp proto: %v", err)
	}

	for _, text := range change.Comments {
		evt.Comments = append(evt.Comments, &gen.SOMAnnotationEvent_Comment{
			Text: text,
			Time: ct,
		})
	}

	if err := writeAnnotationEvent(c, evt); err != nil {
		logging.Errorf(c, "error writing annotation event to bigquery: %v", err)
		// Continue. This isn't fatal.
	}

	return needRefresh, nil
}

func intToTimestamp(s int) (*timestamp.Timestamp, error) {
	if s == 0 {
		return nil, fmt.Errorf("Cannot convert 0 to timestamp.Timestamp")
	}

	ret, err := ptypes.TimestampProto(time.Unix(int64(s/1000), 0))
	return ret, err
}

// Remove removes some data to an annotation. Returns if a refreshe of annotation
// metadata (currently monorail data) is required, and any errors encountered.
func (a *Annotation) Remove(c context.Context, r io.Reader) (bool, error) {
	change := &annotationRemove{}

	err := json.NewDecoder(r).Decode(change)
	if err != nil {
		return false, err
	}

	modified := false

	if change.Time {
		a.SnoozeTime = 0
		modified = true
	}

	if change.Bugs != nil {
		for _, bug := range change.Bugs {
			a.Bugs = removeFromBugList(a.Bugs, bug)
		}
		modified = true
	}

	// Client passes in a list of comment indices to delete.
	deletedComments := []Comment{}
	for _, i := range change.Comments {
		if i < 0 || i >= len(a.Comments) {
			return false, errors.New("Invalid comment index")
		}
		deletedComments = append(deletedComments, a.Comments[i])
		a.Comments = append(a.Comments[:i], a.Comments[i+1:]...)
		modified = true
	}

	if change.GroupID {
		a.GroupID = ""
		modified = true
	}

	if modified {
		a.ModificationTime = clock.Now(c)
	}

	user := auth.CurrentIdentity(c)
	evt := createAnnotationEvent(c, a, gen.SOMAnnotationEvent_DELETE)
	evt.User = user.Email()
	for _, changedBug := range change.Bugs {
		evt.Bugs = append(evt.Bugs, &gen.SOMAnnotationEvent_MonorailBug{
			BugId:     changedBug.BugID,
			ProjectId: changedBug.ProjectID,
		})
	}
	if ts, err := intToTimestamp(a.SnoozeTime); err == nil {
		evt.SnoozeTime = ts
	} else {
		logging.Errorf(c, "error getting timestamp proto: %v", err)
	}

	evt.GroupId = a.GroupID
	for _, comment := range deletedComments {
		if ct, err := ptypes.TimestampProto(comment.Time); err == nil {
			evt.Comments = append(evt.Comments, &gen.SOMAnnotationEvent_Comment{
				Text: comment.Text,
				Time: ct,
			})
		} else {
			logging.Errorf(c, "error getting timestamp proto: %v", err)
		}
	}

	if err := writeAnnotationEvent(c, evt); err != nil {
		logging.Errorf(c, "error writing annotation event to bigquery: %v", err)
		// Continue. This isn't fatal.
	}

	return false, nil
}

func createAnnotationEvent(ctx context.Context, a *Annotation, operation gen.SOMAnnotationEvent_OperationType) *gen.SOMAnnotationEvent {

	evt := &gen.SOMAnnotationEvent{
		AlertKeyDigest: a.KeyDigest,
		AlertKey:       a.Key,
		RequestId:      appengine.RequestID(ctx),
		Operation:      operation,
	}

	if mt, err := ptypes.TimestampProto(a.ModificationTime); err == nil {
		evt.Timestamp = mt
		evt.ModificationTime = mt
	}

	for _, c := range a.Comments {
		if ct, err := ptypes.TimestampProto(c.Time); err == nil {
			evt.Comments = append(evt.Comments, &gen.SOMAnnotationEvent_Comment{
				Text: c.Text,
				Time: ct,
			})
		} else {
			logging.Errorf(ctx, "error getting timestamp proto: %v", err)
		}
	}

	return evt
}

func writeAnnotationEvent(c context.Context, evt *gen.SOMAnnotationEvent) error {
	client, err := bigquery.NewClient(c, info.AppID(c))
	if err != nil {
		return err
	}
	up := bq.NewUploader(c, client, bqDatasetID, bqTableID)
	up.SkipInvalidRows = true
	up.IgnoreUnknownValues = true

	return up.Put(c, evt)
}
