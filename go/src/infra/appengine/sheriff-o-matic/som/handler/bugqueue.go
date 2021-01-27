// Copyright 2016 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"infra/appengine/sheriff-o-matic/som/client"
	"infra/appengine/sheriff-o-matic/som/model"
	"infra/monorail"
	monorailv3 "infra/monorailv2/api/v3/api_proto"

	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/tsmon/field"
	"go.chromium.org/luci/common/tsmon/metric"
	"go.chromium.org/luci/gae/service/datastore"
	"go.chromium.org/luci/gae/service/memcache"
	"go.chromium.org/luci/server/auth"
	"go.chromium.org/luci/server/router"
	"google.golang.org/grpc"
)

const (
	bugQueueCacheFormat = "bugqueue-%s"
)

var (
	bugQueueLength = metric.NewInt("bug_queue_length", "Number of bugs in queue.",
		nil, field.String("label"))
)

// IssueClient is for testing purpose
type IssueClient interface {
	SearchIssues(context.Context, *monorailv3.SearchIssuesRequest, ...grpc.CallOption) (*monorailv3.SearchIssuesResponse, error)
}

// SearchIssueResponseExtras wraps around SearchIssuesResponse
// but adds some information
type SearchIssueResponseExtras struct {
	*monorailv3.SearchIssuesResponse
	Extras map[string]interface{} `json:"extras,omitempty"`
}

// BugQueueHandler handles bug queue-related requests.
type BugQueueHandler struct {
	Monorail               monorail.MonorailClient
	MonorailIssueClient    IssueClient
	DefaultMonorailProject string
}

// A bit of a hack to let us mock getBugsFromMonorail.
func (bqh *BugQueueHandler) getBugsFromMonorail(c context.Context, q string, projectID string,
	can monorail.IssuesListRequest_CannedQuery) (*monorail.IssuesListResponse, error) {
	// TODO(martiniss): make this look up request info based on Tree datastore
	// object
	req := &monorail.IssuesListRequest{
		ProjectId: projectID,
		Q:         q,
	}

	req.Can = can

	before := clock.Now(c)

	res, err := bqh.Monorail.IssuesList(c, req)
	if err != nil {
		logging.Errorf(c, "error getting issuelist: %v", err)
		return nil, err
	}

	logging.Debugf(c, "Fetch to monorail took %v. Got %d bugs.", clock.Now(c).Sub(before), res.TotalResults)
	return res, nil
}

func (bqh *BugQueueHandler) getBugsFromMonorailV3(c context.Context, q string, projectID string) (*SearchIssueResponseExtras, error) {
	// TODO (nqmtuan): Implement pagination if necessary
	projects := []string{"projects/" + projectID}
	req := monorailv3.SearchIssuesRequest{
		Projects: projects,
		Query:    q,
	}
	before := clock.Now(c)
	resp, err := bqh.MonorailIssueClient.SearchIssues(c, &req)
	if err != nil {
		logging.Errorf(c, "error searching issues: %v", err)
		return nil, err
	}
	logging.Debugf(c, "Fetch to monorail took %v. Got %d bugs.", clock.Now(c).Sub(before), len(resp.Issues))

	// Add extra priority field, since Monorail response does not indicate
	// which field is priority field
	respExtras := &SearchIssueResponseExtras{
		SearchIssuesResponse: resp,
	}
	priorityField, err := client.GetMonorailPriorityField(c, projectID)
	if err == nil {
		respExtras.Extras = make(map[string]interface{})
		respExtras.Extras["priority_field"] = priorityField
	}
	return respExtras, nil
}

// Switches chromium.org emails for google.com emails and vice versa.
// Note that chromium.org emails may be different from google.com emails.
func getAlternateEmail(email string) string {
	s := strings.Split(email, "@")
	if len(s) != 2 {
		return email
	}

	user, domain := s[0], s[1]
	if domain == "chromium.org" {
		return fmt.Sprintf("%s@google.com", user)
	}
	return fmt.Sprintf("%s@chromium.org", user)
}

