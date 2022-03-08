// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package util contains utility functions
package util

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"go.chromium.org/luci/server/auth"
)

func SendHttpRequest(c context.Context, req *http.Request, timeout time.Duration) (string, error) {
	c, cancel := context.WithTimeout(c, 30*time.Second)
	defer cancel()

	transport, err := auth.GetRPCTransport(c, auth.AsProject)

	client := &http.Client{
		Transport: transport,
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()
	status := resp.StatusCode
	if status != http.StatusOK {
		return "", fmt.Errorf("Bad response code: %v", status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("Cannot get response body %w", err)
	}
	return string(body), nil
}
