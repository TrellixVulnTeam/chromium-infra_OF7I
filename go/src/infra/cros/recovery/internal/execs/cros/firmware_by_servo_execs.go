// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cros

import (
	"context"
	"fmt"
	"time"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/components/cros/firmware"
	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/log"
	"infra/cros/recovery/tlw"
)

func readGbbFlagsByServoExec(ctx context.Context, info *execs.ExecInfo) error {
	servod := info.NewServod()
	run := info.NewRunner(info.RunArgs.DUT.ServoHost.Name)
	req := &firmware.ReadAPInfoRequest{
		FilePath: defaultAPFilePath(info.RunArgs.DUT),
		GBBFlags: true,
	}
	res, err := firmware.ReadAPInfoByServo(ctx, req, run, servod, info.NewLogger())
	if err != nil {
		return errors.Annotate(err, "read gbb flags").Err()
	}
	log.Debugf(ctx, "Device has GBB flags: %v (%v)", res.GBBFlags, res.GBBFlagsRaw)
	am := info.GetActionArgs(ctx)
	// FORCE_DEV_SWITCH_ON 0x00000008 -> 8
	if am.AsBool(ctx, "validate_in_dev_mode", false) {
		if res.GBBFlags&8 != 8 {
			return errors.Reason("read gbb flags: device is not forced to boot to dev mode").Err()
		}
	} else {
		log.Infof(ctx, "Not expected GBB flags for dev-mode")
	}
	// FORCE_DEV_BOOT_USB 0x00000010 -> 16
	if am.AsBool(ctx, "validate_usb_boot_enabled", false) {
		if res.GBBFlags&16 != 16 {
			return errors.Reason("read gbb flags: usb boot in dev mode is not enabled").Err()
		}
	} else {
		log.Infof(ctx, "Not expected GBB flags for usb boot")
	}
	if am.AsBool(ctx, "remove_file", true) {
		log.Debugf(ctx, "Remove AP image from host")
		if _, err := run(ctx, 30*time.Second, "rm", "-f", req.FilePath); err != nil {
			return errors.Annotate(err, "set gbb flags").Err()
		}
	}
	return nil
}

func checkIfApHasDevSignedImageExec(ctx context.Context, info *execs.ExecInfo) error {
	servod := info.NewServod()
	run := info.NewRunner(info.RunArgs.DUT.ServoHost.Name)
	req := &firmware.ReadAPInfoRequest{
		FilePath: defaultAPFilePath(info.RunArgs.DUT),
		Keys:     true,
	}
	res, err := firmware.ReadAPInfoByServo(ctx, req, run, servod, info.NewLogger())
	if err != nil {
		return errors.Annotate(err, "ap dev signed").Err()
	}
	log.Debugf(ctx, "Device has keys: %v", res.Keys)
	if firmware.IsDevKeys(res.Keys, info.NewLogger()) {
		return nil
	}
	return errors.Reason("ap dev signed: device is not dev signed").Err()
}

// Please be sure that.
func removeAPFileFromServoHostExec(ctx context.Context, info *execs.ExecInfo) error {
	run := info.NewRunner(info.RunArgs.DUT.ServoHost.Name)
	p := defaultAPFilePath(info.RunArgs.DUT)
	if _, err := run(ctx, 30*time.Second, "rm", "-f", p); err != nil {
		// Do not fail if we cannot remove the file.
		log.Infof(ctx, "Fail to remove AP file %q from servo-host: %s", p, err)
	}
	return nil
}

func setGbbFlagsByServoExec(ctx context.Context, info *execs.ExecInfo) error {
	am := info.GetActionArgs(ctx)
	req := &firmware.SetApInfoByServoRequest{
		FilePath: defaultAPFilePath(info.RunArgs.DUT),
		// Set gbb flags to 0x18 to force dev boot and enable boot from USB.
		GBBFlags:       am.AsString(ctx, "gbb_flags", ""),
		UpdateGBBFlags: true,
	}
	servod := info.NewServod()
	run := info.NewRunner(info.RunArgs.DUT.ServoHost.Name)
	if err := firmware.SetApInfoByServo(ctx, req, run, servod, info.NewLogger()); err != nil {
		return errors.Annotate(err, "set gbb flags").Err()
	}
	if am.AsBool(ctx, "remove_file", true) {
		log.Debugf(ctx, "Remove AP image from host")
		if _, err := run(ctx, 30*time.Second, "rm", "-f", req.FilePath); err != nil {
			// Do not fail if we cannot remove the file.
			log.Infof(ctx, "Fail to remove AP file %q from servo-host: %s", req.FilePath, err)
		}
	}
	if !am.AsBool(ctx, "prevent_reboot", false) {
		if err := servod.Set(ctx, "power_state", "reset"); err != nil {
			return errors.Annotate(err, "set gbb flags").Err()
		}
	}
	return nil
}

