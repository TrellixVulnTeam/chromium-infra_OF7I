// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package reviewer

import (
	"context"

	"infra/appengine/rubber-stamper/config"
	"infra/appengine/rubber-stamper/internal/gerrit"
	"infra/appengine/rubber-stamper/tasks/taskspb"
)

// ReviewChange reviews a CL and then either gives a Bot-Commit +1 label or
// leaves a comment explain why the CL shouldn't be passed and removes itself
// as a reviewer.
func ReviewChange(ctx context.Context, t *taskspb.ChangeReviewTask) error {
	cfg, err := config.Get(ctx)
	if err != nil {
		return err
	}
	hostCfg := cfg.HostConfigs[t.Host]

	gc, err := gerrit.GetCurrentClient(ctx, t.Host+"-review.googlesource.com")
	if err != nil {
		return err
	}

	invalidFiles, err := reviewBenignFileChange(ctx, hostCfg, gc, t)
	if err != nil {
		return err
	}
	if len(invalidFiles) > 0 {
		// Invalid BenignFileChange.
		// TODO: leave comments in Gerrit
		// TODO: remove reviewer in Gerrit.
		return nil
	}

	// TODO: Bot-Commit +1 & Owners-Override +1
	return nil
}
