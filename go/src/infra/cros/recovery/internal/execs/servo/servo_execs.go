// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package servo

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/execs/cros"
	"infra/cros/recovery/internal/execs/cros/battery"
	"infra/cros/recovery/internal/execs/servo/topology"
	"infra/cros/recovery/internal/localtlw/servod"
	"infra/cros/recovery/internal/log"
	"infra/cros/recovery/internal/retry"
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

	// This command, when executed from servo host, checks whether the
	// servod process is responsive.
	servodHostCheckupCmd = "dut-control -p %d serialname"

	// This is the threshold voltage value, and actual values lower
	// than this indicate that DUT is not connected.
	maxPPDut5MVWhenNotConnected = 500

	// This token represents the command string that can be present in
	// the extra arguments defined in config.
	commandToken = "command"

	// This token represents the string in config extra arguments that
	// conveys the expected string value for a servod command.
	stringValueExtraArgToken = "expected_string_value"

	// This token represents the string in config extra arguments that
	// conveys the expected int value for a servod command.
	intValueExtraArgToken = "expected_int_value"

	// This token represents the string in config extra arguments that
	// conveys the expected floating-point value for a servod command.
	floatValueExtraArgToken = "expected_float_value"

	// This token represents the string in config extra arguments that
	// conveys the expected boolean value for a servod command.
	boolValueExtraArgToken = "expected_bool_value"
)

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

func servoDetectUSBKeyExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	res, err := ServodCallGet(ctx, args, "image_usbkey_dev")
	if err != nil {
		args.DUT.ServoHost.UsbkeyState = tlw.HardwareStateNotDetected
		return errors.Annotate(err, "servo detect usb key exec: could not obtain usb path on servo: %q", err).Err()
	} else if res.Value.GetString_() == "" {
		args.DUT.ServoHost.UsbkeyState = tlw.HardwareStateNotDetected
		return errors.Reason("servo detect usb key exec: the path to usb drive is empty").Err()
	}
	servoUsbPath := res.Value.GetString_()
	log.Debug(ctx, "Servo Detect USB-Key Exec: USB-key path: %s.", servoUsbPath)
	run := args.NewRunner(args.DUT.ServoHost.Name)
	if _, err := run(ctx, time.Minute, fmt.Sprintf("fdisk -l %s", servoUsbPath)); err != nil {
		args.DUT.ServoHost.UsbkeyState = tlw.HardwareStateNotDetected
		return errors.Annotate(err, "servo detect usb key exec: could not determine whether %q is a valid usb path", servoUsbPath).Err()
	}
	if args.DUT.ServoHost.UsbkeyState == tlw.HardwareStateNeedReplacement {
		// This device has been marked for replacement. A further
		// audit action is required to correct this.
		log.Debug(ctx, "Servo Detect USB-Key Exec: device marked for replacement.")
	} else {
		args.DUT.ServoHost.UsbkeyState = tlw.HardwareStateNormal
	}
	return nil
}

func runCheckOnHost(ctx context.Context, run execs.Runner, usbPath string, timeout time.Duration) (tlw.HardwareState, error) {
	command := fmt.Sprintf(badBlocksCommandPrefix, usbPath)
	log.Debug(ctx, "Run Check On Host: Executing %q", command)
	// The execution timeout for this audit job is configured at the
	// level of the action. So the execution of this command will be
	// bound by that.
	out, err := run(ctx, timeout, command)
	switch {
	case err == nil:
		// TODO(vkjoshi@): recheck if this is required, or does stderr need to be examined.
		if len(out) > 0 {
			return tlw.HardwareStateNeedReplacement, nil
		}
		return tlw.HardwareStateNormal, nil
	case execs.SSHErrorLinuxTimeout.In(err): // 124 timeout
		fallthrough
	case execs.SSHErrorCLINotFound.In(err): // 127 badblocks
		return "", errors.Annotate(err, "run check on host: could not successfully complete check").Err()
	default:
		return tlw.HardwareStateNeedReplacement, nil
	}
}

func servoAuditUSBKeyExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	dutUsb := ""
	if cros.IsSSHable(ctx, args.NewRunner(args.DUT.Name)) == nil {
		log.Debug(ctx, "Servo Audit USB-Key Exec: %q is reachable through SSH", args.DUT.Name)
		var err error = nil
		dutUsb, err = GetUSBDrivePathOnDut(ctx, args)
		if err != nil {
			log.Debug(ctx, "Servo Audit USB-Key Exec: could not determine USB-drive path on DUT: %q, error: %q. This is not critical. We will continue the audit by setting the path to empty string.", args.DUT.Name, err)
		}
	} else {
		log.Debug(ctx, "Servo Audit USB-Key Exec: continue audit from servo-host because DUT %q is not reachable through SSH", args.DUT.Name)
	}
	if dutUsb != "" {
		// DUT is reachable, and we found a USB drive on it.
		state, err := runCheckOnHost(ctx, args.NewRunner(args.DUT.Name), dutUsb, 2*time.Hour)
		if err != nil {
			return errors.Reason("servo audit usb key exec: could not check DUT usb path %q", dutUsb).Err()
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
			return errors.Annotate(err, "servo audit usb key exec: could not obtain usb path on servo: %q", err).Err()
		}
		servoUsbPath := strings.TrimSpace(result.Value.GetString_())
		if servoUsbPath == "" {
			args.DUT.ServoHost.UsbkeyState = tlw.HardwareStateNotDetected
			log.Debug(ctx, "Servo Audit USB-Key Exec: cannot continue audit because the path to USB-Drive is empty")
			return errors.Reason("servo audit usb key exec: the path to usb drive is empty").Err()
		}
		state, err := runCheckOnHost(ctx, args.NewRunner(args.DUT.ServoHost.Name), servoUsbPath, 2*time.Hour)
		if err != nil {
			log.Debug(ctx, "Servo Audit USB-Key Exec: error %q during audit of USB-Drive", err)
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
		return errors.Annotate(err, "is root servo present exec").Err()
	}
	if !topology.IsItemGood(ctx, rootServo) {
		log.Info(ctx, "Is Servo Root Present Exec: no good root servo found")
		return errors.Reason("is servo root present exec: no good root servo found").Err()
	}
	log.Info(ctx, "is servo root present exec: success")
	return nil
}

// Verify that the root servo is enumerated/present on the host.
func servoTopologyUpdateExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	runner := args.NewRunner(args.DUT.ServoHost.Name)
	servoTopology, err := topology.RetrieveServoTopology(ctx, runner, args.DUT.ServoHost.Servo.SerialNumber)
	if err != nil {
		return errors.Annotate(err, "servo topology update exec").Err()
	}
	if servoTopology.Root == nil {
		return errors.Reason("servo topology update exec: root servo not found").Err()
	}
	argsMap := execs.ParseActionArgs(ctx, actionArgs, execs.DefaultSplitter)
	minChildCount := topologyMinChildCountDefaultValue
	persistTopology := persistTopologyDefaultValue
	for k, v := range argsMap {
		log.Debug(ctx, "Servo Topology Update Exec: k:%q, v:%q", k, v)
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
					return errors.Reason("servo topology update exec: malformed min child config in action arg %q:%q", k, v).Err()
				}
			case persistTopologyArg:
				persistTopology, err = strconv.ParseBool(v)
				if err != nil {
					return errors.Reason("servo topology update exec: malformed update servo config in action arg %q:%q", k, v).Err()
				}
			}
		}
	}
	if len(servoTopology.Children) < minChildCount {
		return errors.Reason("servo topology update exec: expected a min of %d children, found %d", minChildCount, len(servoTopology.Children)).Err()
	}
	if persistTopology {
		// This verified topology will be used in all subsequent
		// action that need the servo topology. This will avoid time
		// with re-fetching the topology.
		args.DUT.ServoHost.ServoTopology = servoTopology
	}
	return nil
}

