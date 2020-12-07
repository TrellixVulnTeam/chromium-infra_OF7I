// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package sheet

import (
	"context"
	"net/http"

	"go.chromium.org/luci/common/errors"
	"google.golang.org/api/sheets/v4"
)

// Client consists of resources needed for querying gitiles and gerrit.
type Client struct {
	ss *sheets.Service
}

// ClientInterface is the public API of a Google spreadsheet client git client
type ClientInterface interface {
	Get(ctx context.Context, sheetID string, ranges []string) (*sheets.Spreadsheet, error)
}

// NewClient produces a new client using only simple types available in a command line context
func NewClient(ctx context.Context, hc *http.Client) (*Client, error) {
	c := &Client{}
	sheetsService, err := sheets.New(hc)
	if err != nil {
		return nil, err
	}
	c.ss = sheetsService
	return c, nil
}

// Get returns the content of a spreadsheet's range
func (c *Client) Get(ctx context.Context, sheet string, ranges []string) (*sheets.Spreadsheet, error) {
	if c.ss == nil {
		return nil, errors.Reason("please run NewClient first to initialize the sheets service").Err()
	}
	return c.ss.Spreadsheets.Get(sheet).Ranges(ranges...).IncludeGridData(true).Context(ctx).Do()
}
