// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dut

import (
	"context"
	"fmt"
	"strings"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/log"
)

// hasDutNameActionExec verifies that DUT provides name.
func hasDutNameActionExec(ctx context.Context, info *execs.ExecInfo) error {
	if info.RunArgs.DUT != nil && info.RunArgs.DUT.Name != "" {
		log.Debug(ctx, "DUT name: %q", info.RunArgs.DUT.Name)
		return nil
	}
	return errors.Reason("dut name is empty").Err()
}

// hasDutBoardActionExec verifies that DUT provides board name.
func hasDutBoardActionExec(ctx context.Context, info *execs.ExecInfo) error {
	if d := info.RunArgs.DUT; d != nil && d.Board != "" {
		log.Debug(ctx, "DUT board name: %q", d.Board)
		return nil
	}
	return errors.Reason("dut board name is empty").Err()
}

// hasDutModelActionExec verifies that DUT provides model name.
func hasDutModelActionExec(ctx context.Context, info *execs.ExecInfo) error {
	if d := info.RunArgs.DUT; d != nil && d.Model != "" {
		log.Debug(ctx, "DUT model name: %q", d.Model)
		return nil
	}
	return errors.Reason("dut model name is empty").Err()
}

// dutServolessExec verifies that setup is servoless.
func dutServolessExec(ctx context.Context, info *execs.ExecInfo) error {
	if sh := info.RunArgs.DUT.ServoHost; sh == nil || (sh.Name == "" && sh.ContainerName == "") {
		log.Debug(ctx, "DUT servoless confirmed!")
		return nil
	}
	return errors.Reason("dut is servoless").Err()
}

// hasDutDeviceSkuActionExec verifies that DUT has the device sku label.
func hasDutDeviceSkuActionExec(ctx context.Context, info *execs.ExecInfo) error {
	deviceSkuLabel := info.RunArgs.DUT.DeviceSku
	if deviceSkuLabel == "" {
		return errors.Reason("dut device sku label is empty").Err()
	}
	log.Debug(ctx, "dut device sku label: %s.", deviceSkuLabel)
	return nil
}

const (
	// This token represents the string in config extra arguments that
	// conveys the expected string value(s) for a DUT attribute.
	stringValuesExtraArgToken = "string_values"
	// This token represents whether the success-status of an exec
	// should be inverted. For example, using this flag, we can
	// control whether the value of a DUT Model should, or should not
	// be present in the list of strings mentioned in the config.
	invertResultToken = "invert_result"
)

// dutCheckModelExec checks whether the model name for the current DUT
// matches any of the values specified in config. It returns an error
// based on the directive in config to invert the results.
func dutCheckModelExec(ctx context.Context, info *execs.ExecInfo) error {
	argsMap := info.GetActionArgs(ctx)
	invertResultsFlag := argsMap.AsBool(ctx, invertResultToken, false)
	for _, m := range argsMap.AsStringSlice(ctx, stringValuesExtraArgToken) {
		m = strings.TrimSpace(m)
		if strings.EqualFold(m, info.RunArgs.DUT.Model) {
			msg := fmt.Sprintf("DUT Model %s found in the list of models in config", info.RunArgs.DUT.Model)
			log.Debug(ctx, "Dut Check Model Exec :%s.", msg)
			if invertResultsFlag {
				return errors.Reason("dut check model exec: %s", msg).Err()
			}
			return nil
		}
	}
	msg := "No matching model found"
	log.Debug(ctx, "Dut Check Model Exec: %s", msg)
	if invertResultsFlag {
		return nil
	}
	return errors.Reason("dut check model exec: %s", msg).Err()
}

// dutCheckBoardExec checks whether the board name for the current DUT
// matches any of the values specified in config. It returns an error
// based on the directive in config to invert the results.
func dutCheckBoardExec(ctx context.Context, info *execs.ExecInfo) error {
	argsMap := info.GetActionArgs(ctx)
	invertResultsFlag := argsMap.AsBool(ctx, invertResultToken, false)
	for _, m := range argsMap.AsStringSlice(ctx, stringValuesExtraArgToken) {
		m = strings.TrimSpace(m)
		if strings.EqualFold(m, info.RunArgs.DUT.Board) {
			msg := fmt.Sprintf("DUT Board %s found in the list of boards in config", info.RunArgs.DUT.Model)
			log.Debug(ctx, "Dut Check Board Exec :%s.", msg)
			if invertResultsFlag {
				return errors.Reason("dut check board exec: %s", msg).Err()
			}
			return nil
		}
	}
	msg := "No matching board found"
	log.Debug(ctx, "Dut Check Board Exec: %s", msg)
	if invertResultsFlag {
		return nil
	}
	return errors.Reason("dut check board exec: %s", msg).Err()
}

// servoVerifySerialNumberExec verifies that the servo host attached
// to the DUT has a serial number configured.
func servoVerifySerialNumberExec(ctx context.Context, info *execs.ExecInfo) error {
	if d := info.RunArgs.DUT; d != nil && d.ServoHost != nil && d.ServoHost.Servo != nil && d.ServoHost.Servo.SerialNumber != "" {
		log.Debug(ctx, "Servo Verify Serial Number : %q", d.ServoHost.Servo.SerialNumber)
		return nil
	}
	return errors.Reason("servo verify serial number: serial number is not available").Err()
}

// servoHostPresentExec verifies that servo host present under DUT.
func servoHostPresentExec(ctx context.Context, info *execs.ExecInfo) error {
	if d := info.RunArgs.DUT; d == nil || d.ServoHost == nil {
		return errors.Reason("servo host present: data is not present").Err()
	}
	return nil
}

// dutInAudioBoxExec checks if DUT is in audio box.
func dutInAudioBoxExec(ctx context.Context, info *execs.ExecInfo) error {
	if d := info.RunArgs.DUT; d == nil || d.Audio == nil || !d.Audio.GetInBox() {
		return errors.Reason("is audio box: is not in audio-box").Err()
	}
	return nil
}

// hasBatteryExec checks if DUT is expected to have a battery.
func hasBatteryExec(ctx context.Context, info *execs.ExecInfo) error {
	if d := info.RunArgs.DUT; d == nil || d.Battery == nil {
		return errors.Reason("has battery: data is not present").Err()
	}
	return nil
}

func init() {
	execs.Register("dut_servo_host_present", servoHostPresentExec)
	execs.Register("dut_has_name", hasDutNameActionExec)
	execs.Register("dut_has_board_name", hasDutBoardActionExec)
	execs.Register("dut_has_model_name", hasDutModelActionExec)
	execs.Register("dut_has_device_sku", hasDutDeviceSkuActionExec)
	execs.Register("dut_check_model", dutCheckModelExec)
	execs.Register("dut_check_board", dutCheckBoardExec)
	execs.Register("dut_servoless", dutServolessExec)
	execs.Register("dut_is_in_audio_box", dutInAudioBoxExec)
	execs.Register("dut_servo_has_serial", servoVerifySerialNumberExec)
	execs.Register("dut_has_battery", hasBatteryExec)
}
