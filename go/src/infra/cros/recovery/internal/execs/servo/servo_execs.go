// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package servo

import (
	"context"
	"fmt"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/log"
	"infra/cros/recovery/tlw"
)

// NOTE: That is just fake execs for local testing during developing.
// TODO(otabek@): Replace with real execs.

func servodInitActionExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	req := &tlw.InitServodRequest{
		Resource: args.DUT.Name,
		Options:  defaultServodOptions,
	}
	if err := args.Access.InitServod(ctx, req); err != nil {
		return errors.Annotate(err, "init servod").Err()
	}
	return nil
}

func servodStopActionExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	if err := args.Access.StopServod(ctx, args.DUT.Name); err != nil {
		return errors.Annotate(err, "stop servod").Err()
	}
	return nil
}

func servodRestartActionExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	if err := servodStopActionExec(ctx, args, actionArgs); err != nil {
		log.Debug(ctx, "Servod restart: fail stop servod. Error: %s", err)
	}
	if err := servodInitActionExec(ctx, args, actionArgs); err != nil {
		return errors.Annotate(err, "restart servod").Err()
	}
	return nil
}

func servoDetectUSBKey(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	res, err := ServodCallGet(ctx, args, "image_usbkey_dev")
	if err != nil {
		args.DUT.ServoHost.UsbkeyState = tlw.HardwareStateNotDetected
		return errors.Annotate(err, "servo detect usb key: could not obtain usb path on servo: %q", err).Err()
	} else if res.Value.GetString_() == "" {
		args.DUT.ServoHost.UsbkeyState = tlw.HardwareStateNotDetected
		return errors.Reason("servo detect usb key: the path to usb drive is empty").Err()
	}
	servoUsbPath := res.Value.GetString_()
	log.Debug(ctx, "Servo Detect USB-Key: USB-key path: %s.", servoUsbPath)
	r := args.Access.Run(ctx, args.DUT.ServoHost.Name, fmt.Sprintf("fdisk -l %s", servoUsbPath))
	if r.ExitCode != 0 {
		args.DUT.ServoHost.UsbkeyState = tlw.HardwareStateNotDetected
		log.Debug(ctx, "Servo Detect USB-Key: fdisk command did not succeed, exit code %q.", r.ExitCode)
		return errors.Reason("servo detect usb key: could not determine whether %q is a valid usb path", servoUsbPath).Err()
	}
	if args.DUT.ServoHost.UsbkeyState == tlw.HardwareStateNeedReplacement {
		// This device has been marked for replacement. A further
		// audit action is required to correct this.
		log.Debug(ctx, "Servo Detect USB-Key: device marked for replacement.")
	} else {
		args.DUT.ServoHost.UsbkeyState = tlw.HardwareStateNormal
	}
	return nil
}

func init() {
	execs.Register("servo_host_servod_init", servodInitActionExec)
	execs.Register("servo_host_servod_stop", servodStopActionExec)
	execs.Register("servo_host_servod_restart", servodRestartActionExec)
	execs.Register("servo_detect_usbkey", servoDetectUSBKey)
}
