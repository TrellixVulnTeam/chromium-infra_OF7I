// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cros

import (
	"context"
	"fmt"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/log"
	"infra/cros/recovery/tlw"
)

const (
	// gsCrOSImageBucket is the base URL for the Google Storage bucket for
	// ChromeOS image archives.
	gsCrOSImageBucket = "gs://chromeos-image-archive"
)

// updateProvisionedCrosVersionExec reads OS version from the DUT for provisioned info.
func updateProvisionedCrosVersionExec(ctx context.Context, info *execs.ExecInfo) error {
	version, err := releaseBuildPath(ctx, info.NewRunner(info.RunArgs.DUT.Name))
	if err != nil {
		return errors.Annotate(err, "read os version").Err()
	}
	log.Debugf(ctx, "ChromeOS version on the dut: %s.", version)
	if info.RunArgs.DUT.ProvisionedInfo == nil {
		info.RunArgs.DUT.ProvisionedInfo = &tlw.DUTProvisionedInfo{}
	}
	info.RunArgs.DUT.ProvisionedInfo.CrosVersion = version
	return nil
}

// updateJobRepoURLExec updates job repo URL for the DUT for provisoned info.
func updateJobRepoURLExec(ctx context.Context, info *execs.ExecInfo) error {
	version := info.RunArgs.DUT.ProvisionedInfo.CrosVersion
	if version == "" {
		return errors.Reason("update job repo url: provisioned version not found").Err()
	}
	gsPath := fmt.Sprintf("%s/%s", gsCrOSImageBucket, version)
	jobRepoURL, err := info.RunArgs.Access.GetCacheUrl(ctx, info.RunArgs.DUT.Name, gsPath)
	if err != nil {
		return errors.Annotate(err, "update job repo url").Err()
	}
	log.Debugf(ctx, "New job repo URL: %s.", jobRepoURL)
	info.RunArgs.DUT.ProvisionedInfo.JobRepoURL = jobRepoURL
	return nil
}

func init() {
	execs.Register("cros_update_provision_os_version", updateProvisionedCrosVersionExec)
	execs.Register("cros_update_job_repo_url", updateJobRepoURLExec)
}
