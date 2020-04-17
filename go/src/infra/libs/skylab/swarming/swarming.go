// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package swarming implements a client for creating skylab-swarming tasks and
// getting their results.
package swarming

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	swarming_api "go.chromium.org/luci/common/api/swarming/swarming/v1"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/retry"
	"go.chromium.org/luci/common/retry/transient"
	"google.golang.org/api/googleapi"
)

// Client is a swarming client for creating tasks and waiting for their results.
type Client struct {
	SwarmingService *swarming_api.Service
	server          string
}

// ListedHost is a collection of information about the DUT managed by a particular bot.
type ListedHost struct {
	Hostname string
}

func (l *ListedHost) String() string {
	return l.Hostname
}

// New creates a new Client.
func New(ctx context.Context, h *http.Client, server string) (*Client, error) {
	service, err := newSwarmingService(ctx, h, server)
	if err != nil {
		return nil, err
	}
	c := &Client{
		SwarmingService: service,
		server:          server,
	}
	return c, nil
}

const swarmingAPISuffix = "_ah/api/swarming/v1/"

func newSwarmingService(ctx context.Context, h *http.Client, server string) (*swarming_api.Service, error) {
	s, err := swarming_api.New(h)
	if err != nil {
		return nil, errors.Annotate(err, "create swarming client").Err()
	}

	s.BasePath = server + swarmingAPISuffix
	return s, nil
}

// CreateTask creates a swarming task based on the given request,
// retrying transient errors.
func (c *Client) CreateTask(ctx context.Context, req *swarming_api.SwarmingRpcsNewTaskRequest) (*swarming_api.SwarmingRpcsTaskRequestMetadata, error) {
	var resp *swarming_api.SwarmingRpcsTaskRequestMetadata
	createTask := func() error {
		var err error
		resp, err = c.SwarmingService.Tasks.New(req).Context(ctx).Do()
		return err
	}

	if err := callWithRetries(ctx, "create task", createTask); err != nil {
		return nil, err
	}
	return resp, nil
}

func getFullTaskList(ctx context.Context, call *swarming_api.TasksListCall) ([]*swarming_api.SwarmingRpcsTaskResult, error) {
	var tr []*swarming_api.SwarmingRpcsTaskResult
	var err error
	var tl *swarming_api.SwarmingRpcsTaskList
	for {
		tl, err = call.Context(ctx).Do()
		if err != nil {
			return nil, err
		}
		tr = append(tr, tl.Items...)
		if tl.Cursor == "" {
			break
		}
		call = call.Cursor(tl.Cursor)
	}
	return tr, nil
}

// getActiveLeaseTasksForDimensions gets active leases for any combination of dimensions and
// does no sanity checking on the dimensions provided.
func (c *Client) getActiveLeaseTasksForDimensions(ctx context.Context, dims map[string]string) ([]*swarming_api.SwarmingRpcsTaskResult, error) {
	var tr []*swarming_api.SwarmingRpcsTaskResult
	getResult := func() error {
		tr = nil
		var err error
		var call *swarming_api.TasksListCall
		tags := dimsToTags(dims)
		call = c.SwarmingService.Tasks.List().Tags(tags...).State("RUNNING")
		r, err := getFullTaskList(ctx, call)
		if err != nil {
			return err
		}
		tr = append(tr, r...)
		call = c.SwarmingService.Tasks.List().Tags(tags...).State("PENDING")
		r, err = getFullTaskList(ctx, call)
		if err != nil {
			return err
		}
		tr = append(tr, r...)
		return nil
	}
	if err := callWithRetries(ctx, "get result", getResult); err != nil {
		return nil, errors.Annotate(err, fmt.Sprintf("get active leases tasks for dims (%s)", formatDims(dims))).Err()
	}
	return tr, nil
}

// GetActiveLeaseTasksForModel gets active leases *specifically* targeted to a model.
// Leases that apply to a specific hostname are not counted here.
// TODO(gregorynisbet): Count leases that target specific hostnames as well.
func (c *Client) GetActiveLeaseTasksForModel(ctx context.Context, model string) ([]*swarming_api.SwarmingRpcsTaskResult, error) {
	// TODO(gregorynisbet): Can we search by tags as well?
	var dims = map[string]string{
		"label-model": model,
		"skylab-tool": "lease",
	}
	return c.getActiveLeaseTasksForDimensions(ctx, dims)
}

