// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package utils

import (
	"context"
	"fmt"

	"google.golang.org/grpc/metadata"
	"infra/cmd/shivas/meta"
)

// SetupContext sets up context with client major version number
func SetupContext(ctx context.Context) context.Context {
	md := metadata.Pairs(meta.ClientVersion, fmt.Sprintf("%d", meta.Major))
	return metadata.NewOutgoingContext(ctx, md)
}
