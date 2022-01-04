// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package ufs

import (
	"context"
	"net/http"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/grpc/prpc"
	"go.chromium.org/luci/server/auth"

	ufsAPI "infra/unifiedfleet/api/v1/rpc"
)

// NewUFS client creates a new UFS client when given a hostname.
// The hostname should generally be read from the config.
func NewUFSClient(ctx context.Context, hostname string) (ufsAPI.FleetClient, error) {
	t, err := auth.GetRPCTransport(ctx, auth.AsSelf)
	if err != nil {
		return nil, errors.Annotate(err, "failed to get RPC transport for host %s", hostname).Err()
	}
	hc := &http.Client{Transport: t}
	return ufsAPI.NewFleetPRPCClient(&prpc.Client{
		C:       hc,
		Host:    hostname,
		Options: nil,
	}), nil
}
