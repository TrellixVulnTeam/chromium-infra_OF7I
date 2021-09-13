// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package fake

import (
	"context"

	ufspb "infra/unifiedfleet/api/v1/models"
)

// HwidClient mocks the hwid.ClientInterface
type HwidClient struct {
}

// QueryHwid mocks hwid.ClientInterface.QueryHwid()
func (hc *HwidClient) QueryHwid(ctx context.Context, hwid string) (*ufspb.DutLabel, error) {
	return &ufspb.DutLabel{}, nil
}
