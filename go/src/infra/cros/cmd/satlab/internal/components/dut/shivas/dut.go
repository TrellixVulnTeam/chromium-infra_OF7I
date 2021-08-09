// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package shivas

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"go.chromium.org/luci/common/errors"

	"infra/cros/cmd/satlab/internal/commands"
	"infra/cros/cmd/satlab/internal/paths"
)

// DUT contains all the information necessary to add a DUT.
type DUT struct {
	// TODO(gregorynisbet): remove the namespace field.
	Namespace  string
	Zone       string
	Name       string
	Servo      string
	ShivasArgs map[string][]string
}

// Run adds a DUT if it does not already exist.
func (d *DUT) CheckAndUpdate() error {
	dutMsg, err := d.check()
	if err != nil {
		return errors.Annotate(err, "check and update").Err()
	}
	if len(dutMsg) == 0 {
		return d.update()
	} else {
		fmt.Fprintf(os.Stderr, "DUT already added\n")
	}
	return nil
}

// Check checks for the existnce of a UFS DUT.
func (d *DUT) check() (string, error) {
	args := (&commands.CommandWithFlags{
		Commands: []string{paths.ShivasCLI, "get", "dut"},
		Flags: map[string][]string{
			"namespace": {d.Namespace},
			"zone":      {d.Zone},
		},
		PositionalArgs: []string{d.Name},
	}).ToCommand()
	fmt.Fprintf(os.Stderr, "Add dut: run %s\n", args)
	command := exec.Command(args[0], args[1:]...)
	command.Stderr = os.Stderr
	dutMsgBytes, err := command.Output()
	dutMsg := commands.TrimOutput(dutMsgBytes)
	if err != nil {
		return "", errors.Annotate(err, "check DUT in UFS: running %s", strings.Join(args, " ")).Err()
	}
	return dutMsg, nil
}

// Add a DUT to UFS.
func (d *DUT) update() error {
	fmt.Fprintf(os.Stderr, "Adding DUT\n")

	flags := make(map[string][]string)
	for k, v := range d.ShivasArgs {
		flags[k] = v
	}

	flags["name"] = []string{d.Name}
	// This flag must have the form labstation:port.
	// Do not validate this flag here since we don't want to potentially drift
	// out of sync with the format that shivas expects.
	// TODO(gregorynisbet): Consider pre-populating it.
	flags["servo"] = []string{d.Servo}

	// TODO(gregorynisbet): Consider a different strategy for tracking flags
	// that cannot be passed to shivas add dut.
	args := (&commands.CommandWithFlags{
		Commands: []string{paths.ShivasCLI, "add", "dut"},
		Flags:    flags,
	}).ToCommand()
	fmt.Fprintf(os.Stderr, "Add dut: run %s\n", args)
	command := exec.Command(args[0], args[1:]...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	err := command.Run()
	return errors.Annotate(
		err,
		fmt.Sprintf(
			"add dut: running %s",
			strings.Join(args, " "),
		),
	).Err()
}
