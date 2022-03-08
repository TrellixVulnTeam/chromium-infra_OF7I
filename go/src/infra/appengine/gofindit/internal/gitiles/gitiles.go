// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package gitiles contains logic of interacting with Gitiles.
package gitiles

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"infra/appengine/gofindit/model"
	"infra/appengine/gofindit/util"

	"go.chromium.org/luci/common/logging"

	buildbucketpb "go.chromium.org/luci/buildbucket/proto"
)

var MockedGitilesClientKey = "mocked gitiles client"

// GetChangeLogs gets a list of ChangeLogs in revision range by batch.
// The changelogs contain revisions in (startRevision, endRevision]
func GetChangeLogs(c context.Context, repoUrl string, startRevision string, endRevision string) ([]*model.ChangeLog, error) {
	changeLogs := []*model.ChangeLog{}
	gitilesClient := GetClient(c)
	for {
		url := getChangeLogsUrl(repoUrl, startRevision, endRevision)
		params := map[string]string{
			"n":           "100", // Number of revisions to return in each batch
			"name-status": "1",   // We set name-status so the detailed changelogs are return
			"format":      "json",
		}
		data, err := gitilesClient.sendRequest(c, url, params)
		if err != nil {
			return nil, err
		}

		// Gerrit prepends )]}' to json-formatted response.
		// We need to remove it before converting to struct
		prefix := ")]}'\n"
		data = strings.TrimPrefix(data, prefix)

		resp := &model.ChangeLogResponse{}
		if err = json.Unmarshal([]byte(data), resp); err != nil {
			return nil, fmt.Errorf("Failed to unmarshal data %w. Data: %s", err, data)
		}

		changeLogs = append(changeLogs, resp.Log...)
		if resp.Next == "" { // No more revision
			break
		}
		endRevision = resp.Next // Gitiles returns the most recent revision first
	}
	return changeLogs, nil
}

func GetClient(c context.Context) Client {
	if mockClient, ok := c.Value(MockedGitilesClientKey).(*MockedGitilesClient); ok {
		return mockClient
	}
	return &GitilesClient{}
}

func GetRepoUrl(c context.Context, commit *buildbucketpb.GitilesCommit) string {
	return fmt.Sprintf("https://%s/%s", commit.Host, commit.Project)
}

// getChangeLogsUrl generates a URL for change logs in (startRevision, endRevision]
func getChangeLogsUrl(repoUrl string, startRevision string, endRevision string) string {
	return fmt.Sprintf("%s/+log/%s..%s", repoUrl, startRevision, endRevision)
}

// We need the interface for testing purpose
type Client interface {
	sendRequest(c context.Context, url string, params map[string]string) (string, error)
}

type GitilesClient struct{}

func (cl *GitilesClient) sendRequest(c context.Context, url string, params map[string]string) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	q := req.URL.Query()
	for k, v := range params {
		q.Add(k, v)
	}
	req.URL.RawQuery = q.Encode()

	logging.Infof(c, "Sending request to gitiles %s", req.URL.String())
	return util.SendHttpRequest(c, req, 30*time.Second)
}
