// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package servo

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/execs/cros"
	"infra/cros/recovery/internal/execs/servo/topology"
	"infra/cros/recovery/internal/localtlw/servod"
	"infra/cros/recovery/internal/log"
	"infra/cros/recovery/tlw"
)

const (
	// Time between an usb disk plugged-in and detected in the system.
	usbDetectionDelay = 5

	// The prefix of the badblocks command for verifying USB
	// drives. The USB-drive path will be attached to it when
	// badblocks needs to be executed on a drive.
	badBlocksCommandPrefix = "badblocks -w -e 100 -b 4096 -t random %s"

	// This parameter represents the configuration for minimum number
	// of child servo devices to be verified-for.
	topologyMinChildArg = "min_child"

	// This parameter represents the configuration to control whether
	// the servo topology that read during servo topology verification
	// is persisted for use by other actions.
	persistTopologyArg = "persist_topology"

	// The default value that will be used to drive whether or not the
	// topology needs to be persisted. A value that is passed from the
	// configuration will over-ride this.
	persistTopologyDefaultValue = false

	// The default value that will be used for validating the number
	// of servo children in the servo topology. A value that is passed
	// from the configuration will over-ride this.
	topologyMinChildCountDefaultValue = 1
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

func runCheckOnHost(ctx context.Context, args *execs.RunArgs, resourceName string, usbPath string) (tlw.HardwareState, error) {
	command := fmt.Sprintf(badBlocksCommandPrefix, usbPath)
	log.Debug(ctx, "Run Check On Host: Executing %q", command)
	// The execution timeout for this audit job is configured at the
	// level of the action. So the execution of this command will be
	// bound by that.
	r := args.Access.Run(ctx, resourceName, command)
	switch r.ExitCode {
	case 0:
		// TODO(vkjoshi@): recheck if this is required, or does stderr need to be examined.
		if len(strings.TrimSpace(r.Stdout)) > 0 {
			return tlw.HardwareStateNeedReplacement, nil
		}
		return tlw.HardwareStateNormal, nil
	case 124: // timeout
		return "", errors.Reason("run check on host: could not successfully complete check, error %q", r.Stderr).Err()
	case 127: // badblocks
		return "", errors.Reason("run check on host: could not successfully complete check, error %q", r.Stderr).Err()
	default:
		return tlw.HardwareStateNeedReplacement, nil
	}
}

func servoAuditUSBKey(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	dutUsb := ""
	if cros.IsSSHable(ctx, args, args.DUT.Name) == nil {
		log.Debug(ctx, "Servo Audit USB Key: %q is reachable through SSH", args.DUT.Name)
		var err error = nil
		dutUsb, err = GetUSBDrivePathOnDut(ctx, args)
		if err != nil {
			log.Debug(ctx, "Servo Audit USB Key: could not determine USB-drive path on DUT: %q, error: %q. This is not critical. We will continue the audit by setting the path to empty string.", args.DUT.Name, err)
		}
	} else {
		log.Debug(ctx, "Servo Audit USB Key: continue audit from servo-host because DUT %q is not reachable through SSH", args.DUT.Name)
	}
	if dutUsb != "" {
		// DUT is reachable, and we found a USB drive on it.
		state, err := runCheckOnHost(ctx, args, args.DUT.Name, dutUsb)
		if err != nil {
			return errors.Reason("servo audit usb key: could not check DUT usb path %q", dutUsb).Err()
		}
		args.DUT.ServoHost.UsbkeyState = state
	} else {
		// Either the DUT is not reachable, or it does not have a USB
		// drive attached to it.

		// This statement obtains the path of usb drive on
		// servo-host. It also switches the USB drive on servo
		// multiplexer to servo-host.
		result, err := ServodCallGet(ctx, args, servod.ImageUsbkeyDev)
		if err != nil {
			// A dependency has already checked that the Servo USB is
			// available. But here we again check that no errors
			// occurred while determining USB path, in case something
			// changed between execution of dependency, and this
			// operation.
			args.DUT.ServoHost.UsbkeyState = tlw.HardwareStateNotDetected
			return errors.Annotate(err, "servo audit usb key: could not obtain usb path on servo: %q", err).Err()
		}
		servoUsbPath := strings.TrimSpace(result.Value.GetString_())
		if servoUsbPath == "" {
			args.DUT.ServoHost.UsbkeyState = tlw.HardwareStateNotDetected
			log.Debug(ctx, "Servo Audit USB Key: cannot continue audit because the path to USB-Drive is empty")
			return errors.Reason("servo audit usb key: the path to usb drive is empty").Err()
		}
		state, err := runCheckOnHost(ctx, args, args.DUT.ServoHost.Name, servoUsbPath)
		if err != nil {
			log.Debug(ctx, "Servo Audit USB Key: error %q during audit of USB-Drive", err)
			return errors.Annotate(err, "servo audit usb key: could not check usb path %q on servo-host %q", servoUsbPath, args.DUT.ServoHost.Name).Err()
		}
		args.DUT.ServoHost.UsbkeyState = state
	}
	return nil
}

// Verify that the root servo is enumerated/present on the host.
func isRootServoPresentExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	runner := args.NewRunner(args.DUT.ServoHost.Name)
	rootServo, err := topology.GetRootServo(ctx, runner, args.DUT.ServoHost.Servo.SerialNumber)
	if err != nil {
		return errors.Annotate(err, "is root servo present").Err()
	}
	if !topology.IsItemGood(ctx, rootServo) {
		log.Info(ctx, "is servo root present: no good root servo found")
		return errors.Reason("is servo root present: no good root servo found").Err()
	}
	log.Info(ctx, "is servo root present: success")
	return nil
}

