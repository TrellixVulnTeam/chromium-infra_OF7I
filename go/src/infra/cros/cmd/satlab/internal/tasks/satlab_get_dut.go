// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package tasks

// TODO(gregorynisbet): Validate existence of required flags.

import (
	"fmt"
	"os"
	"os/exec"

	"go.chromium.org/luci/common/errors"

	"infra/cros/cmd/satlab/internal/commands"
	"infra/cros/cmd/satlab/internal/parse"
	"infra/cros/cmd/satlab/internal/paths"
)

// AllFlagMessage is a message telling the use the -all flag in order to list all
// the DUTs. The default behavior of shivas get dut is sort of confusing and takes
// a while.
const allFlagMessage = `to get all DUTs, please use "satlab get dut -all"`

// GetDUT gets information about a DUT.
func GetDUT(serviceAccountJSONPath string, satlabPrefix string, p *parse.CommandParseResult) (string, error) {
	if p == nil {
		return "", errors.New("command parse cannot be nil")
	}

	// 'shivas get dut' will list all DUTs everywhere.
	// This command takes a while to execute and gives no immediate feedback, so provide an error message to the user.
	if len(p.PositionalArgs) == 0 {
		// TODO(gregorynisbet): pick a default behavior for get DUT.
		return "", errors.New(`default "get dut" functionality not implemented`)
	}

	positionalArgs := []string{}
	for _, item := range p.PositionalArgs {
		positionalArgs = append(positionalArgs, fmt.Sprintf("%s-%s", satlabPrefix, item))
	}

	args := (&commands.CommandWithFlags{
		Commands:       []string{paths.ShivasPath, "get", "dut"},
		PositionalArgs: positionalArgs,
	}).ToCommand()
	command := exec.Command(args[0], args[1:]...)
	command.Stderr = os.Stderr
	out, err := command.Output()
	if err != nil {
		return "", errors.Annotate(err, "get dut").Err()
	}
	return string(out), nil
}
