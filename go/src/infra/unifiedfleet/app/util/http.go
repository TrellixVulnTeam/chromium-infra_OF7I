// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package util

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"go.chromium.org/luci/common/logging"
	"golang.org/x/net/context/ctxhttp"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// HTTPError wraps the http response errors.
type HTTPError struct {
	Method     string
	URL        *url.URL
	StatusCode int
	Body       string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("unexpected response: method=%q url=%q status=%d message=%q", e.Method, e.URL, e.StatusCode, e.Body)
}

func ExecuteRequest(ctx context.Context, hc *http.Client, req *http.Request, value proto.Message) error {
	resp, err := ctxhttp.Do(ctx, hc, req)
	defer resp.Body.Close()
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			logging.Debugf(ctx, "fail to read resp.Body: %s", err)
		}
		return &HTTPError{
			Method:     req.Method,
			URL:        req.URL,
			StatusCode: resp.StatusCode,
			Body:       string(body),
		}
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("unable to read the body for %s: %w", req.URL, err)
	}
	logging.Debugf(ctx, "response:\n%v", string(body))

	if err := protojson.Unmarshal(body, value); err != nil {
		return fmt.Errorf("decode response body: %w", err)
	}
	return nil
}
