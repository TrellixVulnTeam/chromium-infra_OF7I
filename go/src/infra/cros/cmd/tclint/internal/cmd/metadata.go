// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmd

import (
	"context"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"

	"infra/cmdsupport/cmdlib"

	metadata "go.chromium.org/chromiumos/config/go/api/test/metadata/v1"
)

// Metadata subcommand: Lint test metadata.
var Metadata = &subcommands.Command{
	UsageLine: "metadata [FLAGS...] INPUT_FILE_GLOB [INPUT_FILE_GLOB...]",
	ShortDesc: "Lint Chrome OS test metadata specification.",
	LongDesc: `Lint a (complete) specification of a Chrome OS test metadata.

The test metadata must be specified as a metadata.Specification payload as
defined at
https://chromium.googlesource.com/chromiumos/config/+/refs/heads/master/proto/chromiumos/config/api/test/metadata/v1/metadata.proto

The lint includes some global uniqueness checks. Thus, validation may be
incomplete if a partial test metadata specification is provided as input.

The test metadata specification may be split over multiple files and
directories, provided via glob patterns in the positional arguments.`,
	CommandRun: func() subcommands.CommandRun {
		c := &metadataRun{}
		c.Flags.BoolVar(
			&c.binaryFormat,
			"binary",
			false,
			`Decode input protobuf payload from the binary wire format.
By default, input is assumed to be encoded as JSON.`,
		)
		return c
	},
}

type metadataRun struct {
	subcommands.CommandRunBase
	binaryFormat bool
}

// Run implements the subcommands.CommandRun interface.
func (c *metadataRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	ctx := cli.GetContext(a, c, env)
	ctx = logging.SetLevel(ctx, logging.Info)
	if err := c.innerRun(ctx, args); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

type metadataFile struct {
	path    string
	payload *metadata.Specification
	errs    errors.MultiError
}

func (c *metadataRun) innerRun(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return errors.Reason("no input files").Err()
	}
	if _, err := c.load(args); err != nil {
		return err
	}
	return errors.Reason("not implemented").Err()
}

func (c *metadataRun) load(globs []string) ([]*metadataFile, error) {
	resp := make([]*metadataFile, 0, len(globs))
	err := forEachFile(
		globs,
		func(f string) {
			pf := &metadataFile{
				path: f,
			}
			if p, err := c.loadOne(f); err != nil {
				pf.errs = errors.NewMultiError(err)
			} else {
				pf.payload = p
			}
			resp = append(resp, pf)
		},
	)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *metadataRun) loadOne(path string) (*metadata.Specification, error) {
	var p metadata.Specification
	var err error
	if c.binaryFormat {
		err = loadFromBinary(path, &p)
	} else {
		err = loadFromJSON(path, &p)
	}
	return &p, err
}
