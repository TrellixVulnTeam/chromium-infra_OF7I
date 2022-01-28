// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cros

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/execs"
)

// example of output from "chromeos-firmwareupdate --manifest"
//
// {
//  "model_name": {
//   "host": { "versions": { "ro": "Google_Rammus.xxxx", "rw": "Google_Rammus.xxxx" },
//      "keys": { "root": "xxxx", "recovery": "xxxx" },
//      "image": "images/bios-rammus.xxxx" },
//    "ec": { "versions": { "ro": "rammus_v2.0.xxxxx", "rw": "rammus_v2.0.xxxx" },
//      "image": "images/ec-rammus.xxxx" },
//    "signature_id": "model_name"
//  }
// }
type versions struct {
	Rw string
	Ro string
}

type keys struct {
	Root     string
	Recovery string
}

type host struct {
	Versions *versions
	Keys     *keys
	Image    string
}

type modelFirmware struct {
	Host         *host
	Ec           *host
	Signature_ID string
}

// AvailableRWFirmware returns the available rw firmware info inside this DUT's modelFirmware struct.
func (mf *modelFirmware) AvailableRWFirmware() (string, error) {
	host := mf.Host
	if host == nil {
		return "", errors.Reason("available rw firmware: host info is not present").Err()
	}
	versions := host.Versions
	if versions == nil {
		return "", errors.Reason("available rw firmware: no versions found").Err()
	}
	rw := versions.Rw
	if rw == "" {
		return "", errors.Reason("available rw firmware: rw version is not provided").Err()
	}
	return strings.TrimSpace(rw), nil
}

const (
	firmwareUpdateManifestCmd = `chromeos-firmwareupdate --manifest`
)

// ReadFirmwareManifest reads the firmware update manifest info and return a modelFirmware struct.
func ReadFirmwareManifest(ctx context.Context, r execs.Runner, dutModel string) (*modelFirmware, error) {
	rawOutput, err := r(ctx, time.Minute, firmwareUpdateManifestCmd)
	if err != nil {
		return nil, errors.Annotate(err, "read firmware manifest").Err()
	}
	firmwareUpdateResultMap := make(map[string]modelFirmware)
	if err := json.Unmarshal([]byte(rawOutput), &firmwareUpdateResultMap); err != nil {
		return nil, errors.Annotate(err, "read firmware manifest").Err()
	}
	modelFw, ok := firmwareUpdateResultMap[dutModel]
	if !ok {
		return nil, errors.Reason("read firmware manifest: model %q is not present in manifest", dutModel).Err()
	}
	return &modelFw, nil
}
