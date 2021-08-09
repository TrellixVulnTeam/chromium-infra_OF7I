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

// Rack is a group of arguments for adding a rack.
type Rack struct {
	Name      string
	Namespace string
	Zone      string
}

// CheckAndUpdate runs check and then update if the item does not exist.
func (r *Rack) CheckAndUpdate() error {
	rackMsg, err := r.check()
	if err != nil {
		return errors.Annotate(err, "check and update").Err()
	}
	if len(rackMsg) == 0 {
		return r.update()
	} else {
		fmt.Fprintf(os.Stderr, "Rack already added\n")
	}
	return nil
}

// Check checks if a rack exists.
func (r *Rack) check() (string, error) {
	args := (&commands.CommandWithFlags{
		Commands:       []string{paths.ShivasCLI, "get", "rack"},
		PositionalArgs: []string{r.Name},
		Flags: map[string][]string{
			"namespace": {r.Namespace},
		},
	}).ToCommand()
	fmt.Fprintf(os.Stderr, "Add rack: run %s\n", args)
	command := exec.Command(args[0], args[1:]...)
	command.Stderr = os.Stderr
	rackMsgBytes, err := command.Output()
	rackMsg := commands.TrimOutput(rackMsgBytes)
	if err != nil {
		return "", errors.Annotate(err, "add rack").Err()
	}
	return rackMsg, nil
}

// Update adds a rack unconditionally to UFS.
func (r *Rack) update() error {
	fmt.Fprintf(os.Stderr, "Adding rack\n")
	args := (&commands.CommandWithFlags{
		Commands: []string{paths.ShivasCLI, "add", "rack"},
		Flags: map[string][]string{
			// TODO(gregorynisbet): Default to OS for everything.
			"namespace": {r.Namespace},
			"name":      {r.Name},
			"zone":      {r.Zone},
		},
	}).ToCommand()
	fmt.Fprintf(os.Stderr, "Add rack: run %s\n", args)
	command := exec.Command(args[0], args[1:]...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	err := command.Run()
	return errors.Annotate(err, "add rack").Err()
}
