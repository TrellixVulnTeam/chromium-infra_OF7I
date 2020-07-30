// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"context"

	ufspb "infra/unifiedfleet/api/v1/proto"
	"infra/unifiedfleet/app/model/configuration"
)

// GetDHCPConfig returns a dhcp reecord based on hostname from datastore.
func GetDHCPConfig(ctx context.Context, hostname string) (*ufspb.DHCPConfig, error) {
	return configuration.GetDHCPConfig(ctx, hostname)
}
