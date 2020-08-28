// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package gitiles

import (
	"context"

	gitilesProto "go.chromium.org/luci/common/proto/gitiles"
)

var clientKey = "gitiles.client"

// GetClient returns gitilesClient from context. If not set, it panics.
func GetClient(ctx context.Context) gitilesProto.GitilesClient {
	v := ctx.Value(&clientKey)
	if v == nil {
		panic("Client is not set, use SetClient")
	}

	return v.(gitilesProto.GitilesClient)

}

// SetClient stores gitilesClient in context. Useful for unit testing.
func SetClient(ctx context.Context, client gitilesProto.GitilesClient) context.Context {
	return context.WithValue(ctx, &clientKey, client)
}