// GetActiveLeaseTasksForHost returns a list of RUNNING or PENDING lease tasks,
// retrying transient errors.
// hostname cannot be empty.
func (c *Client) GetActiveLeaseTasksForHost(ctx context.Context, hostname string) ([]*swarming_api.SwarmingRpcsTaskResult, error) {
	var dims = map[string]string{
		"dut_name":    hostname,
		"skylab-tool": "lease",
	}
	return c.getActiveLeaseTasksForDimensions(ctx, dims)
}

// CancelTask cancels a swarming task by taskID,
// retrying transient errors.
func (c *Client) CancelTask(ctx context.Context, taskID string) error {
	ctx, cf := context.WithTimeout(ctx, 60*time.Second)
	defer cf()
	var tc *swarming_api.SwarmingRpcsCancelResponse
	getResult := func() error {
		var err error
		req := &swarming_api.SwarmingRpcsTaskCancelRequest{
			KillRunning: true,
		}
		tc, err = c.SwarmingService.Task.Cancel(taskID, req).Context(ctx).Do()
		return err
	}
	if err := callWithRetries(ctx, "get result", getResult); err != nil {
		return errors.Annotate(err, fmt.Sprintf("cancel task %s", taskID)).Err()
	}
	if !tc.Ok {
		return errors.New(fmt.Sprintf("task %s is not successfully canceled", taskID))
	}
	return nil
}

// GetResults gets results for the tasks with given IDs,
// retrying transient errors.
func (c *Client) GetResults(ctx context.Context, IDs []string) ([]*swarming_api.SwarmingRpcsTaskResult, error) {
	results := make([]*swarming_api.SwarmingRpcsTaskResult, len(IDs))
	for i, ID := range IDs {
		var r *swarming_api.SwarmingRpcsTaskResult
		getResult := func() error {
			var err error
			r, err = c.SwarmingService.Task.Result(ID).Context(ctx).Do()
			return err
		}
		if err := callWithRetries(ctx, "get result", getResult); err != nil {
			return nil, errors.Annotate(err, fmt.Sprintf("get swarming result for task %s", ID)).Err()
		}
		results[i] = r
	}
	return results, nil
}

// GetResultsForTags gets results for tasks that match all the given tags,
// retrying transient errors.
func (c *Client) GetResultsForTags(ctx context.Context, tags []string) ([]*swarming_api.SwarmingRpcsTaskResult, error) {
	var results *swarming_api.SwarmingRpcsTaskList
	getResults := func() error {
		var err error
		results, err = c.SwarmingService.Tasks.List().Tags(tags...).Context(ctx).Do()
		return err
	}
	if err := callWithRetries(ctx, "get result", getResults); err != nil {
		return nil, errors.Annotate(err, fmt.Sprintf("get swarming result for tags %s", tags)).Err()
	}

	return results.Items, nil
}

// GetRequests gets the task requests for the given task IDs,
// retrying transient errors.
func (c *Client) GetRequests(ctx context.Context, IDs []string) ([]*swarming_api.SwarmingRpcsTaskRequest, error) {
	requests := make([]*swarming_api.SwarmingRpcsTaskRequest, len(IDs))
	for i, ID := range IDs {
		var request *swarming_api.SwarmingRpcsTaskRequest
		getRequest := func() error {
			var err error
			request, err = c.SwarmingService.Task.Request(ID).Context(ctx).Do()
			return err
		}
		if err := callWithRetries(ctx, "get request", getRequest); err != nil {
			return nil, errors.Annotate(err, fmt.Sprintf("rerun task %s", ID)).Err()
		}
		requests[i] = request
	}
	return requests, nil
}

