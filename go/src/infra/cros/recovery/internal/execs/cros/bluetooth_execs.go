// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cros

import (
	"context"
	"reflect"
	"strings"
	"time"

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/log"
	"infra/cros/recovery/tlw"

	"go.chromium.org/luci/common/errors"
)

const (
	// Command to check whether the bluetooth device is powered-on and
	// responsive on system DBus. In case of successful Bluetooth
	// detection, the exit code will be 0 (success) and output string
	// will approximately be like '\s*variant\s+boolean\s+true'. In
	// case of failure, the output will either include 'false' instead
	// of 'true', or the exist code will be non-zero, and output will
	// be empty.
	bluetoothDetectionCmd = `dbus-send --print-reply ` +
		`--system --dest=org.bluez /org/bluez/hci0 ` +
		`org.freedesktop.DBus.Properties.Get ` +
		`string:"org.bluez.Adapter1" string:"Powered"`
)

// auditBluetoothExec will verify bluetooth on the host is detected correctly.
//
// Check if bluetooth on the host has been powered-on and is responding.
func auditBluetoothExec(ctx context.Context, info *execs.ExecInfo) error {
	r := info.DefaultRunner()
	output, err := r(ctx, time.Minute, bluetoothDetectionCmd)
	if err == nil {
		// dbus-send command completed with success
		// example output:
		// 		method return time=1635461296.023563 sender=:1.65 -> destination=:1.276 serial=65 reply_serial=2
		// 		variant       boolean true
		lines := strings.Split(output, "\n")
		if len(lines) == 2 {
			btInfoArray := strings.Fields(lines[1])
			if reflect.DeepEqual(btInfoArray, []string{"variant", "boolean", "true"}) {
				info.RunArgs.DUT.Bluetooth.State = tlw.HardwareStateNormal
				log.Infof(ctx, "set bluetooth state to be: %s", tlw.HardwareStateNormal)
				return nil
			}
		}
	}
	if execs.SSHErrorInternal.In(err) || execs.SSHErrorCLINotFound.In(err) {
		info.RunArgs.DUT.Bluetooth.State = tlw.HardwareStateUnspecified
		log.Infof(ctx, "set bluetooth state to be: %s", tlw.HardwareStateUnspecified)
		return errors.Annotate(err, "audit bluetooth").Err()
	}
	if info.RunArgs.DUT.Bluetooth.Expected {
		// If bluetooth is not detected, but was expected by setup info
		// then we set needs_replacement as it is probably a hardware issue.
		info.RunArgs.DUT.Bluetooth.State = tlw.HardwareStateNeedReplacement
		log.Infof(ctx, "set bluetooth state to be: %s", tlw.HardwareStateNeedReplacement)
		return errors.Annotate(err, "audit bluetooth").Err()
	}
	// the bluetooth state cannot be determined due to cmd failed
	// therefore, set it to HardwareStateNotDetected.
	info.RunArgs.DUT.Bluetooth.State = tlw.HardwareStateNotDetected
	log.Infof(ctx, "set bluetooth state to be: %s", tlw.HardwareStateNotDetected)
	return errors.Annotate(err, "audit bluetooth").Err()
}

func init() {
	execs.Register("cros_audit_bluetooth", auditBluetoothExec)
}
