// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package logdog contains logic of interacting with Logdog.
package logdog

import (
	"context"
	"infra/appengine/gofindit/util"
	"net/http"
	"time"

	"go.chromium.org/luci/common/logging"
)

var MockedLogdogClientKey = "mocked logdog client"

// GetLogFromViewUrl gets the log from the log's viewURL
func GetLogFromViewUrl(c context.Context, viewUrl string) (string, error) {
	cl := GetClient(c)
	return cl.GetLog(c, viewUrl)
}

type LogdogClient struct{}

func (cl *LogdogClient) GetLog(c context.Context, viewUrl string) (string, error) {
	req, err := http.NewRequest("GET", viewUrl, nil)
	if err != nil {
		return "", err
	}

	q := req.URL.Query()
	q.Add("format", "raw")
	req.URL.RawQuery = q.Encode()

	logging.Infof(c, "Sending request to logdog %s", req.URL.String())
	return util.SendHttpRequest(c, req, 30*time.Second)
}

// We need the interface for testing purpose
type Client interface {
	GetLog(c context.Context, viewUrl string) (string, error)
}

func GetClient(c context.Context) Client {
	if mockClient, ok := c.Value(MockedLogdogClientKey).(*MockedLogdogClient); ok {
		return mockClient
	}
	return &LogdogClient{}
}
