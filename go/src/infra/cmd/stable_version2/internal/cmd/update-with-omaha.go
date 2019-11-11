// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"

	sv "go.chromium.org/chromiumos/infra/proto/go/lab_platform"
	gslib "infra/cmd/stable_version2/internal/gs"
	"infra/cmd/stable_version2/internal/site"
	"infra/cmd/stable_version2/internal/utils"
	svlib "infra/libs/cros/stableversion"
	gitlib "infra/libs/cros/stableversion/git"
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
	if err := gsc.Init(ctx, t, utils.Unmarshaller); err != nil {
		return err
	}

	// Fetch up-to-date stable version based on omaha file
	newCrosSV, err := getGSCrosSV(ctx, outDir, gsc)
	if err != nil {
		return err
	}

	// Fetch existing stable version
	hc, err := newHTTPClient(ctx, f)
	if err != nil {
		return err
	}
	gc, err := gitlib.NewClient(ctx, hc, gerritHost, gitilesHost, project, branch)
	if err != nil {
		return err
	}

	oldSV, err := getGitSV(ctx, gc)
	logInvalidCrosSV(ctx, oldSV.GetCros())
	if err != nil {
		return err
	}
	updatedCros := compareCrosSV(ctx, newCrosSV, oldSV.GetCros())
	updatedFirmwareSV, err := getGSFirmwareSV(ctx, gsc, outDir, updatedCros)
	if err != nil {
		return err
	}

	// TODO(xixuan): Add more following logics.
	if c.outputPath == "" {
		for _, u := range updatedCros {
			logging.Debugf(ctx, "cros stable version: %v", u)
		}
		for _, u := range updatedFirmwareSV {
			logging.Debugf(ctx, "firmware: %v", u)
		}
		logging.Infof(ctx, "Number of new SV: %d", len(newCrosSV))
		logging.Infof(ctx, "Number of old SV: %d", len(oldSV.GetCros()))
		logging.Infof(ctx, "Number of updated cros SV: %d", len(updatedCros))
		logging.Infof(ctx, "Number of updated firmware SV: %d", len(updatedFirmwareSV))
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
	cros, err := gslib.ParseOmahaStatus(ctx, omahaBytes)
	if err != nil {
		return nil, errors.Annotate(err, "parse omaha").Err()
	}
	return cros, nil
}

func getGitSV(ctx context.Context, gc *gitlib.Client) (*sv.StableVersions, error) {
	res, err := gc.GetFile(ctx, stableVersionConfigPath)
	if err != nil {
		return nil, err
	}
	// TODO(xixuan): make it a subcommand to check the config file's validity and call it in builder.
	if res == "" {
		logging.Warningf(ctx, "empty stable version config file: %s", stableVersionConfigPath)
		return nil, err
	}
	var allSV sv.StableVersions
	if err := utils.Unmarshaller.Unmarshal(strings.NewReader(res), &allSV); err != nil {
		return nil, err
	}
	return &allSV, nil
}

func logInvalidCrosSV(ctx context.Context, crosSV []*sv.StableCrosVersion) {
	for _, csv := range crosSV {
		if err := svlib.ValidateCrOSVersion(csv.GetVersion()); err != nil {
			logging.Debugf(ctx, "invalid cros version: %s, %s", csv.GetKey().GetBuildTarget().GetName(), csv.GetVersion())
		}
	}
}

func compareCrosSV(ctx context.Context, newCrosSV []*sv.StableCrosVersion, oldCrosSV []*sv.StableCrosVersion) []*sv.StableCrosVersion {
	oldMap := make(map[string]string, len(oldCrosSV))
	for _, csv := range oldCrosSV {
		if err := svlib.ValidateCrOSVersion(csv.GetVersion()); err != nil {
			continue
		}
		oldMap[csv.GetKey().GetBuildTarget().GetName()] = csv.GetVersion()
	}
	var updated []*sv.StableCrosVersion
	for _, nsv := range newCrosSV {
		k := nsv.GetKey().GetBuildTarget().GetName()
		v, ok := oldMap[k]
		if ok {
			nv := nsv.GetVersion()
			cp, err := svlib.CompareCrOSVersions(v, nv)
			if err == nil && cp == -1 {
				updated = append(updated, nsv)
			} else {
				logging.Debugf(ctx, "new version %s is not newer than existing version %s for board %s", nv, v, k)
			}
		} else {
			updated = append(updated, nsv)
		}
	}
	return updated
}

func getGSFirmwareSV(ctx context.Context, gsc gslib.Client, outDir string, updatedCros []*sv.StableCrosVersion) ([]*sv.StableFirmwareVersion, error) {
	var res []*sv.StableFirmwareVersion
	for _, newCros := range updatedCros {
		lf := filepath.Join(outDir, localMetaFilePath(newCros))
		remotePath := gslib.MetaFilePath(newCros)
		if err := gsc.Download(remotePath, lf); err != nil {
			logging.Debugf(ctx, "fail to download %s: %s", remotePath, err)
			continue
		}
		bt, err := ioutil.ReadFile(lf)
		if err != nil {
			logging.Debugf(ctx, "fail to load meta file: %s", lf)
			continue
		}
		firmwareSVs, err := gslib.ParseMetadata(bt)
		if err != nil {
			logging.Debugf(ctx, "fail to parse meta file: %s", err)
			continue
		}
		for _, fsv := range firmwareSVs {
			if fsv.GetVersion() != "" {
				res = append(res, fsv)
			}
		}
	}
	return res, nil
}

func localMetaFilePath(crosSV *sv.StableCrosVersion) string {
	return fmt.Sprintf("%s-%s", crosSV.GetKey().GetBuildTarget().GetName(), crosSV.GetVersion())
}
