// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmd

import (
	"bytes"
	"fmt"
	"io/ioutil"

	"github.com/golang/protobuf/jsonpb"
	"github.com/maruel/subcommands"

	"infra/cmd/skylab/internal/cmd/cmdlib"
	"infra/libs/skylab/inventory"
)

// ValidateNewDutJSON -- take a path and validate the json file associated with it.
var ValidateNewDutJSON = &subcommands.Command{
	UsageLine: "validate-new-dut-json <path>...",
	ShortDesc: "validate json for a new DUT",
	LongDesc:  `validate json for a new DUT.`,
	CommandRun: func() subcommands.CommandRun {
		c := &validateNewDutJSONRun{}
		return c
	},
}

type validateNewDutJSONRun struct {
	subcommands.CommandRunBase
}

func (c *validateNewDutJSONRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	malformedTally := 0
	for _, path := range args {
		contents, err := ioutil.ReadFile(path)
		if err != nil {
			fmt.Fprintf(a.GetErr(), "error validating (%s)\n", path)
			cmdlib.PrintError(a.GetErr(), err)
			return 1
		}
		var spec inventory.DeviceUnderTest
		if err := jsonpb.Unmarshal(bytes.NewReader(contents), &spec); err != nil {
			fmt.Fprintf(a.GetOut(), "BAD\t%s\n", path)
			cmdlib.PrintError(a.GetErr(), err)
			malformedTally++
			continue
		}
		fmt.Fprintf(a.GetOut(), "GOOD\t%s\n", path)
	}
	if malformedTally > 0 {
		return 2
	}
	return 0
}