// Verify that the root servo is enumerated/present on the host.
func servoTopologyUpdate(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	runner := args.NewRunner(args.DUT.ServoHost.Name)
	servoTopology, err := topology.RetrieveServoTopology(ctx, runner, args.DUT.ServoHost.Servo.SerialNumber)
	if err != nil {
		return errors.Annotate(err, "servo verify topology").Err()
	}
	if servoTopology.Root == nil {
		return errors.Reason("servo verify topology: root servo not found").Err()
	}
	argsMap := execs.ParseActionArgs(ctx, actionArgs, execs.DefaultSplitter)
	minChildCount := topologyMinChildCountDefaultValue
	persistTopology := persistTopologyDefaultValue
	for k, v := range argsMap {
		log.Debug(ctx, "Servo Topology Update: k:%q, v:%q", k, v)
		if v != "" {
			// a non-empty value string implies that the corresponding
			// action arg was parsed correctly.
			switch k {
			case topologyMinChildArg:
				// If the configuration contains any min_child parameters,
				// it will be used for validation here. If no such
				// argument is present, we will not conduct any validation
				// of number of child servo based min_child.
				minChildCount, err = strconv.Atoi(v)
				if err != nil {
					return errors.Reason("servo verify topology: malformed min child config in action arg %q:%q", k, v).Err()
				}
			case persistTopologyArg:
				persistTopology, err = strconv.ParseBool(v)
				if err != nil {
					return errors.Reason("servo verify topology: malformed update servo config in action arg %q:%q", k, v).Err()
				}
			}
		}
	}
	if len(servoTopology.Children) < minChildCount {
		return errors.Reason("servo verify topology: expected a min of %d children, found %d", minChildCount, len(servoTopology.Children)).Err()
	}
	if persistTopology {
		// This verified topology will be used in all subsequent
		// action that need the servo topology. This will avoid time
		// with re-fetching the topology.
		args.DUT.ServoHost.ServoTopology = servoTopology
	}
	return nil
}

func init() {
	execs.Register("servo_host_servod_init", servodInitActionExec)
	execs.Register("servo_host_servod_stop", servodStopActionExec)
	execs.Register("servo_host_servod_restart", servodRestartActionExec)
	execs.Register("servo_detect_usbkey", servoDetectUSBKey)
	execs.Register("servo_audit_usbkey", servoAuditUSBKey)
	execs.Register("servo_v4_root_present", isRootServoPresentExec)
	execs.Register("servo_topology_update", servoTopologyUpdate)
}