// Verify that servod is responsive
func servoServodEchoHostExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	runner := args.NewRunner(args.DUT.ServoHost.Name)
	v, err := runner(ctx, time.Minute, fmt.Sprintf(servodHostCheckupCmd, args.DUT.ServoHost.ServodPort))
	if err != nil {
		return errors.Annotate(err, "servo servod echo host exec: servod is not responsive for dut-control commands").Err()
	}
	log.Debug(ctx, "Servo Servod Echo Host Exec: Servod is responsive: %q", v)
	return nil
}

// Verify that the servo firmware is up-to-date.
func servoFirmwareNeedsUpdateExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	runner := args.NewRunner(args.DUT.ServoHost.Name)
	// The servo topology check should have already been done in an
	// action. The topology determined at that time would have been
	// saved in this data structure if the 'updateServo' argument was
	// passed for that action. We will make use of any such persisting
	// topology instead of re-computing it. This is avoid unnecessary
	// expenditure of time in obtaining the topology here.
	devices := topology.AllDevices(args.DUT.ServoHost.ServoTopology)
	var err error
	if devices == nil {
		// This situation can arise if the servo topology has been
		// verified in an earlier action, but the topology was not
		// persisted because the updateServo parameter was not set in
		// that action. In this case we do not have any choice but to
		// re-compute the topology.
		devices, err = topology.ListOfDevices(ctx, runner, args.DUT.ServoHost.Servo.SerialNumber)
		if err != nil {
			errors.Annotate(err, "servo firmware needs update exec").Err()
		}
		log.Debug(ctx, "Servo Firmware Needs Update Exec: topology re-computer because pre-existing servo topology not found.")
	}
	for _, d := range devices {
		if topology.IsItemGood(ctx, d) {
			log.Debug(ctx, "Servo Firmware Needs Update Exec: device type (d.Type) :%q.", d.Type)
			if needsUpdate(ctx, runner, d, args.DUT.ServoHost.Servo.FirmwareChannel) {
				log.Debug(ctx, "Servo Firmware Needs Update Exec: needs update is true")
				return errors.Reason("servo firmware needs update exec: servo needs update").Err()
			}
		}
	}
	return nil
}

// servoSetExec sets the command of the servo a specific value using servod.
// It reads the command and its value from the actionArgs argument.
//
// the actionArgs should be in the format of ["command:....", "string_value:...."]
func servoSetExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	m := execs.ParseActionArgs(ctx, actionArgs, execs.DefaultSplitter)
	command, existed := m["command"]
	if !existed {
		return errors.Reason("servo match state: command not found in the argument").Err()
	}
	stringValue, existed := m["string_value"]
	if !existed {
		return errors.Reason("servo match state: string value not found in the argument").Err()
	}
	command = strings.TrimSpace(command)
	stringValue = strings.TrimSpace(stringValue)
	_, err := ServodCallSet(ctx, args, command, stringValue)
	if err != nil {
		return errors.Annotate(err, "servo match state").Err()
	}
	return nil
}

// Verify that the DUT is connected to Servo using the 'ppdut5_mv'
// servod control.
func servoLowPPDut5Exec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	if _, err := ServodCallHas(ctx, args, servodPPDut5Cmd); err != nil {
		return errors.Annotate(err, "servo low ppdut5 exec").Err()
	}
	res, err := ServodCallGet(ctx, args, servodPPDut5Cmd)
	if err != nil {
		return errors.Annotate(err, "servo low ppdut5 exec").Err()
	}
	voltageValue := res.Value.GetInt()
	if voltageValue < maxPPDut5MVWhenNotConnected {
		return errors.Reason("servo low ppdut5 exec: the ppdut5_mv value %d is lower than the threshold %d", voltageValue, maxPPDut5MVWhenNotConnected).Err()
	}
	// TODO: (vkjoshi@): add metrics to collect the value of the
	// servod control ppdut5_mv when it is below a certain threshold.
	// (ref:http://cs/chromeos_public/src/third_party/labpack/files/server/hosts/servo_repair.py?l=640).
	return nil
}

