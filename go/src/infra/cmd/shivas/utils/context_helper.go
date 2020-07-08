// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package utils

import (
	"context"
	"fmt"

	"google.golang.org/grpc/metadata"
	"infra/cmd/shivas/site"
)

// ClientVersion used as a key in metadata within context
const ClientVersion string = "clientversion"

// SetupContext sets up context with client major version number
func SetupContext(ctx context.Context) context.Context {
	md := metadata.Pairs(ClientVersion, fmt.Sprintf("%d", site.Major))
	return metadata.NewOutgoingContext(ctx, md)
}