// GetBugQueueHandler returns a set of bugs for the current user and tree.
func (bqh *BugQueueHandler) GetBugQueueHandler(ctx *router.Context) {
	c, w, p := ctx.Context, ctx.Writer, ctx.Params

	label := p.ByName("label")
	key := fmt.Sprintf(bugQueueCacheFormat, label)

	item, err := memcache.GetKey(c, key)

	if err == memcache.ErrCacheMiss {
		logging.Debugf(c, "No bug queue data for %s in memcache, refreshing...", label)
		item, err = bqh.refreshBugQueue(c, label, bqh.GetMonorailProjectNameFromLabel(c, label))
	}

	if err != nil {
		errStatus(c, w, http.StatusInternalServerError, err.Error())
		return
	}

	result := item.Value()

	w.Header().Set("Content-Type", "application/json")
	w.Write(result)
}

// GetUncachedBugsHandler bypasses the cache to return the bug queue for current user and tree.
// TODO (nqmtuan): This is not used. We should remove it.
func (bqh *BugQueueHandler) GetUncachedBugsHandler(ctx *router.Context) {
	c, w, p := ctx.Context, ctx.Writer, ctx.Params

	label := p.ByName("label")

	user := auth.CurrentIdentity(c)
	email := getAlternateEmail(user.Email())
	q := fmt.Sprintf("is:open (label:%[1]s -has:owner OR label:%[1]s owner:%s OR owner:%s label:%[1]s)",
		label, user.Email(), email)

	bugs, err := bqh.getBugsFromMonorailV3(c, q, bqh.GetMonorailProjectNameFromLabel(c, label))
	if err != nil && bugs != nil {
		bugQueueLength.Set(c, int64(len(bugs.Issues)), label)
	}

	out, err := json.Marshal(bugs)
	if err != nil {
		errStatus(c, w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(out)
}

// Makes a request to Monorail for bugs in a label and caches the results.
func (bqh *BugQueueHandler) refreshBugQueue(c context.Context, label string, projectID string) (memcache.Item, error) {
	q := fmt.Sprintf("is:open (label=%s)", label)
	res, err := bqh.getBugsFromMonorailV3(c, q, projectID)

	if err != nil {
		return nil, err
	}

	bytes, err := json.Marshal(res)
	if err != nil {
		return nil, err
	}

	item := memcache.NewItem(c, fmt.Sprintf(bugQueueCacheFormat, label)).SetValue(bytes)

	if err = memcache.Set(c, item); err != nil {
		return nil, err
	}

	return item, nil
}

// RefreshBugQueueHandler updates the cached bug queue for current tree.
func (bqh *BugQueueHandler) RefreshBugQueueHandler(ctx *router.Context) {
	c, w, p := ctx.Context, ctx.Writer, ctx.Params
	label := p.ByName("label")
	item, err := bqh.refreshBugQueue(c, label, bqh.GetMonorailProjectNameFromLabel(c, label))

	if err != nil {
		errStatus(c, w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(item.Value())
}

// GetMonorailProjectNameFromLabel returns the default monorail project name
// configured in project settings by comparing the bugqueue label.
func (bqh *BugQueueHandler) GetMonorailProjectNameFromLabel(c context.Context, label string) string {

	if bqh.DefaultMonorailProject == "" {
		bqh.DefaultMonorailProject = bqh.queryTreeForLabel(c, label)
	}

	return bqh.DefaultMonorailProject
}

func (bqh *BugQueueHandler) queryTreeForLabel(c context.Context, label string) string {
	q := datastore.NewQuery("Tree")
	trees := []*model.Tree{}
	if err := datastore.GetAll(c, q, &trees); err == nil {
		for _, tree := range trees {
			if tree.BugQueueLabel == label && tree.DefaultMonorailProjectName != "" {
				return tree.DefaultMonorailProjectName
			}
		}
	}
	return "chromium"
}
