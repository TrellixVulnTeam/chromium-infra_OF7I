// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cros

import (
	"context"
	"strconv"
	"strings"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/log"
	"infra/cros/recovery/tlw"
)

// updateCr50LabelExec will update the DUT's Cr50Phase state into the corresponding Cr50 state.
func updateCr50LabelExec(ctx context.Context, info *execs.ExecInfo) error {
	r := info.DefaultRunner()
	// Example of the rwVersion: `0.5.40`
	rwVersion, err := GetCr50FwVersion(ctx, r, CR50RegionRW)
	if err != nil {
		info.RunArgs.DUT.Cr50Phase = tlw.Cr50PhaseUnspecified
		return errors.Annotate(err, "update cr50 label").Err()
	}
	rwVersionComponents := strings.Split(rwVersion, ".")
	if len(rwVersionComponents) < 2 {
		info.RunArgs.DUT.Cr50Phase = tlw.Cr50PhaseUnspecified
		return errors.Reason("update cr50 label: the number of version component in the rw version is incorrect.").Err()
	}
	// Check the major version to determine prePVT vs PVT.
	// Ex:
	// rwVersionComponents: ["0", "5", "40"].
	// marjoRwVersion: integer value of 5.
	majorRwVersion, err := strconv.ParseInt(rwVersionComponents[1], 10, 64)
	if err != nil {
		info.RunArgs.DUT.Cr50Phase = tlw.Cr50PhaseUnspecified
		return errors.Annotate(err, "update cr50 label").Err()
	}
	if majorRwVersion%2 != 0 {
		// PVT image has a odd major version number.
		// prePVT image has an even major version number.
		info.RunArgs.DUT.Cr50Phase = tlw.Cr50PhasePVT
		log.Info(ctx, "update DUT's Cr50 to be %s", tlw.Cr50PhasePVT)
	} else {
		info.RunArgs.DUT.Cr50Phase = tlw.Cr50PhasePREPVT
		log.Info(ctx, "update DUT's Cr50 to be %s", tlw.Cr50PhasePREPVT)
	}
	return nil
}

// updateCr50KeyIdLabelExec will update the DUT's Cr50KeyEnv state into the corresponding Cr50 key id state.
func updateCr50KeyIdLabelExec(ctx context.Context, info *execs.ExecInfo) error {
	r := info.DefaultRunner()
	roKeyIDString, err := GetCr50FwKeyID(ctx, r, CR50RegionRO)
	if err != nil {
		info.RunArgs.DUT.Cr50KeyEnv = tlw.Cr50KeyEnvUnspecified
		return errors.Annotate(err, "update cr50 key id").Err()
	}
	// Trim "," due to the remaining of the regular expression.
	// Trim "0x" due to the restriction of golang's ParseInt only taking the hex number without "0x".
	// Ex:
	// Before Trim: "0xffffff,"
	// After Trim: "ffffff"
	roKeyIDString = strings.Trim(roKeyIDString, ",0x")
	roKeyID, err := strconv.ParseInt(roKeyIDString, 16, 64)
	if err != nil {
		info.RunArgs.DUT.Cr50KeyEnv = tlw.Cr50KeyEnvUnspecified
		return errors.Annotate(err, "update cr50 key id").Err()
	}
	if roKeyID&(1<<2) != 0 {
		info.RunArgs.DUT.Cr50KeyEnv = tlw.Cr50KeyEnvProd
		log.Info(ctx, "update DUT's Cr50 Key Env to be %s", tlw.Cr50KeyEnvProd)
	} else {
		info.RunArgs.DUT.Cr50KeyEnv = tlw.Cr50KeyEnvDev
		log.Info(ctx, "update DUT's Cr50 Key Env to be %s", tlw.Cr50KeyEnvDev)
	}
	return nil
}

func init() {
	execs.Register("cros_update_cr50_label", updateCr50LabelExec)
	execs.Register("cros_update_cr50_key_id_label", updateCr50KeyIdLabelExec)
}
