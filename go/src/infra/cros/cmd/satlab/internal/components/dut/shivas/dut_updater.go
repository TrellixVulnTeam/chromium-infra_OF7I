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

// DUTUpdater updates a DUT with the given name.
type DUTUpdater struct {
	Name       string
	ShivasArgs map[string][]string
}

// Update invokes shivas with the required arguments to update information
// about a DUT.
func (u *DUTUpdater) Update() error {
	flags := make(map[string][]string)

	for k, v := range u.ShivasArgs {
		flags[k] = v
	}

	flags["name"] = []string{u.Name}

	args := (&commands.CommandWithFlags{
		Commands: []string{paths.ShivasCLI, "update", "dut"},
		Flags:    flags,
	}).ToCommand()
	fmt.Fprintf(os.Stderr, "Update dut: run %s\n", args)
	command := exec.Command(args[0], args[1:]...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	err := command.Run()
	return errors.Annotate(
		err,
		fmt.Sprintf(
			"update dut: running %s",
			strings.Join(args, " "),
		),
	).Err()
}
