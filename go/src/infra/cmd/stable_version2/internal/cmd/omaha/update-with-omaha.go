// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package omaha

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"

	"infra/cmd/stable_version2/internal/cmd"
	gslib "infra/cmd/stable_version2/internal/gs"
	"infra/cmd/stable_version2/internal/site"
	"infra/cmd/stable_version2/internal/utils"
	gitlib "infra/libs/cros/git"
	svlib "infra/libs/cros/stableversion"

	sv "go.chromium.org/chromiumos/infra/proto/go/lab_platform"

	"infra/cmd/stable_version2/internal/cmd/validateconfig/querygs"
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
		c.Flags.BoolVar(&c.dryRun, "dryrun", false, "indicate if it's a dryrun for stable version update")

		return c
	},
}

type updateWithOmahaRun struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags

	outputPath string
	dryRun     bool
}

// Run implements the subcommands.CommandRun interface.
func (c *updateWithOmahaRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmd.PrintError(a.GetErr(), err)
		return 1
	}
	return 0
}

func (c *updateWithOmahaRun) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	ctx := cli.GetContext(a, c, env)
	ctx = cmd.SetupLogging(ctx)
	f := &c.authFlags

	outDir, err := ioutil.TempDir("", cmd.ProgramName)
	if err != nil {
		return err
	}
	defer func() {
		if err := os.RemoveAll(outDir); err != nil {
			logging.Errorf(ctx, "fail to remove temp dir: %s", err)
		}
	}()

	t, err := cmd.NewAuthenticatedTransport(ctx, f)
	if err != nil {
		return errors.Annotate(err, "create authenticated transport").Err()
	}
	var gsc gslibClient
	var gscImpl gslib.Client
	if err := gscImpl.Init(ctx, t, utils.Unmarshaller); err != nil {
		return err
	}
	gsc = &gscImpl

	// Fetch up-to-date stable version based on omaha file
	newCrosSV, err := getGSCrosSV(ctx, outDir, gsc)
	if err != nil {
		return err
	}

	// Fetch existing stable version
	hc, err := cmd.NewHTTPClient(ctx, f)
	if err != nil {
		return err
	}
	gc, err := gitlib.NewClient(ctx, hc, cmd.GerritHost, cmd.GitilesHost, cmd.Project, cmd.Branch)
	if err != nil {
		return err
	}

	oldSV, err := getGitSV(ctx, gc)
	logInvalidCrosSV(ctx, oldSV.GetCros())
	if err != nil {
		return err
	}

	fvFunc := MakeFirmwareVersionFunc(gsc, outDir)
	newSV, err := FileBuilder(ctx, oldSV, newCrosSV, fvFunc)
	if err != nil {
		return err
	}

	validateErr := validateFile(ctx, a, t, newSV)

	if c.dryRun {
		content, err := svlib.WriteSVToString(newSV)
		if err == nil {
			fmt.Printf("%s\n", content)
		}
		if validateErr != nil {
			return validateErr
		}
		if err != nil {
			return err
		}
		return nil
	}

	if validateErr != nil {
		return validateErr
	}

	changeURL, err := commitNew(ctx, gc, newSV)
	if err != nil {
		return err
	}
	logging.Debugf(ctx, "Update stable version CL: %s", changeURL)
	return nil
}

func validateFile(ctx context.Context, a subcommands.Application, t http.RoundTripper, newSV *sv.StableVersions) error {
	// validate config before committing it
	var r querygs.Reader
	if err := r.Init(ctx, t, utils.Unmarshaller, "validate-config"); err != nil {
		return fmt.Errorf("initializing Google Storage client: %s", err)
	}

	res, err := r.ValidateConfig(ctx, newSV)
	if err != nil {
		return fmt.Errorf("valdating config using Google Storage: %s", err)
	}
	res.RemoveAllowedDUTs()
	msg, err := json.MarshalIndent(res, "", "    ")
	if err != nil {
		panic("failed to marshal JSON")
	}

	if count := res.AnomalyCount(); count > 0 {
		fmt.Fprintf(a.GetErr(), "%s\n", msg)
		return fmt.Errorf("(%d) errors detected: %s", count, msg)
	}

	return nil
}

// Get CrOS stable version from omaha status file.
func getGSCrosSV(ctx context.Context, outDir string, gsc gslibClient) ([]*sv.StableCrosVersion, error) {
	localOSFile := filepath.Join(outDir, cmd.OmahaStatusFile)
	if err := gsc.Download(cmd.OmahaGSPath, localOSFile); err != nil {
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

// getGitSV gets stable versions from the git client.
func getGitSV(ctx context.Context, gc *gitlib.Client) (*sv.StableVersions, error) {
	res, err := gc.GetFile(ctx, cmd.StableVersionConfigPath)
	if err != nil {
		return nil, err
	}
	if res == "" {
		logging.Warningf(ctx, "empty stable version config file: %s", cmd.StableVersionConfigPath)
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

func commitNew(ctx context.Context, gc *gitlib.Client, sv *sv.StableVersions) (string, error) {
	newContent, err := svlib.WriteSVToString(sv)
	if err != nil {
		return "", errors.Annotate(err, "convert change").Err()
	}

	u := map[string]string{
		cmd.StableVersionConfigPath: newContent,
	}
	changeInfo, err := gc.UpdateFiles(ctx, "Update stable version (automatically)", u)
	if err != nil {
		return "", errors.Annotate(err, "update change").Err()
	}
	gerritURL, err := gc.SubmitChange(ctx, changeInfo)
	if err != nil {
		return "", errors.Annotate(err, "submit change").Err()
	}
	return gerritURL, nil
}

func localMetaFilePath(crosSV *sv.StableCrosVersion) string {
	return fmt.Sprintf("%s-%s", crosSV.GetKey().GetBuildTarget().GetName(), crosSV.GetVersion())
}
