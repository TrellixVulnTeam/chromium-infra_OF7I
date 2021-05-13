// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	ufspb "infra/unifiedfleet/api/v1/models"
	"infra/unifiedfleet/app/util"

	authclient "go.chromium.org/luci/auth"
	"go.chromium.org/luci/server/auth"
)

const (
	hwidEndpoint      = "chromeoshwid-pa.googleapis.com"
	hwidEndpointScope = "https://www.googleapis.com/auth/chromeoshwid"
)

type HWIDClient struct {
	hc *http.Client
}

func InitHWIDClient(ctx context.Context) (*HWIDClient, error) {
	tr, err := auth.GetRPCTransport(ctx, auth.AsSelf, auth.WithScopes(authclient.OAuthScopeEmail, hwidEndpointScope))
	if err != nil {
		return nil, err
	}
	return &HWIDClient{
		hc: &http.Client{Transport: tr},
	}, nil
}

func (c *HWIDClient) QueryHWID(ctx context.Context, hwid string) (*map[string]string, error) {
	u := &url.URL{
		Scheme: "https",
		Host:   hwidEndpoint,
		Path:   fmt.Sprintf("v2/dutlabel/%s", hwid),
	}
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	dutLabel := &ufspb.GetDutLabelResponse{}
	if err := util.ExecuteRequest(ctx, c.hc, req, dutLabel); err != nil {
		return nil, err
	}

	return parseGetDutLabelResponse(dutLabel), nil
}

func parseGetDutLabelResponse(resp *ufspb.GetDutLabelResponse) *map[string]string {
	data := map[string]string{}
	for _, l := range resp.GetDutLabel().GetLabels() {
		data[l.GetName()] = l.GetValue()
	}
	return &data
}