// servoCheckServodControlExec verifies that servod supports the
// control mentioned in action args. Additionally, if actionArgs
// includes the expected value, this function will verify that the
// value returned by servod for this control matches the expected
// value.
func servoCheckServodControlExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	argsMap := execs.ParseActionArgs(ctx, actionArgs, execs.DefaultSplitter)
	command, ok := argsMap[commandToken]
	log.Debug(ctx, "Servo Check Servod Control Exec: %s ok :%t", commandToken, ok)
	if !ok {
		// It is a failure condition if an action invokes this exec,
		// and does not specify the servod command.
		return errors.Reason("servo check servod control exec: no command is mentioned for this action.").Err()
	} else if len(command) == 0 {
		return errors.Reason("servo check servod control exec: malformed (empty) command.").Err()
	}
	var expectedValue string
	var compare func(ctx context.Context) error
	// TODO (vkjoshi@): revisit the logic of implementations of the
	// function 'compare', e.g., will it make sense to use a helper
	// function for this?
	if expectedValue, ok = argsMap[stringValueExtraArgToken]; ok {
		controlValue, err := servodGetString(ctx, args, command)
		if err != nil {
			return errors.Annotate(err, "servo check servod control exec").Err()
		}
		compare = func(ctx context.Context) error {
			log.Debug(ctx, "Compare (String), expected value %q, actual value %q", expectedValue, controlValue)
			if controlValue != expectedValue {
				log.Debug(ctx, "Compare (String), expected value %q, actual value %q do not match.", expectedValue, controlValue)
				return errors.Reason("compare (string): expected value %q, actual value %q do not match.", expectedValue, controlValue).Err()
			}
			return nil
		}
	} else if expectedValue, ok = argsMap[intValueExtraArgToken]; ok {
		controlValue, err := servodGetInt(ctx, args, command)
		if err != nil {
			return errors.Annotate(err, "servo check servod control exec").Err()
		}
		compare = func(ctx context.Context) error {
			log.Debug(ctx, "Compare (Int), expected value %s, actual value %d", expectedValue, controlValue)
			expectedInt, err := strconv.Atoi(expectedValue)
			if err != nil {
				return errors.Annotate(err, "compare (int32)").Err()
			}
			if controlValue != int32(expectedInt) {
				return errors.Reason("compare: expected value %d, actual value %d do not match", int32(expectedInt), controlValue).Err()
			}
			return nil
		}
	} else if expectedValue, ok = argsMap[floatValueExtraArgToken]; ok {
		controlValue, err := servodGetDouble(ctx, args, command)
		if err != nil {
			return errors.Annotate(err, "servo check servod control exec").Err()
		}
		compare = func(ctx context.Context) error {
			log.Debug(ctx, "Compare (Double), expected value %s, actual value %f", expectedValue, controlValue)
			expectedDouble, err := strconv.ParseFloat(expectedValue, 64)
			if err != nil {
				return errors.Annotate(err, "compare (float64)").Err()
			}
			if controlValue != expectedDouble {
				return errors.Reason("compare: expected value %f, actual value %f do not match", expectedDouble, controlValue).Err()
			}
			return nil
		}
	} else if expectedValue, ok = argsMap[boolValueExtraArgToken]; ok {
		controlValue, err := servodGetBool(ctx, args, command)
		if err != nil {
			return errors.Annotate(err, "servo check servod control exec").Err()
		}
		compare = func(ctx context.Context) error {
			log.Debug(ctx, "Compare (Bool), expected value %s, actual value %t", expectedValue, controlValue)
			expectedBool, err := strconv.ParseBool(expectedValue)
			if err != nil {
				return errors.Annotate(err, "compare (int32)").Err()
			}
			if controlValue != expectedBool {
				return errors.Reason("compare: expected value %t, actual value %t do not match", expectedBool, controlValue).Err()
			}
			return nil
		}
	}
	if compare == nil {
		log.Info(ctx, "Servo Check Servod Control Exec: expected value type not specified in config, or did not match any known types.")
		controlValue, err := servodGetString(ctx, args, command)
		if err != nil {
			return errors.Annotate(err, "Servo Check Servod Control Exec").Err()
		}
		log.Info(ctx, "Servo Check Servod Control Exec: for command %q, read the value %q from servod.", command, controlValue)
	} else if err := compare(ctx); err != nil {
		return errors.Annotate(err, "servo check servod control exec").Err()
	}
	return nil
}

