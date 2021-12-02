// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package handlers

import (
	"net/http"

	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/server/router"

	"infra/appengine/weetbix/internal/clustering/runs"
)

// GetReclusteringProgress serves a GET request for
// /api/projects/:project/reclusteringProgress.
func (h *Handlers) GetReclusteringProgress(ctx *router.Context) {
	projectID, ok := obtainProjectOrError(ctx)
	if !ok {
		return
	}
	progress, err := runs.ReadReclusteringProgress(ctx.Context, projectID)
	if err != nil {
		logging.Errorf(ctx.Context, "Reading progress: %s", err)
		http.Error(ctx.Writer, "Internal server error.", http.StatusInternalServerError)
		return
	}

	respondWithJSON(ctx, progress)
}
