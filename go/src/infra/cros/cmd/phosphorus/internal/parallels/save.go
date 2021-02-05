// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package parallels

import (
	"context"
	"fmt"
	"strings"

	"infra/cros/cmd/phosphorus/internal/cmd"
	"infra/cros/cmd/phosphorus/internal/skylab_local_state/ufs"

	"github.com/maruel/subcommands"
	"go.chromium.org/chromiumos/infra/proto/go/uprev/build_parallels_image"
	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"
)

// Save subcommand: Saves DUT state.
func Save(authOpts auth.Options) *subcommands.Command {
	return &subcommands.Command{
		UsageLine: "build-parallels-image-save -input_json /path/to/input.json",
		ShortDesc: "Update the DUT state in UFS.",
		LongDesc:  `Update the state of a DUT in Unified Fleet System. For use in build_parallels_image.`,
		CommandRun: func() subcommands.CommandRun {
			c := &saveRun{}

			c.AuthFlags.Register(&c.Flags, authOpts)

			c.Flags.StringVar(&c.InputPath, "input_json", "", "Path that contains JSON encoded engprod.build_parallels_image.SaveRequest")
			return c
		},
	}
}

type saveRun struct {
	cmd.CommonRun
}

func (c *saveRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	ctx := cli.GetContext(a, c, env)
	if err := c.ValidateArgs(); err != nil {
		errors.Log(ctx, err)
		c.Flags.Usage()
		return 1
	}

	if err := c.innerRun(ctx, env); err != nil {
		errors.Log(ctx, err)
		return 2
	}
	return 0
}

func (c *saveRun) innerRun(ctx context.Context, env subcommands.Env) error {
	r := &build_parallels_image.SaveRequest{}
	if err := cmd.ReadJSONPB(c.InputPath, r); err != nil {
		return err
	}
	if err := validateSaveRequest(r); err != nil {
		return err
	}
	if err := ufs.SafeUpdateUFSDUTState(ctx, &c.AuthFlags, r.DutName, r.DutState, r.Config.CrosUfsService); err != nil {
		return err
	}
	return nil
}

func validateSaveRequest(r *build_parallels_image.SaveRequest) error {
	missingArgs := validateConfig(r.GetConfig())

	if r.DutName == "" {
		missingArgs = append(missingArgs, "DUT hostname")
	}
	if r.DutState == "" {
		missingArgs = append(missingArgs, "DUT state")
	}
	if len(missingArgs) > 0 {
		return fmt.Errorf("no %s provided", strings.Join(missingArgs, ", "))
	}
	return nil
}
