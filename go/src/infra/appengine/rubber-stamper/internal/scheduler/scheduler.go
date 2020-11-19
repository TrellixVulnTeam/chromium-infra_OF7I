// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package scheduler

import (
	"context"
	"fmt"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	gerritpb "go.chromium.org/luci/common/proto/gerrit"
	"go.chromium.org/luci/server/auth"

	"infra/appengine/rubber-stamper/config"
	"infra/appengine/rubber-stamper/internal/gerrit"
	"infra/appengine/rubber-stamper/tasks"
)

// ScheduleReviews add tasks into Cloud Tasks queue, where each task handles
// one CL's review.
func ScheduleReviews(ctx context.Context) error {
	signer := auth.GetSigner(ctx)
	if signer == nil {
		return errors.New("failed to get the Signer instance representing the service")
	}
	info, err := signer.ServiceInfo(ctx)
	if err != nil {
		return err
	}
	sa := info.ServiceAccountName

	cfg, err := config.Get(ctx)
	if err != nil {
		return err
	}

	errNum := 0
	for host := range cfg.HostConfigs {
		gc, err := gerrit.GetCurrentClient(ctx, getGerritHostURL(host))
		if err != nil {
			return err
		}

		listReq := &gerritpb.ListChangesRequest{
			Query:   "status:open r:" + sa,
			Options: []gerritpb.QueryOption{gerritpb.QueryOption_CURRENT_REVISION},
		}
		resp, err := gc.ListChanges(ctx, listReq)
		for _, cl := range resp.GetChanges() {
			err := tasks.EnqueueChangeReviewTask(ctx, host, cl)
			if err != nil {
				errNum = errNum + 1
				logging.WithError(err).Errorf(ctx, "failed to schedule change review task for host %s, cl %d, revision %s", host, cl.Number, cl.CurrentRevision)
			} else {
				logging.Infof(ctx, "scheduled change review task for host %s, cl %d, revision %s", host, cl.Number, cl.CurrentRevision)
			}
		}
	}

	if errNum > 0 {
		return errors.New(fmt.Sprintf("failed to schedule %d tasks", errNum))
	}
	return nil
}

func getGerritHostURL(host string) string {
	return "https://" + host + "-review.googlesource.com"
}
