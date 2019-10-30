// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmd

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"

	sv "go.chromium.org/chromiumos/infra/proto/go/lab_platform"
	gslib "infra/cmd/stable_version2/internal/gs"
	"infra/cmd/stable_version2/internal/site"
)

// UpdateWithOmaha subcommand: read stable version in omaha json file in GS.
var UpdateWithOmaha = &subcommands.Command{
	UsageLine: `update-with-omaha [FLAGS...] -output_json /path/to/output.json`,
	ShortDesc: "update stable version with omaha files",
	LongDesc: `update stable vesrion with omaha json file in GS.

This command is for builder to get up-to-date stable version from omaha file in GS,
and commit them to stable version config file.
Do not use this command as part of scripts or pipelines as it's unstable.

Output is JSON encoded protobuf defined at
https://chromium.googlesource.com/chromiumos/infra/proto/+/refs/heads/master/src/lab_platform/stable_version.proto`,
	CommandRun: func() subcommands.CommandRun {
		c := &updateWithOmahaRun{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.Flags.StringVar(&c.outputPath, "output_json", "", "Path where JSON encoded lab_platform.StableVersions should be written.")

		return c
	},
}

type updateWithOmahaRun struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags

	outputPath string
}

// Run implements the subcommands.CommandRun interface.
func (c *updateWithOmahaRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		printError(a.GetErr(), err)
		return 1
	}
	return 0
}

func (c *updateWithOmahaRun) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	ctx := cli.GetContext(a, c, env)
	ctx = setupLogging(ctx)
	f := &c.authFlags

	outDir, err := ioutil.TempDir("", programName)
	if err != nil {
		return err
	}
	defer func() {
		if err := os.RemoveAll(outDir); err != nil {
			logging.Errorf(ctx, "fail to remove temp dir: %s", err)
		}
	}()

	t, err := newAuthenticatedTransport(ctx, f)
	if err != nil {
		return errors.Annotate(err, "create authenticated transport").Err()
	}
	var gsc gslib.Client
	if err := gsc.Init(ctx, t, unmarshaller); err != nil {
		return err
	}

	newCrosSV, err := getGSCrosSV(ctx, outDir, gsc)
	if err != nil {
		return err
	}

	// TODO(xixuan): Add more following logics.
	if c.outputPath == "" {
		for _, u := range newCrosSV {
			logging.Debugf(ctx, "%v", u)
		}
	}
	return nil
}

// Get CrOS stable version from omaha status file.
func getGSCrosSV(ctx context.Context, outDir string, gsc gslib.Client) ([]*sv.StableCrosVersion, error) {
	localOSFile := filepath.Join(outDir, omahaStatusFile)
	if err := gsc.Download(omahaGsPath, localOSFile); err != nil {
		return nil, err
	}
	omahaBytes, err := ioutil.ReadFile(localOSFile)
	if err != nil {
		return nil, errors.Annotate(err, "load omaha").Err()
	}
	cros, err := gsc.ParseOmahaStatus(ctx, omahaBytes)
	if err != nil {
		return nil, errors.Annotate(err, "parse omaha").Err()
	}
	return cros, nil
}
