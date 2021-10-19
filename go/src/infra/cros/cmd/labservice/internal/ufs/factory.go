// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package ufs

import (
	"context"

	ufsapi "infra/unifiedfleet/api/v1/rpc"
)

// Used for UFS RPC calls.
const userAgent = "labservice/0.1.0"

// A ClientFactory is used to create UFS clients.
// The client needs a context which is request specific, so the client
// needs to be created per incoming request.
type ClientFactory struct {
	// UFS gRPC service address.
	Service string
	// Path to service account JSON file.
	ServiceAccountPath string
}

func (f *ClientFactory) NewClient(ctx context.Context) (ufsapi.FleetClient, error) {
	return ufsapi.NewClient(
		ctx,
		ufsapi.ServiceName(f.Service),
		ufsapi.ServiceAccountJSONPath(f.ServiceAccountPath),
		ufsapi.UserAgent(userAgent))
}
