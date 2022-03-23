// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cros

import (
	"context"
	"time"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/log"
	"infra/cros/recovery/tlw"
)

const (
	// moSysSkuCmd will retrieve the SKU label of the DUT.
	moSysSkuCmd = "mosys platform sku"
)

// updateDeviceSKUExec updates device's SKU label if not present in inventory
// or keep it the same if the info.RunArgs.DUT already has the value for SKU label.
func updateDeviceSKUExec(ctx context.Context, info *execs.ExecInfo) error {
	r := info.DefaultRunner()
	skuLabelOutput, err := r(ctx, time.Minute, moSysSkuCmd)
	if err != nil {
		log.Debugf(ctx, "Device sku label not found in the DUT.")
		return errors.Annotate(err, "update device sku label").Err()
	}
	info.RunArgs.DUT.DeviceSku = skuLabelOutput
	return nil
}

// isAudioLoopBackStateWorkingExec checks if the DUT's audio loop back state has already been in the working state.
func isAudioLoopBackStateWorkingExec(ctx context.Context, info *execs.ExecInfo) error {
	if info.RunArgs.DUT.Audio.GetLoopbackState() != tlw.DUTAudio_LOOPBACK_WORKING {
		return errors.Reason("audio loop back state working: not working").Err()
	}
	return nil
}

// updateAudioLoopbackLabelExec updates the DUT's audio loop back state to the correct state
// based on the condition as follows:
// if both the Headphone and Mic exists on the DUT, then the state is working.
// For all other cases, state is unspecified.
func updateAudioLoopbackLabelExec(ctx context.Context, info *execs.ExecInfo) error {
	info.RunArgs.DUT.Audio.LoopbackState = tlw.DUTAudio_LOOPBACK_UNSPECIFIED
	defer log.Infof(ctx, "Setting DUT's Audio Loopback State to be %s", info.RunArgs.DUT.Audio.GetLoopbackState())
	r := info.DefaultRunner()
	// check if the Headphone cras audio type exists on the DUT.
	isAudioHeadPhoneExist, err := CrasAudioNodeTypeIsPlugged(ctx, r, CrasAudioHeadphone)
	if err != nil {
		return errors.Annotate(err, "update audio loop back label").Err()
	}
	// check if the Mic cras audio type exists on the DUT.
	isAudioMicExist, err := CrasAudioNodeTypeIsPlugged(ctx, r, CrasAudioMic)
	if err != nil {
		return errors.Annotate(err, "update audio loop back label").Err()
	}
	if isAudioHeadPhoneExist && isAudioMicExist {
		info.RunArgs.DUT.Audio.LoopbackState = tlw.DUTAudio_LOOPBACK_WORKING
	}
	return nil
}

func init() {
	execs.Register("cros_update_device_sku", updateDeviceSKUExec)
	execs.Register("cros_is_audio_loopback_state_working", isAudioLoopBackStateWorkingExec)
	execs.Register("cros_update_audio_loopback_state_label", updateAudioLoopbackLabelExec)
}
