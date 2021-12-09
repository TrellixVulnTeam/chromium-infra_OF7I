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
func hasDutNameActionExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	if args.DUT != nil && args.DUT.Name != "" {
		log.Debug(ctx, "DUT name: %q", args.DUT.Name)
		return nil
	}
	return errors.Reason("dut name is empty").Err()
}

// hasDutBoardActionExec verifies that DUT provides board name.
func hasDutBoardActionExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	if args.DUT != nil && args.DUT.Board != "" {
		log.Debug(ctx, "DUT board name: %q", args.DUT.Board)
		return nil
	}
	return errors.Reason("dut board name is empty").Err()
}

// hasDutModelActionExec verifies that DUT provides model name.
func hasDutModelActionExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	if args.DUT != nil && args.DUT.Model != "" {
		log.Debug(ctx, "DUT model name: %q", args.DUT.Model)
		return nil
	}
	return errors.Reason("dut model name is empty").Err()
}

// hasDutDeviceSkuActionExec verifies that DUT has the device sku label.
func hasDutDeviceSkuActionExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	deviceSkuLabel := args.DUT.DeviceSku
	if deviceSkuLabel == "" {
		return errors.Reason("dut device sku label is empty").Err()
	}
	log.Debug(ctx, "dut device sku label: %s.", deviceSkuLabel)
	return nil
}

// dutCheckModelExec checks whether the model name for the current DUT
// matches any of the values specified in config. It returns an error
// based on the directive in config to invert the results.
func dutCheckModelExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	argsMap := execs.ParseActionArgs(ctx, actionArgs, execs.DefaultSplitter)
	// This token represents the string in config extra arguments that
	// conveys the expected string value(s) for a DUT attribute.
	const stringValuesExtraArgToken = "string_values"
	// This token represents whether the success-status of an exec
	// should be inverted. For example, using this flag, we can
	// control whether the value of a DUT Model should, or should not
	// be present in the list of strings mentioned in the config.
	const invertResultToken = "invert_result"
	invertResultsFlag := argsMap.AsBool(ctx, invertResultToken)
	for _, m := range argsMap.AsStringSlice(ctx, stringValuesExtraArgToken) {
		m = strings.TrimSpace(m)
		if strings.EqualFold(m, args.DUT.Model) {
			msg := fmt.Sprintf("DUT Model %s found in the list of models in config", args.DUT.Model)
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

func init() {
	execs.Register("has_dut_name", hasDutNameActionExec)
	execs.Register("has_dut_board_name", hasDutBoardActionExec)
	execs.Register("has_dut_model_name", hasDutModelActionExec)
	execs.Register("has_dut_device_sku", hasDutDeviceSkuActionExec)
	execs.Register("dut_check_model", dutCheckModelExec)
}
