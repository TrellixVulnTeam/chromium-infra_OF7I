// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package shivas

import (
	"fmt"
	"os"
	"os/exec"

	"go.chromium.org/luci/common/errors"

	"infra/cros/cmd/satlab/internal/commands"
	"infra/cros/cmd/satlab/internal/paths"
)

// Asset is a group of parameters needed to add an asset to UFS.
type Asset struct {
	Asset     string
	Rack      string
	Zone      string
	Model     string
	Board     string
	Namespace string
	Type      string
}

// CheckAndUpdate adds the asset if it does not already exist.
func (a *Asset) CheckAndUpdate() error {
	assetMsg, err := a.check()
	if err != nil {
		return errors.Annotate(err, "check and update").Err()
	}
	if len(assetMsg) == 0 {
		return a.update()
	} else {
		fmt.Fprintf(os.Stderr, "Asset already added\n")
	}
	return nil
}

// Check checks for the existence of the UFS asset.
func (a *Asset) check() (string, error) {
	args := (&commands.CommandWithFlags{
		Commands:       []string{paths.ShivasCLI, "get", "asset"},
		PositionalArgs: []string{a.Asset},
		Flags: map[string][]string{
			"rack":      {a.Rack},
			"zone":      {a.Zone},
			"model":     {a.Model},
			"board":     {a.Board},
			"namespace": {a.Namespace},
			// Type cannot be provided when getting a DUT.
		},
	}).ToCommand()
	fmt.Fprintf(os.Stderr, "Add asset: run %s\n", args)
	command := exec.Command(args[0], args[1:]...)
	command.Stderr = os.Stderr
	assetMsgBytes, err := command.Output()
	assetMsg := commands.TrimOutput(assetMsgBytes)
	if err != nil {
		return "", errors.Annotate(err, "add asset").Err()
	}
	return assetMsg, nil
}

// Update adds an asset unconditionally to UFS.
func (a *Asset) update() error {
	// Add the asset.
	fmt.Fprintf(os.Stderr, "Adding asset\n")
	args := (&commands.CommandWithFlags{
		Commands: []string{paths.ShivasCLI, "add", "asset"},
		Flags: map[string][]string{
			"model":     {a.Model},
			"board":     {a.Board},
			"rack":      {a.Rack},
			"zone":      {a.Zone},
			"name":      {a.Asset},
			"namespace": {a.Namespace},
			"type":      {a.Type},
		},
	}).ToCommand()
	fmt.Fprintf(os.Stderr, "Add asset: run %s\n", args)
	command := exec.Command(args[0], args[1:]...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	err := command.Run()
	return errors.Annotate(err, "add asset").Err()
}
