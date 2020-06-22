// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package gs

import (
	"context"
	"io/ioutil"
	"net/http"

	"go.chromium.org/luci/common/gcloud/gs"
	"go.chromium.org/luci/common/logging"
)

// Client refers to a google storage client.
type Client struct {
	C gs.Client
}

// ClientInterface is the public API of a gs client
type ClientInterface interface {
	GetFile(ctx context.Context, path string) ([]byte, error)
}

// NewClient produces a new google storage client
func NewClient(ctx context.Context, t http.RoundTripper) (*Client, error) {
	gsc := &Client{}
	c, err := gs.NewProdClient(ctx, t)
	if err != nil {
		return nil, err
	}
	gsc.C = c
	return gsc, nil
}

// GetFile returns the contents of the file located in a google storage path
func (gsc *Client) GetFile(ctx context.Context, p string) ([]byte, error) {
	logging.Debugf(ctx, "getting file from path: %s", gs.Path(p))
	r, err := gsc.C.NewReader(gs.Path(p), 0, -1)
	if err != nil {
		return []byte{}, err
	}
	readBytes, err := ioutil.ReadAll(r)
	if err != nil {
		return []byte{}, err
	}
	return readBytes, nil
}
