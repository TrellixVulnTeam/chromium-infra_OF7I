// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package handlers

import (
	"encoding/json"
	"net/http"

	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/server/router"

	// Store auth sessions in the datastore.
	_ "go.chromium.org/luci/server/encryptedcookies/session/datastore"

	"infra/appengine/weetbix/internal/config"
)

// Handlers provides methods servicing Weetbix HTTP routes.
type Handlers struct {
	cloudProject string
	// prod is set when running in production (not a dev workstation).
	prod bool
}

// NewHandlers initialises a new Handlers instance.
func NewHandlers(cloudProject string, prod bool) *Handlers {
	return &Handlers{cloudProject: cloudProject, prod: prod}
}

func obtainProjectOrError(ctx *router.Context) (project string, ok bool) {
	projectCfgs, err := config.Projects(ctx.Context)
	if err != nil {
		logging.Errorf(ctx.Context, "Obtain project config: %v", err)
		http.Error(ctx.Writer, "Internal server error.", http.StatusInternalServerError)
		return "", false
	}
	projectID := ctx.Params.ByName("project")
	if _, ok := projectCfgs[projectID]; !ok {
		http.Error(ctx.Writer, "Project does not exist in Weetbix.", http.StatusBadRequest)
		return "", false
	}
	return projectID, true
}

func respondWithJSON(ctx *router.Context, data interface{}) {
	bytes, err := json.Marshal(data)
	if err != nil {
		logging.Errorf(ctx.Context, "Marshalling JSON for response: %s", err)
		http.Error(ctx.Writer, "Internal server error.", http.StatusInternalServerError)
	}
	ctx.Writer.Header().Add("Content-Type", "application/json")
	if _, err := ctx.Writer.Write(bytes); err != nil {
		logging.Errorf(ctx.Context, "Writing JSON response: %s", err)
	}
}
