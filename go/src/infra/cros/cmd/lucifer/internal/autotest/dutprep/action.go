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
type Action int

const (
	// NoAction can be used as a null Action value.
	NoAction Action = iota
	// StageUSB action
	StageUSB
	// InstallTestImage action
	InstallTestImage
	// InstallFirmware action
	InstallFirmware
	// RunPreDeployVerification action
	RunPreDeployVerification
	// VerifyRecoveryMode action
	VerifyRecoveryMode
	// SetupLabstation action
	SetupLabstation
	// UpdateLabel action
	UpdateLabel
)

//go:generate stringer -type=Action

// ParseAction parses the string argument accepted by lucifer and autotest
// tools into an Action.
func ParseAction(s string) (Action, error) {
	for i, n := range actionArgumentSequence {
		if n == s {
			return actionSequence[i], nil
		}
	}
	return NoAction, fmt.Errorf("unknown action %s", s)
}

// Arg returns the argument string used by lucifer and autotest tools for an
// action.
func (a Action) Arg() string {
	for i, o := range actionSequence {
		if o == a {
			return actionArgumentSequence[i]
		}
	}
	return ""
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
// Keep actionSequence and actionArgumentSequence in-sync as they're used to
// translate the actions to and from arguments accepted by lucifer and autotest
// tools.
var actionSequence = [...]Action{
	StageUSB,
	InstallTestImage,
	InstallFirmware,
	RunPreDeployVerification,
	VerifyRecoveryMode,
	SetupLabstation,
	UpdateLabel,
}
var actionArgumentSequence = [...]string{
	"stage-usb",
	"install-test-image",
	"install-firmware",
	"run-pre-deploy-verification",
	"verify-recovery-mode",
	"setup-labstation",
	"update-label",
}

func containsAction(as []Action, q Action) bool {
	for _, a := range as {
		if q == a {
			return true
		}
	}
	return false
}
