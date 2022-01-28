// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cros

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/execs"
)

const (
	// crasAudioNodesQueryCmd query nodes information from Cras audio utilities using dbus-send.
	crasAudioNodesQueryCmd = "dbus-send --system --type=method_call --print-reply " +
		"--dest=org.chromium.cras /org/chromium/cras " +
		"org.chromium.cras.Control.GetNodes"
	// findCrasAudioNodeTypeRegexp is the regular expression that matches the string in the format of:
	// Ex:
	// string "Type"
	// variant             string "INTERNAL_MIC"
	findCrasAudioNodeTypeRegexp = `string "Type"\s+variant\s+string "%s"`
	// CrasAudioHeadphone is the string value for finding the nodeType of audio headphone
	// in the output of the crasAudioNodesQueryCmd.
	CrasAudioHeadphone = "HEADPHONE"
	// CrasAudioMic is the string value for finding the nodeType of audio mic
	// in the output of the crasAudioNodesQueryCmd.
	CrasAudioMic = "MIC"
)

// CrasAudioNodeTypeIsPlugged finds if the specified audio type node is present
// in the list of all plugged Cras Audio Nodes.
//
// Example of the type "INTERNAL_MIC" Cras Audio Nodes present on the DUT:
//
// dict entry(
//	string "Type"
//	variant             string "INTERNAL_MIC"
// )
//
// @param nodeType : A string representing Cras Audio Node Type
// @returns:
//	if the err == nil, the boolean value returned represents whether the given audio node type is found in the output.
//	if the err != nil, the execution of this function is not successful, the boolean value returned is set as default.
func CrasAudioNodeTypeIsPlugged(ctx context.Context, r execs.Runner, nodeType string) (bool, error) {
	output, err := r(ctx, time.Minute, crasAudioNodesQueryCmd)
	if err != nil {
		return false, errors.Annotate(err, "node type of %s is plugged", nodeType).Err()
	}
	nodeTypeRegexp, err := regexp.Compile(fmt.Sprintf(findCrasAudioNodeTypeRegexp, nodeType))
	if err != nil {
		return false, errors.Annotate(err, "node type of %s is plugged", nodeType).Err()
	}
	nodeTypeExist := nodeTypeRegexp.MatchString(output)
	return nodeTypeExist, nil
}
