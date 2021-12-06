// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/proto"
	"go.chromium.org/chromiumos/config/go/payload"
	"go.chromium.org/chromiumos/config/go/test/api"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"infra/libs/fleet/boxster/swarming"
	ufspb "infra/unifiedfleet/api/v1/models"
	chromeosLab "infra/unifiedfleet/api/v1/models/chromeos/lab"
)

const (
	dutStateSourceStr  = "dut_state"
	labConfigSourceStr = "lab_config"
)

type TleSource struct {
	path       string
	configType string
}

var TLE_LABEL_MAPPING = map[string]*TleSource{
	"misc-license":                createTleSourceForLabConfig("chromeos_machine_lse.device_lse.dut.licenses..type", "lab_config"),
	"peripheral-audio-board":      createTleSourceForLabConfig("UNIMPLEMENTED", "lab_config"),
	"peripheral-audio-box":        createTleSourceForLabConfig("chromeos_machine_lse.device_lse.dut.peripherals.audio.audio_box", "lab_config"),
	"peripheral-audio-cable":      createTleSourceForLabConfig("chromeos_machine_lse.device_lse.dut.peripherals.audio.audio_cable", "lab_config"),
	"peripheral-audio-loopback":   createTleSourceForLabConfig("UNIMPLEMENTED", "lab_config"),
	"peripheral-camerabox-facing": createTleSourceForLabConfig("chromeos_machine_lse.device_lse.dut.peripherals.camerabox_info.facing", "lab_config"),
	"peripheral-camerabox-light":  createTleSourceForLabConfig("chromeos_machine_lse.device_lse.dut.peripherals.camerabox_info.light", "lab_config"),
	"peripheral-chameleon":        createTleSourceForLabConfig("UNIMPLEMENTED", "lab_config"),
	"peripheral-num-btpeer":       createTleSourceForLabConfig("working_bluetooth_btpeer", "dut_state"),
	"peripheral-servo-state":      createTleSourceForLabConfig("servo", "dut_state"),
	"peripheral-servo-usb-state":  createTleSourceForLabConfig("servo_usb_state", "dut_state"),
	"peripheral-wificell":         createTleSourceForLabConfig("chromeos_machine_lse.device_lse.dut.peripherals.wifi.wificell", "lab_config"),
	"swarming-pool":               createTleSourceForLabConfig("chromeos_machine_lse.device_lse.dut.pools", "lab_config"),
}

func createTleSourceForLabConfig(path string, configType string) *TleSource {
	return &TleSource{
		path:       path,
		configType: configType,
	}
}

// Convert converts one DutAttribute label to multiple Swarming labels.
//
// For all TleSource labels needed to be converted for UFS, the implementation
// is handled in this file. All other labels uses the Boxster Swarming lib for
// conversion.
func Convert(ctx context.Context, dutAttr *api.DutAttribute, flatConfig *payload.FlatConfig, lse *ufspb.MachineLSE, dutState *chromeosLab.DutState) ([]string, error) {
	if dutAttr.GetTleSource() != nil {
		return convertTleSource(ctx, dutAttr, lse, dutState)
	}
	return swarming.ConvertAll(dutAttr, flatConfig)
}

// convertTleSource handles the label conversion of MachineLSE and DutState.
func convertTleSource(ctx context.Context, dutAttr *api.DutAttribute, lse *ufspb.MachineLSE, dutState *chromeosLab.DutState) ([]string, error) {
	labelNames, err := swarming.GetLabelNames(dutAttr)
	if err != nil {
		return nil, err
	}

	labelMapping, err := getTleLabelMapping(dutAttr.GetId().GetValue())
	if err != nil {
		logging.Warningf(ctx, "fail to find TLE label mapping: %s", err.Error())
		return nil, nil
	}

	switch labelMapping.configType {
	case dutStateSourceStr:
		return constructTleLabels(labelNames, labelMapping.path, dutState)
	case labConfigSourceStr:
		return constructTleLabels(labelNames, labelMapping.path, lse)
	default:
		return nil, errors.New(fmt.Sprintf("%s is not a valid label source", labelMapping.configType))
	}
}

// constructTleLabels returns label values of a set of label names.
//
// constructTleLabels retrieves label values from a proto message based on a
// given path. For each given label name, a full label in the form of
// `${name}:val1,val2` is constructed and returned as part of an array.
func constructTleLabels(labelNames []string, path string, pm proto.Message) ([]string, error) {
	valuesStr, err := swarming.GetLabelValuesStr(fmt.Sprintf("$.%s", path), pm)
	if err != nil {
		return nil, err
	}
	return swarming.FormLabels(labelNames, valuesStr)
}

// getTleLabelMapping gets the predefined label mapping based on a label name.
func getTleLabelMapping(label string) (*TleSource, error) {
	if val, ok := TLE_LABEL_MAPPING[label]; ok {
		return val, nil
	}
	return nil, status.Errorf(codes.NotFound, "No TLE label mapping found for %s", label)
}
