// Copyright 2019 The LUCI Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package backend

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/wrappers"

	"go.chromium.org/gae/service/info"
	"go.chromium.org/gae/service/memcache"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/sync/parallel"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"infra/appengine/arquebus/app/backend/model"
	"infra/appengine/arquebus/app/config"
	"infra/monorailv2/api/api_proto"
)

const (
	// OptOutLabel stops Arquebus updating the issue, if added.
	OptOutLabel = "Arquebus-Opt-Out"
	// IssueUpdateMaxConcurrency is the maximum number of
	// monorail.UpdateIssue()s that can be invoked in parallel.
	IssueUpdateMaxConcurrency = 4
	// The length of an issue update throttle window, which is created after
	// an issue update performed to prevent the issue from being updated again
	// during the period.
	issueUpdateThrottleDuration = 1 * time.Hour
)

// searchAndUpdateIssues searches and update issues for the Assigner.
func searchAndUpdateIssues(c context.Context, assigner *model.Assigner, task *model.Task) (int32, error) {
	assignee, ccs, err := findAssigneeAndCCs(c, assigner, task)
	if err != nil {
		task.WriteLog(c, "Failed to find assignees and CCs; %s", err)
		return 0, err
	}
	if assignee == nil && ccs == nil {
		// early stop if there is no one available to assign or cc issues to.
		task.WriteLog(
			c, "No one was available to be assigned or CCed; "+
				"skipping issue searches and updates",
		)
		return 0, nil
	}

	mc := getMonorailClient(c)
	issues, err := searchIssues(c, mc, assigner, task)
	if err != nil {
		task.WriteLog(c, "Failed to search issues; %s", err)
		return 0, err
	}

	// As long as it succeeded to update at least one issue, the task is
	// not marked as failed.
	nUpdated, nFailed := updateIssues(c, mc, assigner, task, issues, assignee, ccs)
	if nUpdated == 0 && nFailed > 0 {
		return 0, errors.New("all issue updates failed")
	}
	return nUpdated, nil
}

func searchIssues(c context.Context, mc monorail.IssuesClient, assigner *model.Assigner, task *model.Task) ([]*monorail.Issue, error) {
	task.WriteLog(c, "Started searching issues")

	// Inject -label:Arquebus-Opt-Out into the search query.
	var query strings.Builder
	s := strings.Split(assigner.IssueQuery.Q, " OR ")
	for i, q := range s {
		// Split("ABC OR ", " OR ") returns ["ABC", ""]
		if q == "" {
			continue
		}
		query.WriteString(fmt.Sprintf("%s -label:%s", q, OptOutLabel))
		if i < (len(s) - 1) {
			query.WriteString(" OR ")
		}
	}

	res, err := mc.ListIssues(c, &monorail.ListIssuesRequest{
		Query:        query.String(),
		CannedQuery:  uint32(monorail.SearchScope_OPEN),
		ProjectNames: assigner.IssueQuery.ProjectNames,

		// TODO(crbug/965385) - paginate through the search results, until
		// the total number of issues to be updated reaches the maximum number
		// of issues to be updated in one Task.
		Pagination: &monorail.Pagination{
			Start:    0,
			MaxItems: 20,
		},
	})
	if err != nil {
		return nil, err
	}

	if len(res.Issues) > 0 {
		var buf bytes.Buffer
		for i, issue := range res.Issues {
			if issue == nil {
				continue
			}
			buf.WriteString(fmt.Sprintf("%s/%d", issue.ProjectName, issue.LocalId))
			if i < len(res.Issues)-1 {
				buf.WriteString(" ")
			}
		}
		task.WriteLog(c, "Found %d issues: %s", len(res.Issues), buf.String())
	} else {
		task.WriteLog(c, "Found 0 issues")
	}
	return res.Issues, nil
}

