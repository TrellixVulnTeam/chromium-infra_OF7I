// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package gerrit

import (
	"context"
	"net/http"
	"time"

	"go.chromium.org/luci/common/api/gerrit"
	"go.chromium.org/luci/common/data/caching/lru"
	"go.chromium.org/luci/server/auth"
)

var gerritScope = "https://www.googleapis.com/auth/gerritcodereview"

type factory struct {
	clientCache *lru.Cache // caches Gerrit Clients
}

func newFactory() *factory {
	return &factory{
		// rubber-stamper supports <1000 gerrit hosts.
		clientCache: lru.New(1000),
	}
}

func (f *factory) makeClient(ctx context.Context, gerritHost string) (Client, error) {
	client, err := f.clientCache.GetOrCreate(ctx, gerritHost, func() (value interface{}, ttl time.Duration, err error) {
		t, err := auth.GetRPCTransport(ctx, auth.AsSelf, auth.WithScopes(gerritScope))
		if err != nil {
			return
		}
		value, err = gerrit.NewRESTClient(&http.Client{Transport: t}, gerritHost, true)
		ttl = 0
		return
	})
	if err != nil {
		return nil, err
	}
	return client.(Client), err
}
