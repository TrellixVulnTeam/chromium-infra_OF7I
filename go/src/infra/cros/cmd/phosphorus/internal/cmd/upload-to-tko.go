// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/maruel/subcommands"
	"github.com/pkg/errors"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/phosphorus"
	"go.chromium.org/luci/common/cli"

	"infra/cros/cmd/phosphorus/internal/autotest"
	"infra/cros/cmd/phosphorus/internal/autotest/atutil"
)

// UploadToTKO subcommand: Parse test results and upload them to TKO.
var UploadToTKO = &subcommands.Command{
	UsageLine: "upload-to-tko -input_json /path/to/input.json",
	ShortDesc: "Parse test results and upload them to TKO.",
	LongDesc: `Parse test results and upload them to TKO.

A wrapper around 'tko/parse'.`,
	CommandRun: func() subcommands.CommandRun {
		c := &uploadToTKORun{}
		c.Flags.StringVar(&c.inputPath, "input_json", "", "Path that contains JSON encoded test_platform.phosphorus.UploadToTkoRequest")
		return c
	},
}

type uploadToTKORun struct {
	commonRun
}

func (c *uploadToTKORun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		fmt.Fprintf(a.GetErr(), "%s\n", err)
		return 1
	}
	return 0
}

func (c *uploadToTKORun) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	var r phosphorus.UploadToTkoRequest
	if err := readJSONPb(c.inputPath, &r); err != nil {
		return err
	}

	if err := validateUploadToTkoRequest(r); err != nil {
		return err
	}

	ctx := cli.GetContext(a, c, env)

	return runTKOUploadStep(ctx, r)
}

func validateUploadToTkoRequest(r phosphorus.UploadToTkoRequest) error {
	missingArgs := getCommonMissingArgs(r.Config)

	if len(missingArgs) > 0 {
		return fmt.Errorf("no %s provided", strings.Join(missingArgs, ", "))
	}

	return nil
}

// runTKOUploadStep extracts test results out of the status.log files
// and uploads them to TKO. It is a wrapper around tko/parse.
func runTKOUploadStep(ctx context.Context, r phosphorus.UploadToTkoRequest) error {
	_, err := atutil.TKOParse(
		autotest.Config{
			AutotestDir: r.Config.Bot.AutotestDir,
		},
		r.Config.Task.ResultsDir,
		os.Stdout)

	if err != nil {
		return errors.Wrap(err, "upload to TKO")
	}

	return nil
}
