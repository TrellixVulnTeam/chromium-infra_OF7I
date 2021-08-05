// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package satlab

import (
	"fmt"
	"strings"

	"go.chromium.org/luci/common/errors"

	"infra/cros/cmd/satlab/internal/commands"
	"infra/cros/cmd/satlab/internal/parse"
	"infra/cros/cmd/satlab/internal/tasks"
)

// UseDroneCredential is the stem of a flag (the flag is "-use-drone-credential") that directs
// the satlab tool to grab the credential off of a drone. Later versions of this tool will grab
// a more appropriate service account json file from some container or volume somewhere.
// TODO(gregorynisbet)
const useDroneCredential = "use-drone-credential"

// CommandArities is a map from subcommands to the number of
// components of the subcommand. For example "add" has arity 2
// because "add dut" is a complete command.
//
// TODO(gregorynisbet): Hardcode an arity of 2.
//
var commandArities = map[string]int{
	"add":    2,
	"get":    2,
	"delete": 2,
}

// KnownNullaryFlags is a set of flags that never take an argument
// in any context. One example is the -json flag.
//
// TODO(gregorynisbet): Move this inside the parser.
//
var knownNullaryFlags = map[string]bool{
	"json": true,
}

// Entrypoint takes a list of command line arguments excluding argv[0] and delegates to the appropriate subcommand.
func Entrypoint(args []string) error {
	// TODO(gregorynisbet): Moving parsing into the respective subcommands,
	// even if the implementation is shared.
	parsed, err := parse.ParseCommand(
		args,
		commandArities,
		knownNullaryFlags,
	)
	if err != nil {
		return errors.Annotate(err, "entrypoint").Err()
	}

	// Check for the presence of the --use-drone-credential flag.
	// If the flag is present, get the drone credential and do stuff.
	serviceAccountJSON := ""
	if _, ok := parsed.Flags["use-drone-credential"]; ok {
		var err error
		serviceAccountJSON, err = getServiceAccountJSONPath()
		if err != nil {
			return err
		}
	}

	// Get the name of the satlab DHB.
	dockerHostBoxIdentifier, err := commands.GetDockerHostBoxIdentifier()
	if err != nil {
		return err
	}
	satlabPrefix := fmt.Sprintf("satlab-%s", dockerHostBoxIdentifier)

	mainCmd := ""
	if len(parsed.Commands) >= 1 {
		mainCmd = strings.ToLower(parsed.Commands[0])
	}
	subCmd := ""
	if len(parsed.Commands) >= 2 {
		subCmd = strings.ToLower(parsed.Commands[1])
	}

	switch mainCmd {
	case "add":
		// TODO(gregorynisbet): support more command than just "DUT".
		if subCmd != "dut" {
			return errors.New(fmt.Sprintf("only add dut supported not %q", subCmd))
		}
		return tasks.AddDUT(serviceAccountJSON, satlabPrefix, parsed)
	case "get":
		// TODO(gregorynisbet): support more command than just "DUT".
		if subCmd != "dut" {
			return errors.New(fmt.Sprintf("only get dut supported not %q", subCmd))
		}
		out, err := tasks.GetDUT(serviceAccountJSON, satlabPrefix, parsed)
		if out != "" {
			fmt.Printf("%s\n", out)
		}
		return err
	case "delete":
		// TODO(gregorynisbet): support more command than just "DUT".
		if mainCmd != "dut" {
			return errors.New(fmt.Sprintf("only delete dut supported not %q", subCmd))
		}
		return tasks.DeleteDUT(serviceAccountJSON, satlabPrefix, parsed)
	}

	return errors.Annotate(
		err,
		fmt.Sprintf(
			"unrecognized command: %s",
			strings.ToLower(strings.Join(parsed.Commands, " ")),
		),
	).Err()
}

// GetServiceAccountJSONPath gets a local path on the current machine to service account credentials
//
// Getting credentials for the service account and setting things up is one of the first things that we do;
// it is part of setting up the command since every subsequent call to shivas will use this credential.
func getServiceAccountJSONPath() (string, error) {
	content, err := commands.GetServiceAccountContent()
	if err != nil {
		return "", err
	}
	path, err := commands.MakeTempFile(content)
	if err != nil {
		return "", err
	}
	return path, nil
}
