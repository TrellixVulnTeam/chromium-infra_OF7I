// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package handlers

import (
	"context"

	"go.chromium.org/luci/server/router"
)

const testProject = "testproject"

// routerForTesting returns a *router.Router to use for testing
// handlers.
func routerForTesting(ctx context.Context) *router.Router {
	router := router.NewWithRootContext(ctx)
	prod := true
	h := NewHandlers("cloud-project", prod)
	h.RegisterRoutes(router, nil)
	return router
}
