// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cros

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/log"
	"infra/cros/recovery/logger/metrics"
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
		log.Infof(ctx, "update DUT's Cr50 to be %s", tlw.Cr50PhasePVT)
	} else {
		info.RunArgs.DUT.Cr50Phase = tlw.Cr50PhasePREPVT
		log.Infof(ctx, "update DUT's Cr50 to be %s", tlw.Cr50PhasePREPVT)
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
		log.Infof(ctx, "update DUT's Cr50 Key Env to be %s", tlw.Cr50KeyEnvProd)
	} else {
		info.RunArgs.DUT.Cr50KeyEnv = tlw.Cr50KeyEnvDev
		log.Infof(ctx, "update DUT's Cr50 Key Env to be %s", tlw.Cr50KeyEnvDev)
	}
	return nil
}

// reflashCr50FwExec reflashes CR50 firmware and reboot AP from DUT side to wake it up.
//
// @params: actionArgs should be in the format of:
// Ex: ["flash_timeout:x", "wait_timeout:x"]
func reflashCr50FwExec(ctx context.Context, info *execs.ExecInfo) error {
	argsMap := info.GetActionArgs(ctx)
	// Timeout for executing the cr50 fw flash command on the DUT. Default to be 120s.
	flashTimeout := argsMap.AsDuration(ctx, "flash_timeout", 120, time.Second)
	// Delay to wait for the fw flash command to be efftive. Default to be 30s.
	waitTimeout := argsMap.AsDuration(ctx, "wait_timeout", 30, time.Second)
	// Command to update cr50 firmware with post-reset and reboot the DUT.
	updateCmd := `gsctool -ap /opt/google/cr50/firmware/cr50.bin.%s`
	if info.RunArgs.DUT.Cr50Phase == tlw.Cr50PhasePREPVT {
		updateCmd = fmt.Sprintf(updateCmd, "prepvt")
	} else {
		updateCmd = fmt.Sprintf(updateCmd, "prod")
	}
	karteAction := &metrics.Action{
		// TODO(@gregorynisbet): When karte' Search API is capable of taking in asset tag,
		// change the query to use asset tag instead of using hostname.
		Hostname:   info.RunArgs.DUT.Name,
		ActionKind: metrics.Cr50FwReflashKind,
		StartTime:  time.Now(),
		Status:     metrics.ActionStatusFail,
	}
	if mErr := info.RunArgs.Metrics.Create(ctx, karteAction); mErr != nil {
		log.Debugf(ctx, "Reflash cr50 firmware: cannot create karte metrics: %s", mErr)
	}
	defer func() {
		// Recoding cr 50 fw reflash to Karte.
		log.Debugf(ctx, "Updating cr 50 fw reflash record in Karte.")
		karteAction.StopTime = time.Now()
		if mErr := info.RunArgs.Metrics.Update(ctx, karteAction); mErr != nil {
			log.Debugf(ctx, "Reflash cr 50 fw: Metrics error: %s", mErr)
		}
	}()
	run := info.NewRunner(info.RunArgs.DUT.Name)
	// For "gsctool", we use the traditional runner because the exit code of both 0 and 1
	// indicates successful execution of the command.
	//
	// r.ExitCode == 0: All up to date, no update needed.
	// r.ExitCode == 1: Update completed, reboot required (errors includes GsctoolRequireRebootError tag).
	_, err := run(ctx, flashTimeout, updateCmd)
	if err != nil {
		errorCode, ok := errors.TagValueIn(execs.ErrCodeTag, err)
		if !ok {
			return errors.Annotate(err, "reflash cr50 fw: cannot find error code").Err()
		}
		if errorCode != 1 {
			return errors.Annotate(err, "reflash cr50 fw: fail to flash %q", info.RunArgs.DUT.Cr50Phase).Err()
		}
	}
	log.Debugf(ctx, "cr50 fw update successfully.")
	// reboot the DUT for the reflash of the cr50 fw to be effective.
	if out, err := run(ctx, 30*time.Second, "reboot && exit"); err != nil {
		// Client closed connected as rebooting.
		log.Debugf(ctx, "Client exit as device rebooted: %s", err)
		karteAction.FailReason = fmt.Sprintf("%s : reflash cr50 fw", err)
		return errors.Annotate(err, "reflash cr50 fw").Err()
	} else {
		log.Debugf(ctx, "Stdout: %s", out)
	}
	karteAction.Status = metrics.ActionStatusSuccess
	log.Debugf(ctx, "waiting for %d seconds to let cr50 fw reflash be effective.", waitTimeout)
	time.Sleep(waitTimeout)
	return nil
}

func init() {
	execs.Register("cros_update_cr50_label", updateCr50LabelExec)
	execs.Register("cros_update_cr50_key_id_label", updateCr50KeyIdLabelExec)
	execs.Register("cros_reflash_cr50_fw", reflashCr50FwExec)
}