// GetTaskState gets the state of the given task,
// retrying transient errors.
func (c *Client) GetTaskState(ctx context.Context, ID string) (*swarming_api.SwarmingRpcsTaskStates, error) {
	var result *swarming_api.SwarmingRpcsTaskStates
	getState := func() error {
		var err error
		result, err = c.SwarmingService.Tasks.GetStates().TaskId(ID).Context(ctx).Do()
		return err
	}
	if err := callWithRetries(ctx, "get state", getState); err != nil {
		return nil, errors.Annotate(err, fmt.Sprintf("get task state for task ID %s", ID)).Err()
	}
	return result, nil
}

// GetTaskOutputs gets the task outputs for the given IDs,
// retrying transient errors.
func (c *Client) GetTaskOutputs(ctx context.Context, IDs []string) ([]*swarming_api.SwarmingRpcsTaskOutput, error) {
	results := make([]*swarming_api.SwarmingRpcsTaskOutput, len(IDs))
	for i, ID := range IDs {
		var result *swarming_api.SwarmingRpcsTaskOutput
		getResult := func() error {
			var err error
			result, err = c.SwarmingService.Task.Stdout(ID).Context(ctx).Do()
			return err
		}
		if err := callWithRetries(ctx, "get result", getResult); err != nil {
			return nil, errors.Annotate(err, fmt.Sprintf("get swarming stdout for task %s", ID)).Err()
		}
		results[i] = result
	}
	return results, nil
}

// BotExists checks if an bot exists with the given dimensions.
func (c *Client) BotExists(ctx context.Context, dims []*swarming_api.SwarmingRpcsStringPair) (bool, error) {
	var resp *swarming_api.SwarmingRpcsBotList
	err := callWithRetries(ctx, "check bot exists", func() error {
		call := c.SwarmingService.Bots.List().Dimensions(flattenStringPairs(dims)...).IsDead("FALSE").Limit(1)
		r, err := call.Context(ctx).Do()
		if err != nil {
			return errors.Annotate(err, "bot exists").Err()
		}
		if r == nil {
			return errors.Reason("bot exists: nil RPC response").Err()
		}
		// Assign to captured variable only on success.
		resp = r
		return nil
	})
	if err != nil {
		return false, err
	}
	return len(resp.Items) > 0, nil
}

// getSwarmingRpcsBotList -- get a SwarmingRpcsBotList, retrying as appropriate for swarming
func getSwarmingRpcsBotList(ctx context.Context, c *Client, call *swarming_api.BotsListCall) (*swarming_api.SwarmingRpcsBotList, error) {
	var tl *swarming_api.SwarmingRpcsBotList
	f := func() error {
		var err error
		tl, err = call.Context(ctx).Do()
		return err
	}
	err := callWithRetries(ctx, "get bot list", f)
	if err != nil {
		return nil, err
	}
	return tl, nil
}

// GetBots returns a slice of bots
func (c *Client) GetBots(ctx context.Context, dims []*swarming_api.SwarmingRpcsStringPair) ([]*swarming_api.SwarmingRpcsBotInfo, error) {
	var out []*swarming_api.SwarmingRpcsBotInfo

	call := c.SwarmingService.Bots.List().Dimensions(flattenStringPairs(dims)...)
	for {
		tl, err := getSwarmingRpcsBotList(ctx, c, call)
		if err != nil {
			return nil, err
		}
		call = call.Cursor(tl.Cursor)
		for _, item := range tl.Items {
			out = append(out, item)
		}
		if tl.Cursor == "" {
			return out, nil
		}
	}
}

// GetListedBots returns information about the DUTs managed by bots satisfying particular dimensions.
func (c *Client) GetListedBots(ctx context.Context, dims []*swarming_api.SwarmingRpcsStringPair) ([]*ListedHost, error) {
	var out []*ListedHost

	bots, err := c.GetBots(ctx, dims)
	if err != nil {
		return nil, err
	}

	for _, bot := range bots {
		var err error
		newEntry := &ListedHost{}
		newEntry.Hostname, err = LookupDimension(bot.Dimensions, "dut_name")
		if err != nil {
			continue
		}
		out = append(out, newEntry)
	}

	return out, nil
}

