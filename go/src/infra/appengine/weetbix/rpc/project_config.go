// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package rpc

import (
	"context"
	"time"

	"go.chromium.org/luci/common/errors"

	"infra/appengine/weetbix/internal/config/compiledcfg"
)

// readProjectConfig reads project config. This is intended for use in
// top-level RPC handlers. The caller should directly return any errors
// returned as the error of the RPC; the returned errors have been
// properly annotated with an appstatus.
func readProjectConfig(ctx context.Context, project string) (*compiledcfg.ProjectConfig, error) {
	cfg, err := compiledcfg.Project(ctx, project, time.Time{})
	if err != nil {
		if err == compiledcfg.NotExistsErr {
			return nil, failedPreconditionError(errors.New("project does not exist in Weetbix"))
		}
		// GRPCifyAndLog will log this, and report an internal error to the caller.
		return nil, errors.Annotate(err, "obtain project config").Err()
	}
	return cfg, nil
}
