// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cros

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/components/cros/firmware"
	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/log"
)

func readGbbFlagsByServoExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	servod := args.NewServod()
	run := args.NewRunner(args.DUT.ServoHost.Name)
	req := &firmware.ReadAPInfoRequest{
		FilePath: defaultAPFilePath(args),
		GBBFlags: true,
	}
	res, err := firmware.ReadAPInfoByServo(ctx, req, run, servod, args.Logger)
	if err != nil {
		return errors.Annotate(err, "read gbb flags").Err()
	}
	log.Debug(ctx, "Device has GBB flags: %v (%v)", res.GBBFlags, res.GBBFlagsRaw)
	am := execs.ParseActionArgs(ctx, actionArgs, execs.DefaultSplitter)
	// FORCE_DEV_SWITCH_ON 0x00000008 -> 8
	if am.AsBool(ctx, "in_dev_mode", true) {
		if res.GBBFlags&8 != 8 {
			return errors.Reason("read gbb flags: device is not forced to boot to dev mode").Err()
		}
	} else {
		log.Info(ctx, "Not expected GBB flags for dev-mode")
	}
	// FORCE_DEV_BOOT_USB 0x00000010 -> 16
	if am.AsBool(ctx, "usb_boot_enabled", true) {
		if res.GBBFlags&16 != 16 {
			return errors.Reason("read gbb flags: usb boot in dev mode is not enabled").Err()
		}
	} else {
		log.Info(ctx, "Not expected GBB flags for usb boot")
	}
	if am.AsBool(ctx, "remove_file", true) {
		log.Debug(ctx, "Remove AP image from host")
		if _, err := run(ctx, 30*time.Second, "rm", "-f", req.FilePath); err != nil {
			return errors.Annotate(err, "set gbb flags").Err()
		}
	}
	return nil
}

const (
	devSignedFirmwareKeyPrefix = "b11d"
)

func checkIfApHasDevSignedImageExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	servod := args.NewServod()
	run := args.NewRunner(args.DUT.ServoHost.Name)
	req := &firmware.ReadAPInfoRequest{
		FilePath: defaultAPFilePath(args),
		Keys:     true,
	}
	res, err := firmware.ReadAPInfoByServo(ctx, req, run, servod, args.Logger)
	if err != nil {
		return errors.Annotate(err, "ap dev signed").Err()
	}
	log.Debug(ctx, "Device has keys: %v", res.Keys)
	for _, key := range res.Keys {
		if strings.HasPrefix(key, devSignedFirmwareKeyPrefix) {
			log.Debug(ctx, "Device has dev signed key: %q !", key)
			return nil
		}
	}
	return errors.Reason("ap dev signed: device is not dev signed").Err()
}

// Please be sure that.
func removeAPFileFromServoHostExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	run := args.NewRunner(args.DUT.ServoHost.Name)
	p := defaultAPFilePath(args)
	if _, err := run(ctx, 30*time.Second, "rm", "-f", p); err != nil {
		// Do not fail if we cannot remove the file.
		log.Info(ctx, "Fail to remove AP file %q from servo-host: %s", p, err)
	}
	return nil
}

func setGbbFlagsByServoExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	am := execs.ParseActionArgs(ctx, actionArgs, execs.DefaultSplitter)
	req := &firmware.SetApInfoByServoRequest{
		FilePath: defaultAPFilePath(args),
		// Set gbb flags to 0x18 to force dev boot and enable boot from USB.
		GBBFlags:       am.AsString(ctx, "gbb_flags", ""),
		UpdateGBBFlags: true,
	}
	servod := args.NewServod()
	run := args.NewRunner(args.DUT.ServoHost.Name)
	if err := firmware.SetApInfoByServo(ctx, req, run, servod, args.Logger); err != nil {
		return errors.Annotate(err, "set gbb flags").Err()
	}
	if am.AsBool(ctx, "remove_file", true) {
		log.Debug(ctx, "Remove AP image from host")
		if _, err := run(ctx, 30*time.Second, "rm", "-f", req.FilePath); err != nil {
			// Do not fail if we cannot remove the file.
			log.Info(ctx, "Fail to remove AP file %q from servo-host: %s", req.FilePath, err)
		}
	}
	if !am.AsBool(ctx, "prevent_reboot", false) {
		if err := servod.Set(ctx, "power_state", "reset"); err != nil {
			return errors.Annotate(err, "set gbb flags").Err()
		}
	}
	return nil
}

// DefaultAPFilePath provides default path to AP file.
// Path used to minimize cycle to read AP from the DUT and other operation over it.
func defaultAPFilePath(args *execs.RunArgs) string {
	return fmt.Sprintf("/tmp/bios_%v.bin", args.DUT.Name)
}

func init() {
	execs.Register("cros_read_gbb_by_servo", readGbbFlagsByServoExec)
	execs.Register("cros_ap_is_dev_signed_by_servo", checkIfApHasDevSignedImageExec)
	execs.Register("cros_set_gbb_by_servo", setGbbFlagsByServoExec)
	execs.Register("cros_remove_default_ap_file_servo_host", removeAPFileFromServoHostExec)
}