const (
	// removeFileCmd is the linux file removal command that used to remove files in the filesToRemoveSlice.
	removeFileCmd = `rm %s`
)

var filesToRemoveSlice = []string{
	"/var/lib/metrics/uma-events",
	"/var/spool/crash/*",
	"/var/log/chrome/*",
	"/var/log/ui/*",
	"/home/chronos/BrowserMetrics/*",
}

// servoLabstationDiskCleanUpExec remove files that are in the filesToRemoveSlice.
func servoLabstationDiskCleanUpExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	r := args.NewRunner(args.ResourceName)
	// Remove all files in the filesToRemoveSlice during the labstation disk clean up process.
	for _, filePath := range filesToRemoveSlice {
		if _, err := r(ctx, time.Minute, fmt.Sprintf(removeFileCmd, filePath)); err != nil {
			log.Debug(ctx, "servo labstation disk clean up: %s", err.Error())
		}
		log.Info(ctx, "labstation file removed: %s", filePath)
	}
	return nil
}

const (
	// removeOldServodLogsCmd is the command to remove any servod files that is older than the maximum days specified by d.
	removeOldServodLogsCmd = `/usr/bin/find /var/log/servod_* -mtime +%d -print -delete`
)

// servoServodOldLogsCleanupExec removes the old servod log files that existed more than keepLogsMaxDays days.
//
// @params: actionArgs should be in the format of: ["max_days:5"]
func servoServodOldLogsCleanupExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	daysMap := execs.ParseActionArgs(ctx, actionArgs, execs.DefaultSplitter)
	keepLogsMaxDaysString, existed := daysMap["max_days"]
	if !existed {
		return errors.Reason("servod old logs: missing max days information in the argument").Err()
	}
	keepLogsMaxDaysString = strings.TrimSpace(keepLogsMaxDaysString)
	if keepLogsMaxDaysString == "" {
		return errors.Reason("servod old logs: max days information is empty").Err()
	}
	keepLogsMaxDays, err := strconv.ParseInt(keepLogsMaxDaysString, 10, 64)
	if err != nil {
		return errors.Annotate(err, "servod old logs").Err()
	}
	log.Info(ctx, "The max number of days for keeping old servod logs is: %v", keepLogsMaxDays)
	r := args.NewRunner(args.ResourceName)
	// remove old servod logs.
	if _, err := r(ctx, time.Minute, fmt.Sprintf(removeOldServodLogsCmd, keepLogsMaxDays)); err != nil {
		log.Debug(ctx, "servo servod old logs clean up: %s", err.Error())
	}
	return nil
}

// servoValidateBatteryChargingExec uses servod controls to check
// whether or not the battery on a DUT is capable of getting
// charged. It marks the DUT for replacement if its battery cannot be
// charged.
func servoValidateBatteryChargingExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	// This is the number of times we will try to read the value of
	// battery gcontrols from servod.
	const servodBatteryReadRetryLimit = 3
	// This is the servod control to determine battery's last full
	// charge.
	const batteryFullChargeServodControl = "battery_full_charge_mah"
	// This is the servod control to determine the bettery's full
	// capacity by design.
	const batteryDesignFullCapacityServodControl = "battery_full_design_mah"
	var lastFullCharge, batteryCapacity int32
	var getLastFullCharge = func() error {
		var err error
		lastFullCharge, err = servodGetInt(ctx, args, batteryFullChargeServodControl)
		return err
	}
	if err := retry.LimitCount(ctx, servodBatteryReadRetryLimit, -1, getLastFullCharge, "get last full charge"); err != nil {
		log.Debug(ctx, "Servo Validate Battery Charging Exec: could not read last full charge despite trying %d times", servodBatteryReadRetryLimit)
		return errors.Annotate(err, "servo validate battery charging exec").Err()
	}
	log.Debug(ctx, "Servo Validate Battery Charging Exec: last full charge is %d", lastFullCharge)
	var getBatteryCapacity = func() error {
		var err error
		batteryCapacity, err = servodGetInt(ctx, args, batteryDesignFullCapacityServodControl)
		return err
	}
	if err := retry.LimitCount(ctx, servodBatteryReadRetryLimit, -1, getBatteryCapacity, "get battery capacity"); err != nil {
		log.Debug(ctx, "Servo Validate Battery Charging Exec: could not read battery capacity despite trying %d times", servodBatteryReadRetryLimit)
		return errors.Annotate(err, "servo validate battery charging exec").Err()
	}
	log.Debug(ctx, "Servo Validate Battery Charging Exec: battery capacity is %d", batteryCapacity)
	hardwareState := battery.DetermineHardwareStatus(ctx, float64(lastFullCharge), float64(batteryCapacity))
	log.Info(ctx, "Battery hardware state: %s", hardwareState)
	if hardwareState == tlw.HardwareStateUnspecified {
		return errors.Reason("audit battery: dut battery did not detected or state cannot extracted").Err()
	}
	if hardwareState == tlw.HardwareStateNeedReplacement {
		log.Info(ctx, "Detected issue with storage on the DUT.")
		args.DUT.Battery.State = tlw.HardwareStateNeedReplacement
	}
	return nil
}

