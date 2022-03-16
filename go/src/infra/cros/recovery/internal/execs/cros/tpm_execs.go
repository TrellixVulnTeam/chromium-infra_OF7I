// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cros

import (
	"context"
	"time"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/components/cros"
	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/log"
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
func matchDevTPMKernelVersionExec(ctx context.Context, info *execs.ExecInfo) error {
	if err := cros.MatchCrossystemValueToExpectation(ctx, info.DefaultRunner(), tpmKernelVersionCommand, devTPMKernelVersion); err != nil {
		return errors.Annotate(err, "match dev tpm kernel version").Err()
	}
	return nil
}

// matchDevTPMFirmwareVersionExec verifies dev's tpm firmware version is match to expected value.
//
// For dev-signed firmware, tpm_fwver reported from
// crossystem should always be 0x10001. Firmware update on DUTs with
// incorrect tmp_fwver may fail due to firmware rollback protection.
func matchDevTPMFirmwareVersionExec(ctx context.Context, info *execs.ExecInfo) error {
	if err := cros.MatchCrossystemValueToExpectation(ctx, info.DefaultRunner(), tpmFirmwareVersionCommand, devTpmFirmwareVersion); err != nil {
		return errors.Annotate(err, "match dev tpm firmware version").Err()
	}
	return nil
}

// isTPMPresentExec confirms that the given DUT's TPM is present.
func isTPMPresentExec(ctx context.Context, info *execs.ExecInfo) error {
	r := info.DefaultRunner()
	rawOutput, err := r(ctx, time.Minute, "cryptohome --action=status")
	if err != nil {
		return errors.Annotate(err, "tpm present").Err()
	}
	_, readErr := ReadCryptoHomeStatusInfo(ctx, rawOutput)
	return errors.Annotate(readErr, "tpm present: cannot read crypto home status info").Err()
}

// isTPMInGoodStatusExec confirms that the given DUT's TPM is in good state.
func isTPMInGoodStatusExec(ctx context.Context, info *execs.ExecInfo) error {
	r := info.DefaultRunner()
	rawOutput, err := r(ctx, time.Minute, "cryptohome --action=status")
	if err != nil {
		return errors.Annotate(err, "tpm in good status").Err()
	}
	cryptoHS, err := ReadCryptoHomeStatusInfo(ctx, rawOutput)
	if err != nil {
		return errors.Annotate(err, "tpm in good status").Err()
	}
	enabled, enabledOk := cryptoHS.ReadTPMBool("enabled")
	if !enabledOk {
		log.Error(ctx, `Cannot read field "enabled"`)
		log.Error(ctx, `Cannot determine cryptohome valid status, skipping check`)
		return nil
	}
	if !enabled {
		log.Error(ctx, "TPM status: hardware is not working.")
		return errors.Reason("tpm in good status: tpm is not enabled").Err()
	}
	canConnect, canConnectOk := cryptoHS.ReadTPMBool("can_connect")
	lastErrorValue, lastErrorValueOk := cryptoHS.ReadTPMFloat64("last_error")
	if !canConnectOk {
		log.Error(ctx, `Cannot read field "can_connect"`)
		log.Error(ctx, `Cannot determine cryptohome valid status, skipping check`)
		return nil
	}
	if !lastErrorValueOk {
		log.Error(ctx, `Cannot read field "last_error"`)
		log.Error(ctx, `Cannot determine cryptohome valid status, skipping check`)
		return nil
	}
	if !canConnect {
		return errors.Reason("tpm in good status: tpm connect failed -- last_error=%v", lastErrorValue).Err()
	}
	owned, ownedOk := cryptoHS.ReadTPMBool("owned")
	canLoadSrk, canLoadSrkOk := cryptoHS.ReadTPMBool("can_load_srk")
	if !ownedOk {
		log.Error(ctx, `Cannot read field value:"owned"`)
		log.Error(ctx, `Cannot determine cryptohome valid status, skipping check`)
		return nil
	}
	if !canLoadSrkOk {
		log.Error(ctx, `Cannot read field value:"can_load_srk"`)
		log.Error(ctx, `Cannot determine cryptohome valid status, skipping check`)
		return nil
	}
	if owned && !canLoadSrk {
		return errors.Reason("tpm in good status: cannot load the tpm srk").Err()
	}
	canLoadSrkPk, canLoadSrkPkOk := cryptoHS.ReadTPMBool("can_load_srk_pubkey")
	if !canLoadSrkPkOk {
		log.Error(ctx, `Cannot read field value:"can_load_srk_pubkey"`)
		log.Error(ctx, `Cannot determine cryptohome valid status, skipping check`)
		return nil
	}
	if canLoadSrk && !canLoadSrkPk {
		return errors.Reason("tpm in good status: cannot load the tpm srk public key").Err()
	}
	return nil
}

func init() {
	execs.Register("cros_match_dev_tpm_firmware_version", matchDevTPMFirmwareVersionExec)
	execs.Register("cros_match_dev_tpm_kernel_version", matchDevTPMKernelVersionExec)
	execs.Register("cros_is_tpm_present", isTPMPresentExec)
	execs.Register("cros_is_tpm_in_good_status", isTPMInGoodStatusExec)
}
