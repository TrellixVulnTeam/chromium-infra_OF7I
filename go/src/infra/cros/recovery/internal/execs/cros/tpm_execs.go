// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cros

import (
	"context"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/execs"
)

const (
	// Expected value of tpm dev-signed firmware version.
	devTpmFirmwareVersion = "0x00010001"
	// Expected value of tpm dev-signed kernel version.
	devTPMKernelVersion = "0x00010001"
	// Command for checking the tpm kernel version.
	tpmKernelVersionCommand = "tpm_kernver"
	// Command for checking the tpm firmware version.
	tpmFirmwareVersionCommand = "tpm_fwver"
)

// isOnDevTPMKernelVersionExec verifies dev's tpm kernel version is match to expected value.
//
// For dev-signed firmware, tpm_kernver reported from
// crossystem should always be 0x10001. Firmware update on DUTs with
// incorrect tpm_kernver may fail due to firmware rollback protection.
func matchDevTPMKernelVersionExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	if err := matchCrosSystemValueToExpectation(ctx, args, tpmKernelVersionCommand, devTPMKernelVersion); err != nil {
		return errors.Annotate(err, "match dev tpm kernel version").Err()
	}
	return nil
}

// matchDevTPMFirmwareVersionExec verifies dev's tpm firmware version is match to expected value.
//
// For dev-signed firmware, tpm_fwver reported from
// crossystem should always be 0x10001. Firmware update on DUTs with
// incorrect tmp_fwver may fail due to firmware rollback protection.
func matchDevTPMFirmwareVersionExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	if err := matchCrosSystemValueToExpectation(ctx, args, tpmFirmwareVersionCommand, devTpmFirmwareVersion); err != nil {
		return errors.Annotate(err, "match dev tpm firmware version").Err()
	}
	return nil
}

func init() {
	execs.Register("cros_match_dev_tpm_firmware_version", matchDevTPMFirmwareVersionExec)
	execs.Register("cros_match_dev_tpm_kernel_version", matchDevTPMKernelVersionExec)
}