// updateIssues update the issues with the desired status and property values.
//
// It is expected that Monorail may become flaky, unavailable, or slow
// temporarily. Therefore, updateIssues tries to update as many issues as
// possible.
func updateIssues(c context.Context, mc monorail.IssuesClient, assigner *model.Assigner, task *model.Task, issues []*monorail.Issue, assignee *monorail.UserRef, ccs []*monorail.UserRef) (nUpdated, nFailed int32) {
	mh := config.Get(c).MonorailHostname

	isThrottled := func(issue *monorail.Issue) bool {
		key := fmt.Sprintf("%s/%s/%d", mh, issue.ProjectName, issue.LocalId)
		item, err := memcache.GetKey(c, key)

		if err == nil {
			writeTaskLogWithLink(c, task, issue, "issue update throttled - %s", item.Expiration())
			return true
		}
		// TODO(1027998): Replace memcached with DS.
		if err == memcache.ErrCacheMiss {
			item.SetExpiration(issueUpdateThrottleDuration)
			writeTaskLogWithLink(
				c, task, issue, "Throttling issue updates with expiration %s",
				item.Expiration(),
			)
			if err = memcache.Set(c, item); err != nil {
				writeTaskLogWithLink(c, task, issue, "failed to throttle issue updates: %s", err.Error())
			}
			return false
		}
		// memcache.GetKey() failed.
		writeTaskLogWithLink(
			c, task, issue, "failed to look up a throttle duration: %s", err.Error(),
		)
		return false
	}

	update := func(issue *monorail.Issue) {
		delta, err := createIssueDelta(c, mc, task, issue, assignee, ccs)
		switch {
		case err != nil:
			atomic.AddInt32(&nFailed, 1)
			return
		case delta == nil:
			writeTaskLogWithLink(
				c, task, issue, "No delta found; skip updating",
			)
			return
		}
		if assigner.IsDryRun {
			// dry-run is checked here, because it is expected to run all
			// the steps, but UpdateIssue.
			writeTaskLogWithLink(
				c, task, issue, "Dry-run is set; skip updating",
			)
			return
		}
		writeTaskLogWithLink(c, task, issue, "Updating")
		_, err = mc.UpdateIssue(c, &monorail.UpdateIssueRequest{
			IssueRef: &monorail.IssueRef{
				ProjectName: issue.ProjectName,
				LocalId:     issue.LocalId,
			},
			SendEmail:      true,
			Delta:          delta,
			CommentContent: genCommentContent(c, assigner, task),
		})
		if err != nil {
			writeTaskLogWithLink(c, task, issue, "UpdateIssue failed: %s", err)
			atomic.AddInt32(&nFailed, 1)
			return
		}
		atomic.AddInt32(&nUpdated, 1)
		return
	}

	parallel.WorkPool(IssueUpdateMaxConcurrency, func(tasks chan<- func() error) {
		for _, issue := range issues {
			// In-Scope variable for goroutine closure.
			issue := issue
			tasks <- func() error {
				if !isThrottled(issue) {
					update(issue)
				}
				return nil
			}
		}
	})
	return
}

// writeTaskLogWithLink invokes task.WriteLog with a link to the issue.
//
// It's necessary to have a link of the issue added to each log, because
// multiple issue updates are performed in parallel, and it will be hard to
// group logs by the issue, if there is no issue information in each log.
func writeTaskLogWithLink(c context.Context, task *model.Task, issue *monorail.Issue, format string, args ...interface{}) {
	format = fmt.Sprintf(
		"[https://%s/p/%s/issues/detail?id=%d] %s",
		config.Get(c).MonorailHostname, issue.ProjectName, issue.LocalId,
		format,
	)
	task.WriteLog(c, format, args...)
}