func updateFwWithFwImageByServo(ctx context.Context, info *execs.ExecInfo) error {
	const (
		firmwareTarName = "firmware_from_source.tar.bz2"
	)
	sv, err := info.Versioner().Cros(ctx, info.RunArgs.DUT.Name)
	if err != nil {
		return errors.Annotate(err, "cros provision").Err()
	}
	mn := "update fw with fw-image by servo"
	am := info.GetActionArgs(ctx)
	imageName := am.AsString(ctx, "version_name", sv.FwImage)
	log.Debugf(ctx, "Used fw image name: %s", imageName)
	gsBucket := am.AsString(ctx, "gs_bucket", gsCrOSImageBucket)
	log.Debugf(ctx, "Used gs bucket name: %s", gsBucket)
	gsImagePath := am.AsString(ctx, "gs_image_path", fmt.Sprintf("%s/%s", gsBucket, imageName))
	log.Debugf(ctx, "Used fw image path: %s", gsImagePath)
	fwDownloadDir := am.AsString(ctx, "fw_download_dir", defaultFwFolderPath(info.RunArgs.DUT))
	log.Debugf(ctx, "Used fw image path: %s", gsImagePath)
	// Requesting convert GC path to caches service path.
	// Example: `http://Addr:8082/download/chromeos-image-archive/board-firmware/R99-XXXXX.XX.0`
	downloadPath, err := info.RunArgs.Access.GetCacheUrl(ctx, info.RunArgs.DUT.Name, gsImagePath)
	if err != nil {
		return errors.Annotate(err, mn).Err()
	}
	fwFileName := am.AsString(ctx, "fw_filename", firmwareTarName)
	downloadFilename := fmt.Sprintf("%s/%s", downloadPath, fwFileName)
	req := &firmware.InstallFwFromFwImageRequest{
		DownloadImagePath:    downloadFilename,
		DownloadImageTimeout: am.AsDuration(ctx, "download_timeout", 600, time.Second),
		DownloadDir:          fwDownloadDir,
		Board:                am.AsString(ctx, "dut_board", info.RunArgs.DUT.Board),
		Model:                am.AsString(ctx, "dut_model", info.RunArgs.DUT.Model),
		UpdateEC:             am.AsBool(ctx, "update_ec", false),
		UpdateAP:             am.AsBool(ctx, "update_ap", false),
	}
	servod := info.NewServod()
	run := info.NewRunner(info.RunArgs.DUT.ServoHost.Name)
	err = firmware.InstallFwFromFwImage(ctx, req, run, servod, info.NewLogger())
	return errors.Annotate(err, mn).Err()
}

// DefaultAPFilePath provides default path to AP file.
// Path used to minimize cycle to read AP from the DUT and other operation over it.
func defaultAPFilePath(d *tlw.Dut) string {
	return fmt.Sprintf("/tmp/bios_%v.bin", d.Name)
}

// defaultFwFolderPath provides default path to directory used for firmware extraction.
func defaultFwFolderPath(d *tlw.Dut) string {
	return fmt.Sprintf("/tmp/fw_%v", d.Name)
}

func init() {
	execs.Register("cros_read_gbb_by_servo", readGbbFlagsByServoExec)
	execs.Register("cros_ap_is_dev_signed_by_servo", checkIfApHasDevSignedImageExec)
	execs.Register("cros_set_gbb_by_servo", setGbbFlagsByServoExec)
	execs.Register("cros_remove_default_ap_file_servo_host", removeAPFileFromServoHostExec)
	execs.Register("cros_update_fw_with_fw_image_by_servo_from", updateFwWithFwImageByServo)
}
