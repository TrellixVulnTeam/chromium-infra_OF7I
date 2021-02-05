// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/maruel/subcommands"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/phosphorus"
	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"
	gcgs "go.chromium.org/luci/common/gcloud/gs"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/lucictx"

	"infra/cros/cmd/phosphorus/internal/gs"
)

// UploadToGS subcommand: Upload selected directory to Google Storage.
func UploadToGS(authOpts auth.Options) *subcommands.Command {
	return &subcommands.Command{
		UsageLine: "upload-to-gs -input_json /path/to/input.json -output_json /path/to/output.json",
		ShortDesc: "Upload files to Google Storage.",
		LongDesc:  `Upload files to Google Storage. A wrapper around 'gsutil'.`,
		CommandRun: func() subcommands.CommandRun {
			c := &uploadToGSRun{}
			c.Flags.StringVar(&c.InputPath, "input_json", "", "Path that contains JSON encoded test_platform.phosphorus.UploadToGSRequest")
			c.Flags.StringVar(&c.OutputPath, "output_json", "", "Path that will contain JSON encoded test_platform.phosphorus.UploadToGSResponse")
			c.AuthFlags.Register(&c.Flags, authOpts)
			return c
		},
	}
}

type uploadToGSRun struct {
	CommonRun
}

func (c *uploadToGSRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	ctx := cli.GetContext(a, c, env)
	if err := c.innerRun(ctx, args, env); err != nil {
		logApplicationError(ctx, a, err)
		return 1
	}
	return 0
}

func (c *uploadToGSRun) innerRun(ctx context.Context, args []string, env subcommands.Env) error {
	r := &phosphorus.UploadToGSRequest{}
	if err := ReadJSONPB(c.InputPath, r); err != nil {
		return err
	}
	if err := validateUploadToGSRequest(r); err != nil {
		return err
	}
	ctx, err := useSystemAuth(ctx, &c.AuthFlags)
	if err != nil {
		return err
	}
	path, err := runGSUploadStep(ctx, c.AuthFlags, r)
	if err != nil {
		return err
	}
	out := phosphorus.UploadToGSResponse{
		GsUrl: path,
	}
	if err = WriteJSONPB(c.OutputPath, &out); err != nil {
		return err
	}
	return nil

}

func validateUploadToGSRequest(r *phosphorus.UploadToGSRequest) error {
	missingArgs := make([]string, 0)
	if r.GetLocalDirectory() == "" {
		missingArgs = append(missingArgs, "local_directory")
	}
	if r.GetGsDirectory() == "" {
		missingArgs = append(missingArgs, "gs_directory")
	}

	if len(missingArgs) > 0 {
		return errors.Reason("missing arguments: %s", strings.Join(missingArgs, ", ")).Err()
	}
	return nil
}

// TODO(crbug.com/1133890): Replace with value from builder config.
const maxConcurrentUploads = 20

// runGSUploadStep uploads all files in the specified directory to GS.
func runGSUploadStep(ctx context.Context, authFlags authcli.Flags, r *phosphorus.UploadToGSRequest) (string, error) {
	localPath := r.GetLocalDirectory()
	path := gcgs.Path(r.GetGsDirectory())

	gsC, err := newGSClient(ctx, &authFlags)
	if err != nil {
		return "", err
	}
	w := gs.NewDirWriter(gsC, maxConcurrentUploads)

	// TODO(crbug.com/1130071) Set timeout from the recipe.
	// Hard-coded here to stop the bleeding fast.
	wCtx, cancel := context.WithTimeout(ctx, time.Hour)
	defer cancel()

	if err = w.WriteDir(wCtx, localPath, path); err != nil {
		logging.Debugf(ctx, "Directory listing for failed upload: %s", dirList(localPath))
		return "", err
	}
	logging.Infof(ctx, "All files uploaded.")
	return r.GetGsDirectory(), nil
}

func useSystemAuth(ctx context.Context, authFlags *authcli.Flags) (context.Context, error) {
	authOpts, err := authFlags.Options()
	if err != nil {
		return nil, errors.Annotate(err, "switching to system auth").Err()
	}

	authCtx, err := lucictx.SwitchLocalAccount(ctx, "system")
	if err == nil {
		// If there's a system account use it (the case of running on Swarming).
		// Otherwise default to user credentials (the local development case).
		authOpts.Method = auth.LUCIContextMethod
		return authCtx, nil
	}
	logging.Warningf(ctx, "System account not found, err %s.\nFalling back to user credentials for auth.\n", err)
	return ctx, nil
}

// Gives a list of files and directories under the given path. Intended for
// debugging a failed tempdir removal.
func dirList(absPath string) string {
	ch := make(chan string)
	files := make([]string, 0)
	ctx, can := context.WithCancel(context.Background())
	defer can()
	go func() {
		for {
			select {
			case f := <-ch:
				files = append(files, f)
			case <-ctx.Done():
				return
			}
		}
	}()
	fileName := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			ch <- fmt.Sprintf("err %s while checking path %s", err, path)
		} else {
			ch <- path
		}
		return err
	}
	_ = filepath.Walk(absPath, fileName)
	return strings.Join(files, ", ")
}

func newGSClient(ctx context.Context, f *authcli.Flags) (gcgs.Client, error) {
	o, err := f.Options()
	if err != nil {
		return nil, errors.Annotate(err, "new Google Storage client").Err()
	}
	a := auth.NewAuthenticator(ctx, auth.SilentLogin, o)
	rt, err := a.Transport()
	if err != nil {
		return nil, errors.Annotate(err, "new Google Storage client").Err()
	}
	cli, err := gcgs.NewProdClient(ctx, rt)
	if err != nil {
		return nil, errors.Annotate(err, "new Google Storage client").Err()
	}
	return cli, nil
}