func createIssueDelta(c context.Context, mc monorail.IssuesClient, task *model.Task, issue *monorail.Issue, assignee *monorail.UserRef, ccs []*monorail.UserRef) (*monorail.IssueDelta, error) {
	// Monorail search responses often contain several minutes old snapshot
	// of Issue property values. Therefore, it is necessary to invoke
	// GetIssues() to get the fresh data before generating IssueDelta.
	//
	// TODO(crbug/monorail/5629) - If Monorail supports test-and-update API,
	// then use the API instead of GetIssue() + UpdateIssue()
	res, err := mc.GetIssue(c, &monorail.GetIssueRequest{
		IssueRef: &monorail.IssueRef{
			ProjectName: issue.ProjectName,
			LocalId:     issue.LocalId,
		},
	})
	if err != nil {
		// NotFound shouldn't be considered as an error. It is just that
		// the search response contained stale data.
		if status.Code(err) == codes.NotFound {
			writeTaskLogWithLink(c, task, issue, "The issue no longer exists")
			return nil, nil
		}
		writeTaskLogWithLink(c, task, issue, "GetIssue failed: %s", err)
		logging.Errorf(c, "GetIssue failed: %s", err)
		return nil, err
	}
	if issue = res.GetIssue(); issue == nil {
		// If a response doesn't contain a valid Issue object, then it's
		// likely a bug of Monorail.
		writeTaskLogWithLink(c, task, issue, "Invalid response from GetIssue")
		return nil, errors.New("invalid response from GetIssue")
	}

	// log the current status of the issue so that it becomes easy to find
	// how Arquebus made the issue update decision.
	logIssueStatus(c, task, issue)

	delta := &monorail.IssueDelta{}
	needUpdate := false
	// iff the issue has the intended owner, set the status to "Assigned".
	// Otherwise, keep the existing status.
	if assignee != nil && !proto.Equal(issue.OwnerRef, assignee) {
		needUpdate = true
		delta.OwnerRef = assignee
		delta.Status = &wrappers.StringValue{Value: "Assigned"}
		writeTaskLogWithLink(
			c, task, issue, "Found a new owner: %s", assignee.DisplayName,
		)
	}
	ccsToAdd := findCcsToAdd(task, issue.CcRefs, ccs)
	if len(ccsToAdd) > 0 {
		needUpdate = true
		delta.CcRefsAdd = ccsToAdd
		writeTaskLogWithLink(
			c, task, issue, "Found new CC(s): %s", delta.CcRefsAdd,
		)
	}
	if needUpdate {
		return delta, nil
	}
	return nil, nil
}

// findCcsToAdd() returns a list of UserRefs that have not been cc-ed yet, but
// should be.
func findCcsToAdd(task *model.Task, existingCCs, proposedCCs []*monorail.UserRef) []*monorail.UserRef {
	if len(proposedCCs) == 0 {
		return []*monorail.UserRef{}
	}
	ccmap := make(map[uint64]*monorail.UserRef, len(existingCCs))
	for _, cc := range existingCCs {
		ccmap[cc.UserId] = cc
	}

	var ccsToAdd []*monorail.UserRef
	for _, cc := range proposedCCs {
		if _, exist := ccmap[cc.UserId]; !exist {
			ccsToAdd = append(ccsToAdd, cc)
		}
	}
	return ccsToAdd
}

func genCommentContent(c context.Context, assigner *model.Assigner, task *model.Task) string {
	taskURL := fmt.Sprintf(
		"https://%s.appspot.com/assigner/%s/task/%d",
		info.AppID(c), url.QueryEscape(assigner.ID), task.ID,
	)
	messages := []string{
		"Issue update by Arquebus.",
		"Task details: " + taskURL,
		fmt.Sprintf(
			"To stop Arquebus updating this issue, please add the label %q.",
			OptOutLabel,
		),
	}
	if assigner.Comment != "" {
		messages = append(
			messages, "-----------------------------------------------",
		)
		messages = append(messages, assigner.Comment)
	}
	return strings.Join(messages, "\n")
}

// logIssueStatus adds a log entry with the status of an issue.
func logIssueStatus(c context.Context, task *model.Task, issue *monorail.Issue) {
	status := ""
	if issue.StatusRef != nil {
		status = issue.StatusRef.Status
	}
	owner := ""
	if issue.OwnerRef != nil {
		owner = issue.OwnerRef.DisplayName
	}

	var buf bytes.Buffer
	for i, user := range issue.CcRefs {
		if user == nil {
			continue
		}
		if i < len(issue.CcRefs)-1 {
			buf.WriteString(" ")
		}
		buf.WriteString(user.DisplayName)
	}

	msg := strings.Join([]string{
		"The current issue status",
		"- Status: %s",
		"- Owner: %s",
		"- CCs: %s",
		"- ModifiedTime: %d",
	}, "\n")
	writeTaskLogWithLink(
		c, task, issue, msg, status, owner, buf.String(), issue.ModifiedTimestamp,
	)
}