// LookupDimension gets a single string value associated with a dimension
func LookupDimension(dims []*swarming_api.SwarmingRpcsStringListPair, key string) (string, error) {
	for _, pair := range dims {
		if pair.Key == key {
			if len(pair.Value) == 0 {
				return "", fmt.Errorf("found key, 0 values")
			}
			if len(pair.Value) > 1 {
				return "", fmt.Errorf("found key, (%d) values", len(pair.Value))
			}
			return pair.Value[0], nil
		}
	}
	// TODO(gregorynisbet): truncate key if it's too long
	return "", fmt.Errorf("no corresponding value for key (%s)", key)
}

func flattenStringPairs(pairs []*swarming_api.SwarmingRpcsStringPair) []string {
	ss := make([]string, len(pairs))
	for i, p := range pairs {
		ss[i] = fmt.Sprintf("%s:%s", p.Key, p.Value)
	}
	return ss
}

// dimsToTags takes swarming dimensions specified as a map
// and returns them as a list of Key:Value pairs.
func dimsToTags(m map[string]string) []string {
	var out []string
	for k, v := range m {
		out = append(out, fmt.Sprintf("%s:%s", k, v))
	}
	return out
}

// GetTaskURL gets a URL for the task with the given ID.
func (c *Client) GetTaskURL(taskID string) string {
	return TaskURL(c.server, taskID)
}

var retryableCodes = map[int]bool{
	http.StatusInternalServerError: true, // 500
	http.StatusBadGateway:          true, // 502
	http.StatusServiceUnavailable:  true, // 503
	http.StatusGatewayTimeout:      true, // 504
	http.StatusInsufficientStorage: true, // 507
}

func retryParams() retry.Iterator {
	// crbug.com/1061200: Swarming's response on internal error indicates that
	// we should retry in 30 seconds. We try after (15 + 22.5 + 33.75) seconds
	// to balance responsiveness against additional load due to retries.
	return &retry.ExponentialBackoff{
		Limited: retry.Limited{
			Delay:   15 * time.Second,
			Retries: 3,
		},
		Multiplier: 1.5,
	}
}

func tagErrIfTransient(f func() error) func() error {
	return func() error {
		err := f()
		tag := false
		errors.Walk(
			err,
			func(ierr error) bool {
				if errIsTransient(ierr) {
					tag = true
					return false
				}
				return true
			},
		)
		if tag {
			return transient.Tag.Apply(err)
		}
		return err
	}
}

func errIsTransient(err error) bool {
	if err == nil {
		return false
	}
	if e, ok := err.(net.Error); ok && e.Temporary() {
		return true
	}
	if e, ok := err.(*googleapi.Error); ok && retryableCodes[e.Code] {
		return true
	}
	if strings.Contains(err.Error(), "connection reset by peer") {
		return true
	}
	if strings.Contains(err.Error(), "unexpected EOF") {
		return true
	}
	return false
}

// callWithRetries calls the given function, retrying transient swarming
// errors, with swarming-appropriate backoff and delay.
func callWithRetries(ctx context.Context, opname string, f func() error) error {
	return retry.Retry(ctx, transient.Only(retryParams), tagErrIfTransient(f), retry.LogCallback(ctx, opname))
}

// TaskURL returns a URL to inspect a task with the given ID.
func TaskURL(swarmingService string, taskID string) string {
	return fmt.Sprintf("%stask?id=%s", swarmingService, taskID)
}

// TaskListURLForTags returns a tasklist URL filtered by the given tags.
func TaskListURLForTags(swarmingService string, tags []string) string {
	h := parseSwarmingHost(swarmingService)
	u := url.URL{
		Scheme: "https",
		Host:   h,
		Path:   "tasklist",
	}
	q := u.Query()
	for _, t := range tags {
		q.Add("f", t)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func parseSwarmingHost(s string) string {
	u, err := url.Parse(s)
	// Not a valid URL, return back the input string.
	if err != nil || u.Scheme == "" {
		return s
	}
	return u.Host
}

// formatDims converts dimensions stored in a map into a human-readable format.
func formatDims(m map[string]string) string {
	var out []string
	for k, v := range m {
		out = append(out, fmt.Sprintf("%s:%s", k, v))
	}
	return strings.Join(out, ", ")
}
