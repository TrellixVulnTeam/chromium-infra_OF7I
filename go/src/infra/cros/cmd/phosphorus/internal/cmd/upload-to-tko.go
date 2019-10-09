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
	UsageLine: "upload-to-tko",
	ShortDesc: "Parse test results and upload them to TKO.",
	LongDesc: `Parse test results and upload them to TKO.

A wrapper around 'tko/parse'.`,
	CommandRun: func() subcommands.CommandRun {
		c := &uploadToTKORun{}
		c.Flags.StringVar(&c.inputPath, "input_json", "", "Path that contains JSON encoded test_platform.phosphorus.Config")
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
	var pc phosphorus.Config
	if err := readJSONPb(c.inputPath, &pc); err != nil {
		return err
	}

	if err := validateRequestConfig(pc); err != nil {
		return err
	}

	ctx := cli.GetContext(a, c, env)

	return runTKOUploadStep(ctx, pc)
}

func validateRequestConfig(pc phosphorus.Config) error {
	missingArgs := getCommonMissingArgs(&pc)

	if len(missingArgs) > 0 {
		return fmt.Errorf("no %s provided", strings.Join(missingArgs, ", "))
	}

	return nil
}

// runTKOUploadStep extracts test results out of the status.log files
// and uploads them to TKO. It is a wrapper around tko/parse.
func runTKOUploadStep(ctx context.Context, pc phosphorus.Config) error {
	_, err := atutil.TKOParse(
		autotest.Config{
			AutotestDir: pc.Bot.AutotestDir,
		},
		pc.Task.ResultsDir,
		3, // "SKYLAB_PROVISION"
		os.Stdout)

	if err != nil {
		return errors.Wrap(err, "upload to TKO")
	}

	return nil
}
