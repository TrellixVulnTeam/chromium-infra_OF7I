// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cros

import (
	"context"
	"strings"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/log"
)

const (
	// Postfix -cheetsth to distinguish ChromeOS build during Cheets provisioning.
	cheets_suffix = "-cheetsth"
)

// matchCrosVersionToInvExec matches the cros-version match version on the DUT.
func matchCrosVersionToInvExec(ctx context.Context, info *execs.ExecInfo) error {
	osFromDUT, err := releaseBuildPath(ctx, info.DefaultRunner())
	if err != nil {
		return errors.Annotate(err, "match cros version to inventory").Err()
	}
	osFromInv := info.RunArgs.DUT.ProvisionedInfo.CrosVersion
	if osFromInv == "" {
		log.Infof(ctx, "No exsiting chromeos version label detected")
		return nil
	}
	// known cases where the version label will not match the original
	// CHROMEOS_RELEASE_BUILDER_PATH setting:
	// * Tests for the `arc-presubmit` append "-cheetsth" to the label.
	if strings.HasSuffix(osFromInv, cheets_suffix) {
		log.Debugf(ctx, "chromeos label with %s suffix detected, this suffix will be ignored when comparing label.", cheets_suffix)
		endingIndex := len(osFromInv) - len(cheets_suffix)
		osFromInv = osFromInv[:endingIndex]
	}
	log.Debugf(ctx, "OS version from DUT: %s; OS version cached in label: %s", osFromDUT, osFromInv)
	if osFromDUT != osFromInv {
		return errors.Reason("match cros version to inventory: no match").Err()
	}
	return nil
}

// matchJobRepoURLVersionToInvExec confirms the label/inventory's job_repo_url field contains cros-version on the DUT.
// if job_repo url is empty, then skipping this check.
func matchJobRepoURLVersionToInvExec(ctx context.Context, info *execs.ExecInfo) error {
	jobRepoUrlFromInv := info.RunArgs.DUT.ProvisionedInfo.JobRepoURL
	if jobRepoUrlFromInv == "" {
		log.Infof(ctx, "job repo url is empty, skipping check")
		return nil
	}
	osFromDUT, err := releaseBuildPath(ctx, info.DefaultRunner())
	if err != nil {
		return errors.Annotate(err, "match cros version to inventory").Err()
	}
	if !strings.Contains(jobRepoUrlFromInv, osFromDUT) {
		return errors.Reason("match job repo url version to inventory: chromeos image on the DUT does not match to job_repo_url from label").Err()
	}
	return nil
}

func init() {
	execs.Register("cros_match_cros_version_to_inventory", matchCrosVersionToInvExec)
	execs.Register("cros_match_job_repo_url_version_to_inventory", matchJobRepoURLVersionToInvExec)
}
