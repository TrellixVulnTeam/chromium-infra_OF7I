// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package reviewer

import (
	"context"
	"fmt"
	"strings"

	"go.chromium.org/luci/common/logging"
	gerritpb "go.chromium.org/luci/common/proto/gerrit"

	"infra/appengine/rubber-stamper/config"
	"infra/appengine/rubber-stamper/internal/gerrit"
	"infra/appengine/rubber-stamper/internal/util"
	"infra/appengine/rubber-stamper/tasks/taskspb"
)

// ReviewChange reviews a CL and then either gives a Bot-Commit +1 label or
// leaves a comment explain why the CL shouldn't be passed and removes itself
// as a reviewer.
// For CLs that are not reverts/relands/cherry picks, each file that fails to
// match a BenignFilePattern will be called out in the comment the Rubber
// Stamper leaves on the CL.
// For CLs that are reverts/relands/cherry picks, and fail to pass the “clean
// revert, reland, cherry pick” set of rules, and also fail the
// BenignFilePattern checks, the rubber stamper will prepend the list of
// nonconformant files with a note that this CL is not a clean
// revert/reland/cherrypick, and provide a shortlink to more documentation.
func ReviewChange(ctx context.Context, t *taskspb.ChangeReviewTask) error {
	var msg string
	cfg, err := config.Get(ctx)
	if err != nil {
		return err
	}
	hostCfg := cfg.HostConfigs[t.Host]

	gc, err := gerrit.GetCurrentClient(ctx, t.Host+"-review.googlesource.com")
	if err != nil {
		return err
	}

	if t.RevertOf != 0 {
		// The change is a possible clean revert.
		msg, err = reviewCleanRevert(ctx, cfg, gc, t)
		if err != nil {
			return err
		}
		if msg == "" {
			return approveChange(ctx, gc, t)
		}
	}

	invalidFiles, err := reviewBenignFileChange(ctx, hostCfg, gc, t)
	if err != nil {
		return err
	}
	if len(invalidFiles) > 0 {
		// Invalid BenignFileChange.
		// TODO: Add a go link in the msg, which tells users what the config
		// looks like, what classes of CLs are safe, and how to send a CL to
		// update the config to add a fileset.
		if msg == "" {
			msg = "The change cannot be auto-reviewed. The following files do not match the benign file configuration: " +
				strings.Join(invalidFiles[:], ", ")
		}
		return declineChange(ctx, gc, t, msg)
	}

	return approveChange(ctx, gc, t)
}

// Approve a CL.
func approveChange(ctx context.Context, gc gerrit.Client, t *taskspb.ChangeReviewTask) error {
	labels := map[string]int32{"Bot-Commit": 1}
	if t.AutoSubmit {
		labels["Commit-Queue"] = 2
	}

	setReviewReq := &gerritpb.SetReviewRequest{
		Number:     t.Number,
		Labels:     labels,
		RevisionId: t.Revision,
	}
	logging.Debugf(ctx, "change (host %s, cl %d, revision %s) is valid", t.Host, t.Number, t.Revision)
	_, err := gc.SetReview(ctx, setReviewReq)
	if err != nil {
		return fmt.Errorf("failed to add label for host %s, cl %d, revision %s: %v", t.Host, t.Number, t.Revision, err.Error())
	}
	return nil
}

// Decline a CL. The msg tells why the CL shouldn't be approved.
func declineChange(ctx context.Context, gc gerrit.Client, t *taskspb.ChangeReviewTask, msg string) error {
	setReviewReq := &gerritpb.SetReviewRequest{
		Number:     t.Number,
		RevisionId: t.Revision,
		Message:    msg,
	}
	logging.Debugf(ctx, "change (host %s, cl %d, revision %s) cannot be auto-reviewed: %s", t.Host, t.Number, t.Revision, msg)
	_, err := gc.SetReview(ctx, setReviewReq)
	if err != nil {
		return fmt.Errorf("failed to leave comment for host %s, cl %d, revision %s: %v", t.Host, t.Number, t.Revision, err.Error())
	}

	sa, err := util.GetServiceAccountName(ctx)
	if err != nil {
		return err
	}
	deleteReviewerReq := &gerritpb.DeleteReviewerRequest{
		Number:    t.Number,
		AccountId: sa,
	}
	_, err = gc.DeleteReviewer(ctx, deleteReviewerReq)
	if err != nil {
		return fmt.Errorf("failed to delete reviewer for host %s, cl %d, revision %s: %v", t.Host, t.Number, t.Revision, err.Error())
	}
	return nil
}
