// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package servo

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/localtlw/servod"
	"infra/cros/recovery/internal/log"
)

// GetUSBDrivePathOnDut finds and returns the path of USB drive on a DUT.
func GetUSBDrivePathOnDut(ctx context.Context, args *execs.RunArgs) (string, error) {
	// switch USB on servo multiplexer to the DUT-side
	if _, err := ServodCallSet(ctx, args, servod.ImageUsbkeyDirection, servod.ImageUsbkeyTowardsDUT); err != nil {
		return "", errors.Annotate(err, "get usb drive path on dut: could not switch USB to DUT").Err()
	}
	// A detection delay is required when attaching this USB drive to DUT
	time.Sleep(usbDetectionDelay * time.Second)
	r := args.Access.Run(ctx, args.DUT.Name, "ls /dev/sd[a-z]")
	if r.ExitCode == 0 {
		for _, p := range strings.Split(strings.TrimSpace(r.Stdout), "\n") {
			cmd := fmt.Sprintf(". /usr/share/misc/chromeos-common.sh; get_device_type %s", p)
			newResult := args.Access.Run(ctx, args.DUT.Name, cmd)
			if newResult.ExitCode != 0 {
				return "", errors.Reason("get usb drive path on dut: could not check %q", p).Err()
			}
			if strings.TrimSpace(newResult.Stdout) == "USB" {
				fdiskResult := args.Access.Run(ctx, args.DUT.Name, fmt.Sprintf("fdisk -l %s", p))
				if fdiskResult.ExitCode == 0 {
					return p, nil
				} else {
					log.Debug(ctx, "Get USB-drive path on dut: checked candidate usb drive path %q and found it incorrect.", p)
				}
			}
		}
		log.Debug(ctx, "Get USB-drive path on dut: did not find any valid USB drive path on the DUT.")
	}
	return "", errors.Reason("get usb drive path on dut: did not find any USB Drive connected to the DUT as we checked that DUT is up").Err()
}
