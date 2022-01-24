// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package handlers

import (
	"net/http"

	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/server/auth/xsrf"
	"go.chromium.org/luci/server/router"
)

// GetXSRFToken serves a GET request for
// /api/xsrfToken.
func (h *Handlers) GetXSRFToken(ctx *router.Context) {
	tok, err := xsrf.Token(ctx.Context)
	if err != nil {
		logging.Errorf(ctx.Context, "Obtain XSRF token: %s", err)
		http.Error(ctx.Writer, "Internal server error.", http.StatusInternalServerError)
		return
	}

	data := map[string]string{
		"token": tok,
	}
	respondWithJSON(ctx, data)
}
