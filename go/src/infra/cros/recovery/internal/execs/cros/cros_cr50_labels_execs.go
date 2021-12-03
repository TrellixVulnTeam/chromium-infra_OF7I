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
func updateCr50LabelExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	r := args.NewRunner(args.ResourceName)
	// Example of the rwVersion: `0.5.40`
	rwVersion, err := GetCr50FwVersion(ctx, r, CR50RegionRW)
	if err != nil {
		args.DUT.Cr50Phase = tlw.Cr50PhaseUnspecified
		return errors.Annotate(err, "update cr50 label").Err()
	}
	rwVersionComponents := strings.Split(rwVersion, ".")
	if len(rwVersionComponents) < 2 {
		args.DUT.Cr50Phase = tlw.Cr50PhaseUnspecified
		return errors.Reason("update cr50 label: the number of version component in the rw version is incorrect.").Err()
	}
	// Check the major version to determine prePVT vs PVT.
	// Ex:
	// rwVersionComponents: ["0", "5", "40"].
	// marjoRwVersion: integer value of 5.
	majorRwVersion, err := strconv.ParseInt(rwVersionComponents[1], 10, 64)
	if err != nil {
		args.DUT.Cr50Phase = tlw.Cr50PhaseUnspecified
		return errors.Annotate(err, "update cr50 label").Err()
	}
	if majorRwVersion%2 != 0 {
		// PVT image has a odd major version number.
		// prePVT image has an even major version number.
		args.DUT.Cr50Phase = tlw.Cr50PhasePVT
		log.Info(ctx, "update DUT's Cr50 to be %s", tlw.Cr50PhasePVT)
	} else {
		args.DUT.Cr50Phase = tlw.Cr50PhasePREPVT
		log.Info(ctx, "update DUT's Cr50 to be %s", tlw.Cr50PhasePREPVT)
	}
	return nil
}

func init() {
	execs.Register("cros_update_cr50_label", updateCr50LabelExec)
}
