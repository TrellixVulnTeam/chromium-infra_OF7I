// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package handlers

import (
	"go.chromium.org/luci/server/router"
)

// RegisterRoutes registers routes explicitly handled by the handler.
func (h *Handlers) RegisterRoutes(r *router.Router, mw router.MiddlewareChain) {
	r.GET("/api/authState", mw, h.GetAuthState)
	r.GET("/", mw, h.IndexPage)
}
