// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package client

import (
	"context"

	kartepb "infra/cros/karte/api"
)

// ContextKey is an opaque type that holds a context key.
type contextKey string

// ContextConstant is an arbitrary value whose address will
// be the context key used to store and retrieve Karte contexts.
const contextConstant = contextKey("karte client")

// WithKarteClient adds a Karte client to the context.
func WithKarteClient(ctx context.Context, client kartepb.KarteClient) context.Context {
	return context.WithValue(ctx, contextConstant, client)
}

// GetKarteClient retrieves a Karte client from the context.
func GetKarteClient(ctx context.Context) kartepb.KarteClient {
	client, ok := ctx.Value(contextConstant).(kartepb.KarteClient)
	if !ok {
		panic("impossible: karte client from context does not have type KarteClient")
	}
	return client
}
