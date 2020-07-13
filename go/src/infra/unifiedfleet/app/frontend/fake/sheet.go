// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package fake

import (
	"context"
	"encoding/json"
	"io/ioutil"

	"go.chromium.org/luci/common/errors"
	"google.golang.org/api/sheets/v4"
)

// SheetClient mocks the sheet.ClientInterface
type SheetClient struct {
}

// Get mocks the sheet.ClientInterface.Get()
func (sc *SheetClient) Get(ctx context.Context, sheetID string, ranges []string) (*sheets.Spreadsheet, error) {
	if sheetID == "test_sheet" {
		// It's hardcoded for unittest, please run `make test` in base path /infra/unifiedfleet.
		return SheetData("../frontend/fake/sheet_data.json")
	}
	return nil, errors.Reason("Unspecified mock path %s", sheetID).Err()
}

// SheetData returns the fake sheet get API response for other testers
func SheetData(path string) (*sheets.Spreadsheet, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var resp sheets.Spreadsheet
	if err := json.Unmarshal(b, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