// initDutForServoExec initializes the DUT and sets all servo signals
// to default values.
func initDutForServoExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	verbose := true
	if _, err := ServodCallHwInit(ctx, args, verbose); err != nil {
		return errors.Annotate(err, "init dut for servo exec").Err()
	}
	usbMuxControl := "usb_mux_oe1"
	if _, err := ServodCallHas(ctx, args, usbMuxControl); err == nil {
		if _, err2 := ServodCallSet(ctx, args, usbMuxControl, "on"); err2 != nil {
			return errors.Annotate(err, "init dut for servo exec").Err()
		}
		if _, err := ServodCallSet(ctx, args, "image_usbkey_pwr", "off"); err != nil {
			return errors.Annotate(err, "init dut for servo exec").Err()
		}
	} else {
		log.Debug(ctx, "Init Dut For Servo Exec: servod control %q is not available.", usbMuxControl)
	}
	return nil
}

// servoUpdateServoFirmwareExec updates all the servo devices' firmware based on the condition specified by the actionArgs
//
// @params try_attempt_count:  Count of attempts to update servo. For force option the count attempts is always 1 (one).
// @params try_force_update_after_fail:   Try force force option if fail to update in normal mode.
// @params force_update:       Run updater with force option. Override try_force_update_after_fail option.
// @params ignore_version:     Skip check the version on the device.
//
// @params: actionArgs should be in the format of:
// Ex: ["try_attempt_count:x", "try_force_update_after_fail:true/false", "force_update:true/false", "ignore_version:true/false"]
func servoUpdateServoFirmwareExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	fwUpdateMap := execs.ParseActionArgs(ctx, actionArgs, execs.DefaultSplitter)
	// If the passed in "try_attempt_count" is either 0 or cannot be parsed successfully,
	// then, we default the count to be 1 to at least try to update it once.
	tryAttemptCount := fwUpdateMap.AsInt(ctx, "try_attempt_count", 1)
	tryForceUpdateAfterFail := fwUpdateMap.AsBool(ctx, "try_force_update_after_fail")
	forceUpdate := fwUpdateMap.AsBool(ctx, "force_update")
	ignoreVersion := fwUpdateMap.AsBool(ctx, "ignore_version")
	run := args.NewRunner(args.DUT.ServoHost.Name)
	if forceUpdate {
		// If requested to update with force then first attempt will be with force
		// and there no second attempt.
		tryAttemptCount = 1
		tryForceUpdateAfterFail = false
	}
	req := FwUpdaterRequest{
		UseContainer:            IsContainerizedServoHost(ctx, args.DUT.ServoHost),
		FirmwareChannel:         args.DUT.ServoHost.Servo.FirmwareChannel,
		TryAttemptCount:         tryAttemptCount,
		TryForceUpdateAfterFail: tryForceUpdateAfterFail,
		ForceUpdate:             forceUpdate,
		IgnoreVersion:           ignoreVersion,
	}
	failBoards := UpdateBoardsServoFw(ctx, run, req, topology.AllDevices(args.DUT.ServoHost.ServoTopology))
	if len(failBoards) != 0 {
		args.DUT.ServoHost.Servo.State = tlw.ServoStateNeedReplacement
		return errors.Reason("servo update servo firmware: %d servo devices fails the update process", len(failBoards)).Err()
	}
	return nil
}

