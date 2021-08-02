// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"context"

	"infra/unifiedfleet/app/model/configuration"

	"go.chromium.org/chromiumos/config/go/test/api"
)

// GetDutAttribute returns DutAttribute for the given DutAttribute ID from datastore.
func GetDutAttribute(ctx context.Context, id string) (*api.DutAttribute, error) {
	return configuration.GetDutAttribute(ctx, id)
}

// ListDutAttributes lists the DutAttributes from datastore.
func ListDutAttributes(ctx context.Context, keysOnly bool) ([]*api.DutAttribute, error) {
	return configuration.ListDutAttributes(ctx, keysOnly)
}
