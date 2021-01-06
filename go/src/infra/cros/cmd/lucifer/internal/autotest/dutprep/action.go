// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

/*
Package dutprep contains functions helpful in interaction with the Autotest DUT
preparation tool.
*/
package dutprep

import (
	"fmt"
)

// An Action indicates the DUT preparation action to do.
type Action string

const (
	// NoAction can be used as a null Action value.
	NoAction Action = "no-action"
	// StageUSB action
	StageUSB Action = "stage-usb"
	// InstallTestImage action
	InstallTestImage Action = "install-test-image"
	// InstallFirmware action
	InstallFirmware Action = "install-firmware"
	// VerifyRecoveryMode action
	VerifyRecoveryMode Action = "verify-recovery-mode"
	// SetupLabstation action
	SetupLabstation Action = "setup-labstation"
	// UpdateLabel action
	UpdateLabel Action = "update-label"
	// RunPreDeployVerification action
	RunPreDeployVerification Action = "run-pre-deploy-verification"
)

// ParseAction parses the string argument accepted by lucifer and autotest
// tools into an Action.
func ParseAction(s string) (Action, error) {
	a := Action(s)
	for _, n := range actionSequence {
		if n == a {
			return a, nil
		}
	}
	return NoAction, fmt.Errorf("unknown action %s", s)
}

// String returns the string representation of action.
func (a Action) String() string {
	return string(a)
}

// SortActions sorts the given Action slice in the order they should be
// executed to prepare a host.
func SortActions(actions []Action) []Action {
	sorted := make([]Action, 0, len(actions))
	for _, a := range actionSequence {
		if containsAction(actions, a) {
			sorted = append(sorted, a)
		}
	}
	return sorted
}

// actionSequence orders the actions as they should be executed on a host.
var actionSequence = [...]Action{
	StageUSB,
	InstallTestImage,
	InstallFirmware,
	VerifyRecoveryMode,
	SetupLabstation,
	UpdateLabel,
	RunPreDeployVerification,
}

func containsAction(as []Action, q Action) bool {
	for _, a := range as {
		if q == a {
			return true
		}
	}
	return false
}
