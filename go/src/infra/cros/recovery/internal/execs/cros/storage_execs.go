// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cros

import (
	"context"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/execs/cros/storage"
	"infra/cros/recovery/internal/log"
	"infra/cros/recovery/tlw"
)

const (
	readStorageInfoCMD = ". /usr/share/misc/storage-info-common.sh; get_storage_info"
)

// storageStateMap maps state from storageState type to tlw.HardwareState type
var storageStateMap = map[storage.StorageState]tlw.HardwareState{
	storage.StorageStateNormal:    tlw.HardwareStateNormal,
	storage.StorageStateWarning:   tlw.HardwareStateAcceptable,
	storage.StorageStateCritical:  tlw.HardwareStateNeedReplacement,
	storage.StorageStateUndefined: tlw.HardwareStateUnspecified,
}

// auditStorageSMARTExec confirms that it is able to audi smartStorage info and mark the dut if it needs replacement.
func auditStorageSMARTExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	r := args.NewRunner(args.ResourceName)
	rawOutput, err := r(ctx, readStorageInfoCMD)
	if err != nil {
		return errors.Annotate(err, "audit storage smart").Err()
	}
	ss, err := storage.ParseSMARTInfo(ctx, rawOutput)
	if err != nil {
		return errors.Annotate(err, "audit storage smart").Err()
	}
	log.Debug(ctx, "Detected storage type: %q", ss.StorageType)
	log.Debug(ctx, "Detected storage state: %q", ss.StorageState)
	convertedHardwareState, ok := storageStateMap[ss.StorageState]
	if !ok {
		return errors.Reason("audit storage smart: cannot find corresponding hardware state match in the map").Err()
	}
	if convertedHardwareState == tlw.HardwareStateUnspecified {
		return errors.Reason("audit storage smart: DUT storage did not detected or state cannot extracted").Err()
	}
	if convertedHardwareState == tlw.HardwareStateNeedReplacement {
		log.Debug(ctx, "Detected issue with storage on the DUT")
		args.DUT.Storage.State = tlw.HardwareStateNeedReplacement
		return errors.Reason("audit storage smart: hardware state need replacement").Err()
	}
	return nil
}

func init() {
	execs.Register("cros_audit_storage_smart", auditStorageSMARTExec)
}
