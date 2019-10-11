// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package inventory includes gerrit-related functions.
// This is a TEMP package to mitigate crbug.com/1011236 & b/142340801.
// Mostly duplicated from crosskylabadmin, will delete it after inventory V2 is launched.
package inventory

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	gerritapi "go.chromium.org/luci/common/api/gerrit"
	gitilesapi "go.chromium.org/luci/common/api/gitiles"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/proto/gerrit"
	"go.chromium.org/luci/common/proto/gitiles"
	"go.chromium.org/luci/common/retry/transient"

	"golang.org/x/net/context"
)

const gerritHost = "chrome-internal-review.googlesource.com"
const gitilesHost = "chrome-internal.googlesource.com"
const project = "chromeos/infra_internal/skylab_inventory"
const branch = "master"
const crosPrefix = `chromeos`

// C includes clients required for inventory changes.
type C struct {
	gerritC  gerrit.GerritClient
	gitilesC gitiles.GitilesClient
}

// CreateC creates an inventory client with gerrit & gitiles clients.
func CreateC(hc *http.Client) (*C, error) {
	gerritC, err := gerritapi.NewRESTClient(hc, gerritHost, true)
	if err != nil {
		return nil, err
	}
	gitilesC, err := gitilesapi.NewRESTClient(hc, gitilesHost, true)
	if err != nil {
		return nil, err
	}
	return &C{
		gerritC:  gerritC,
		gitilesC: gitilesC,
	}, nil
}

func (c *C) fetchLatestSHA1(ctx context.Context) (string, error) {
	resp, err := c.gitilesC.Log(ctx, &gitiles.LogRequest{
		Project:    project,
		Committish: fmt.Sprintf("refs/heads/%s", branch),
		PageSize:   1,
	})
	if err != nil {
		return "", errors.Annotate(err, "fetch sha1 for %s branch of %s", branch, project).Err()
	}
	if len(resp.Log) == 0 {
		return "", fmt.Errorf("fetch sha1 for %s branch of %s: empty git-log", branch, project)
	}
	return resp.Log[0].GetId(), nil
}

// CreateChange creates a gerrit change.
func (c *C) CreateChange(ctx context.Context, subject string) (*gerrit.ChangeInfo, error) {
	latestSHA, err := c.fetchLatestSHA1(ctx)
	if err != nil {
		return nil, err
	}
	changeInfo, err := c.gerritC.CreateChange(ctx, &gerrit.CreateChangeRequest{
		Project:    project,
		Ref:        branch,
		Subject:    subject,
		BaseCommit: latestSHA,
	})
	if err != nil {
		return nil, err
	}
	return changeInfo, nil
}

// MakeDeleteHostChange edit CL by deleting the inventory file of a given host.
func (c *C) MakeDeleteHostChange(ctx context.Context, changeInfo *gerrit.ChangeInfo, host string) error {
	_, err := c.gerritC.DeleteEditFileContent(ctx, &gerrit.DeleteEditFileContentRequest{
		Number:   changeInfo.Number,
		Project:  changeInfo.Project,
		FilePath: invPathForDut(host),
	})
	return err
}

// e.g. data/skylab/chromeos6/chromeos6-***.textpb
func invPathForDut(hostname string) string {
	comps := strings.Split(hostname, "-")
	var path string
	if len(comps) == 0 || !strings.HasPrefix(comps[0], crosPrefix) {
		// Keep chromeos as prefix for regular expression.
		path = "chromeos-misc"
	} else {
		path = comps[0]
	}
	return fmt.Sprintf("data/skylab/%s/%s.textpb", path, hostname)
}

// SubmitChange submit the change to gerrit.
func (c *C) SubmitChange(ctx context.Context, changeInfo *gerrit.ChangeInfo) error {
	if _, err := c.gerritC.ChangeEditPublish(ctx, &gerrit.ChangeEditPublishRequest{
		Number:  changeInfo.Number,
		Project: changeInfo.Project,
	}); err != nil {
		return err
	}

	ci, err := c.gerritC.GetChange(ctx, &gerrit.GetChangeRequest{
		Number:  changeInfo.Number,
		Options: []gerrit.QueryOption{gerrit.QueryOption_CURRENT_REVISION},
	})
	if err != nil {
		return err
	}

	if _, err = c.gerritC.SetReview(ctx, &gerrit.SetReviewRequest{
		Number:     changeInfo.Number,
		Project:    changeInfo.Project,
		RevisionId: ci.CurrentRevision,
		Labels: map[string]int32{
			"Code-Review": 2,
			"Verified":    1,
		},
	}); err != nil {
		return err
	}

	if _, err := c.gerritC.SubmitChange(ctx, &gerrit.SubmitChangeRequest{
		Number:  changeInfo.Number,
		Project: changeInfo.Project,
	}); err != nil {
		// Mark this error as transient so that the operation will be retried.
		// Errors in submit are mostly caused because of conflict with a concurrent
		// change to the inventory.
		return errors.Annotate(err, "commit file contents").Tag(transient.Tag).Err()
	}

	return nil
}

// AbandonChange abandon the change to gerrit.
func (c *C) AbandonChange(ctx context.Context, ci *gerrit.ChangeInfo) error {
	if _, err := c.gerritC.AbandonChange(ctx, &gerrit.AbandonChangeRequest{
		Number:  ci.Number,
		Project: ci.Project,
		Message: "CL cleanup on error",
	}); err != nil {
		return err
	}
	return nil
}

// ChangeURL returns a URL to the gerrit change with given changeNumber.
func ChangeURL(changeNumber int) (string, error) {
	p, err := url.PathUnescape(project)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("https://%s/c/%s/+/%d", gerritHost, p, changeNumber), nil
}
