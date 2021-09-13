// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package hwid

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	ufspb "infra/unifiedfleet/api/v1/models"
	"infra/unifiedfleet/app/util"
)

const hwidEndpoint = "chromeoshwid-pa.googleapis.com"

type Client struct {
	Hc *http.Client
}

type ClientInterface interface {
	QueryHwid(ctx context.Context, hwid string) (*ufspb.DutLabel, error)
}

// QueryHwid takes a hwid string and queries the Hwid server for corresponding
// DutLabels
func (c *Client) QueryHwid(ctx context.Context, hwid string) (*ufspb.DutLabel, error) {
	u := &url.URL{
		Scheme: "https",
		Host:   hwidEndpoint,
		Path:   fmt.Sprintf("v2/dutlabel/%s", hwid),
	}
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	resp := &ufspb.GetDutLabelResponse{}
	if err := util.ExecuteRequest(ctx, c.Hc, req, resp); err != nil {
		return nil, err
	}

	return resp.GetDutLabel(), nil
}
