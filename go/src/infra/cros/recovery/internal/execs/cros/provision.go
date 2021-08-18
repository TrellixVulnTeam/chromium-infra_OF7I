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

// readOsVersionExec reads OS version from the DUT for provisioned info.
func readOSVersionExec(ctx context.Context, args *execs.RunArgs) error {
	version, err := releaseBuildPath(ctx, args.DUT.Name, args)
	if err != nil {
		return errors.Annotate(err, "read os version").Err()
	}
	log.Debug(ctx, "ChromeOS version on the dut: %s.", version)
	if args.DUT.ProvisionedInfo == nil {
		args.DUT.ProvisionedInfo = &tlw.DUTProvisionedInfo{}
	}
	args.DUT.ProvisionedInfo.CrosVersion = version
	return nil
}

// updateJobRepoURLExec updates job repo URL for the DUT for provisoned info.
func updateJobRepoURLExec(ctx context.Context, args *execs.RunArgs) error {
	version := args.DUT.ProvisionedInfo.CrosVersion
	if version == "" {
		return errors.Reason("update job repo url: provisioned version not found").Err()
	}
	gsPath := fmt.Sprintf("%s/%s", gsCrOSImageBucket, version)
	jobRepoURL, err := args.Access.GetCacheUrl(ctx, args.DUT.Name, gsPath)
	if err != nil {
		return errors.Annotate(err, "update job repo url").Err()
	}
	log.Debug(ctx, "New job repo URL: %s.", jobRepoURL)
	args.DUT.ProvisionedInfo.JobRepoURL = jobRepoURL
	return nil
}

func init() {
	execs.Register("cros_read_os_version", readOSVersionExec)
	execs.Register("cros_update_job_repo_url", updateJobRepoURLExec)
}