const (
	// Servod uart command for use in servo v4/v4p1.
	//
	// TODO: (@yunzhiyu) use servodUartCmd for both servo v4 /v4p1
	// in the future instead of servodUartV4Cmd and servodUartV4P1Cmd separately.
	// Currently blocked by: (b/198638900).
	servodUartV4Cmd   = "servo_v4_uart_cmd"
	servodUartV4P1Cmd = "servo_v4p1_uart_cmd"
	servodUartCmd     = "root.servo_uart_cmd"
)

// servoFakeDisconnectDUTExec tries to unplug DUT from servo and restore the connection.
//
// @params: actionArgs should be in the format of:
// Ex: ["delay_in_ms:x", "timeout_in_ms:x"]
func servoFakeDisconnectDUTExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	argsMap := execs.ParseActionArgs(ctx, actionArgs, execs.DefaultSplitter)
	// Delay to disconnect in milliseconds. Default to be 100ms.
	delayMS := argsMap.AsInt(ctx, "delay_in_ms", 100)
	// Timeout to wait to restore the connection. Default to be 2000ms.
	timeoutMS := argsMap.AsInt(ctx, "timeout_in_ms", 2000)
	disconnectCmd := fmt.Sprintf(`fakedisconnect %d %d`, delayMS, timeoutMS)
	uartCmd := servodUartV4Cmd
	// TODO: (yunzhiyu@): change logic to use unified servod cmd: "root.servo_uart_cmd"
	// As in the (b/204369636), currently blocked by (b/198638900).
	if _, err := ServodCallHas(ctx, args, uartCmd); err != nil {
		log.Debug(ctx, "Servod control %q is not supported", servodUartCmd)
		uartCmd = servodUartV4P1Cmd
		log.Debug(ctx, "Using Servod control %q instead", uartCmd)
	}
	if _, err := ServodCallSet(ctx, args, uartCmd, disconnectCmd); err != nil {
		return errors.Annotate(err, "servod fake disconnect servo").Err()
	}
	// Formula to cover how long we wait to see the effect
	// when we convert params to seconds and then +2 seconds to apply effect.
	waitFinishExecutionTimeout := (delayMS+timeoutMS)/1000 + 2
	time.Sleep(time.Duration(waitFinishExecutionTimeout) * time.Second)
	return nil
}

func init() {
	execs.Register("servo_host_servod_init", servodInitActionExec)
	execs.Register("servo_host_servod_stop", servodStopActionExec)
	execs.Register("servo_host_servod_restart", servodRestartActionExec)
	execs.Register("servo_detect_usbkey", servoDetectUSBKeyExec)
	execs.Register("servo_audit_usbkey", servoAuditUSBKeyExec)
	execs.Register("servo_v4_root_present", isRootServoPresentExec)
	execs.Register("servo_topology_update", servoTopologyUpdateExec)
	execs.Register("servo_servod_echo_host", servoServodEchoHostExec)
	execs.Register("servo_fw_need_update", servoFirmwareNeedsUpdateExec)
	execs.Register("servo_set", servoSetExec)
	execs.Register("servo_low_ppdut5", servoLowPPDut5Exec)
	execs.Register("servo_check_servod_control", servoCheckServodControlExec)
	execs.Register("servo_labstation_disk_cleanup", servoLabstationDiskCleanUpExec)
	execs.Register("servo_servod_old_logs_cleanup", servoServodOldLogsCleanupExec)
	execs.Register("servo_battery_charging", servoValidateBatteryChargingExec)
	execs.Register("init_dut_for_servo", initDutForServoExec)
	execs.Register("servo_update_servo_firmware", servoUpdateServoFirmwareExec)
	execs.Register("servo_fake_disconnect_dut", servoFakeDisconnectDUTExec)
}
