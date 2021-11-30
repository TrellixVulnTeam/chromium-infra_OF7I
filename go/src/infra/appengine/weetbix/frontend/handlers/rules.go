// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package handlers

import (
	"net/http"

	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/server/router"
	"go.chromium.org/luci/server/span"

	// Store auth sessions in the datastore.
	_ "go.chromium.org/luci/server/encryptedcookies/session/datastore"

	"infra/appengine/weetbix/internal/clustering/rules"
)

// ListRules serves a GET request for
// /api/projects/:project/rules.
func (h *Handlers) ListRules(ctx *router.Context) {
	transctx, cancel := span.ReadOnlyTransaction(ctx.Context)
	defer cancel()

	projectID := ctx.Params.ByName("project")
	rs, err := rules.ReadActive(transctx, projectID)
	if err != nil {
		logging.Errorf(ctx.Context, "Reading rules: %s", err)
		http.Error(ctx.Writer, "Internal server error.", http.StatusInternalServerError)
		return
	}

	respondWithJSON(ctx, rs)
}
