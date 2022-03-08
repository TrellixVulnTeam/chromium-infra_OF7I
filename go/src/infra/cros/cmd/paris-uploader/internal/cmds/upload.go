// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmds

import (
	"context"
	"fmt"
	"time"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"
	lucigs "go.chromium.org/luci/common/gcloud/gs"

	"infra/cros/cmd/paris-uploader/site"
	"infra/cros/recovery/upload"
)

// UploadCmd uploads a folder to Google Storage.
var UploadCmd = &subcommands.Command{
	UsageLine: `upload`,
	ShortDesc: `Upload a folder to Google Storage`,
	CommandRun: func() subcommands.CommandRun {
		r := &uploadCmdRun{}
		r.authFlags.Register(&r.Flags, site.DefaultAuthOptions)

		r.Flags.StringVar(&r.path, "path", "", "local path")
		r.Flags.StringVar(&r.gs, "gs", "", "gs path")
		r.Flags.IntVar(&r.workers, "workers", 8, "max worker processes")

		return r
	},
}

// UploadCmdRun holds the arguments for uploadCmd.
type uploadCmdRun struct {
	subcommands.CommandRunBase

	authFlags authcli.Flags

	path    string
	gs      string
	workers int
}

// Run is the main entrypoint for the upload command.
func (c *uploadCmdRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	ctx := cli.GetContext(a, c, env)
	if err := c.innerRun(ctx, a, args, env); err != nil {
		fmt.Fprintf(a.GetErr(), "%s: %s\n", a.GetName(), err)
		return 1
	}
	return 0
}

// InnerRun is the implementation for the upload command.
func (c *uploadCmdRun) innerRun(ctx context.Context, a subcommands.Application, args []string, env subcommands.Env) error {
	authOpts, err := c.authFlags.Options()
	if err != nil {
		return errors.Annotate(err, "upload").Err()
	}
	authenticator := auth.NewAuthenticator(ctx, auth.SilentLogin, authOpts)
	roundTripper, err := authenticator.Transport()
	if err != nil {
		return errors.Annotate(err, "upload").Err()
	}
	gsClient, err := lucigs.NewProdClient(ctx, roundTripper)
	if err != nil {
		return errors.Annotate(err, "upload").Err()
	}

	ctx, cancel := context.WithTimeout(ctx, time.Hour)
	defer cancel()
	if err := upload.Upload(ctx, gsClient, &upload.Params{
		SourceDir:         c.path,
		GSURL:             c.gs,
		MaxConcurrentJobs: c.workers,
	}); err != nil {
		return errors.Annotate(err, "upload").Err()
	}
	return nil
}
