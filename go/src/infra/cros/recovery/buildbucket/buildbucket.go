// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package buildbucket

import (
	"context"

	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/grpc/prpc"

	"infra/libs/skylab/buildbucket"
)

// Client is a buildbucket client.
type Client = buildbucket.Client

// NewLabpackClient creates a new client directed at the "labpack" chromeos builder.
func NewLabpackClient(ctx context.Context, authFlags authcli.Flags, prpcOptions *prpc.Options) (buildbucket.Client, error) {
	return buildbucket.NewClient(ctx, authFlags, prpcOptions, "chromeos", "labpack", "labpack")
}
