// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cli

import (
	"fmt"
	"infra/tools/dirmd"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/common/data/text"
)

func cmdValidate() *subcommands.Command {
	return &subcommands.Command{
		UsageLine: `validate [file1 [file2]...]`,
		ShortDesc: "validate metadata files",
		LongDesc: text.Doc(`
			Validate metadata files.

			The positional arguments must be paths to the files.
			A valid file has a base filename "DIR_METADATA" or "OWNERS".
			The format of its contents correspond to the base name.

			The subcommand returns a non-zero exit code if any of the files is
			invalid.
		`),
		CommandRun: func() subcommands.CommandRun {
			return &validateRun{}
		},
	}
}

type validateRun struct {
	baseCommandRun
}

func (r *validateRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	exitCode := 0
	for _, fileName := range args {
		if err := dirmd.ValidateFile(fileName); err != nil {
			fmt.Printf("%s: %s\n", fileName, err)
			exitCode = 1
		} else {
			fmt.Printf("%s: valid\n", fileName)
		}
	}
	return exitCode
}
