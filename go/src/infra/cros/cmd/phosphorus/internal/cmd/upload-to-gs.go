// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmd

import (
	"context"
	"fmt"
	"infra/cros/cmd/phosphorus/internal/gs"
	"strings"

	"github.com/maruel/subcommands"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/phosphorus"
	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"
)

// UploadToGS subcommand: Upload selected directory to Google Storage.
func UploadToGS(authOpts auth.Options) *subcommands.Command {
	return &subcommands.Command{
		UsageLine: "upload-to-gs -input_json /path/to/input.json -output_json /path/to/output.json",
		ShortDesc: "Upload files to Google Storage.",
		LongDesc:  `Upload files to Google Storage. A wrapper around 'gsutil'.`,
		CommandRun: func() subcommands.CommandRun {
			c := &uploadToGSRun{}
			c.Flags.StringVar(&c.inputPath, "input_json", "", "Path that contains JSON encoded test_platform.phosphorus.UploadToGSRequest")
			c.Flags.StringVar(&c.outputPath, "output_json", "", "Path that will contain JSON encoded test_platform.phosphorus.UploadToGSResponse")
			c.authFlags.Register(&c.Flags, authOpts)
			return c
		},
	}
}

type uploadToGSRun struct {
	commonRun
}

func (c *uploadToGSRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		fmt.Fprintf(a.GetErr(), "%s\n", err)
		return 1
	}
	return 0
}

func (c *uploadToGSRun) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	var r phosphorus.UploadToGSRequest
	if err := readJSONPb(c.inputPath, &r); err != nil {
		return err
	}

	if err := validateUploadToGSRequest(r); err != nil {
		return err
	}

	ctx := cli.GetContext(a, c, env)

	path, err := runGSUploadStep(ctx, c.authFlags, r)
	if err != nil {
		return err
	}

	out := phosphorus.UploadToGSResponse{
		GsUrl: path,
	}
	if err = writeJSONPb(c.outputPath, &out); err != nil {
		return err
	}
	return nil

}

func validateUploadToGSRequest(r phosphorus.UploadToGSRequest) error {
	missingArgs := make([]string, 0)
	if r.GetConfig().GetTask().GetSynchronousOffloadDir() == "" {
		missingArgs = append(missingArgs, "offload dir")
	}

	if r.GetGsDirectory() == "" {
		missingArgs = append(missingArgs, "GS directory")
	}

	if len(missingArgs) > 0 {
		return errors.Reason("no %s provided", strings.Join(missingArgs, ", ")).Err()
	}

	return nil
}

// runGSUploadStep uploads all files in the specified directory to GS.
func runGSUploadStep(ctx context.Context, authFlags authcli.Flags, r phosphorus.UploadToGSRequest) (string, error) {
	gsPath := gs.Concat(r.GetGsDirectory(), "synchronous_offloads", r.GetTaskId())
	localPath := r.GetConfig().GetTask().GetSynchronousOffloadDir()
	w, err := createDirWriter(ctx, localPath, gsPath, &authFlags)
	if err != nil {
		return "", err
	}
	if err = w.WriteDir(); err != nil {
		return "", err
	}
	return string(gsPath), nil
}

// createDirWriter creates a DirWriter for the given paths, first producing an authed client with the given context and flags
func createDirWriter(ctx context.Context, localPath string, gsPath gs.Path, authFlags *authcli.Flags) (gs.DirWriter, error) {
	cli, err := gs.NewAuthedClient(ctx, authFlags)
	if err != nil {
		return nil, err
	}
	return gs.NewDirWriter(localPath, gsPath, cli)
}
