// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package fake

import (
	"context"

	ufspb "infra/unifiedfleet/api/v1/models"

	"go.chromium.org/luci/common/errors"
)

// HwidClient mocks the hwid.ClientInterface
type HwidClient struct {
}

func mockDutLabel() *ufspb.DutLabel {
	return &ufspb.DutLabel{
		PossibleLabels: []string{
			"test-possible-1",
			"test-possible-2",
		},
		Labels: []*ufspb.DutLabel_Label{
			{
				Name:  "test-label-1",
				Value: "test-value-1",
			},
			{
				Name:  "Sku",
				Value: "test-sku",
			},
			{
				Name:  "variant",
				Value: "test-variant",
			},
		},
	}
}

// QueryHwid mocks hwid.ClientInterface.QueryHwid()
func (hc *HwidClient) QueryHwid(ctx context.Context, hwid string) (*ufspb.DutLabel, error) {
	if hwid == "test" || hwid == "test-no-cached-hwid-data" {
		return mockDutLabel(), nil
	} else if hwid == "test-no-server" {
		return &ufspb.DutLabel{}, errors.Reason("Mocked failure; could not query data").Err()
	}
	return &ufspb.DutLabel{}, errors.Reason("Unspecified mock hwid").Err()
}
