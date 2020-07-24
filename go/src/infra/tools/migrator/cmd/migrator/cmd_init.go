// Copyright 2020 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package main

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/maruel/subcommands"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"

	"infra/tools/migrator/plugsupport"
	"infra/tools/migrator/plugsupport/templates"
)

func cmdInit(opts cmdBaseOptions) *subcommands.Command {
	return &subcommands.Command{
		UsageLine: "init path/to/directory",
		ShortDesc: "Initialize a new migration project folder.",
		LongDesc: `Creates a new lucicfg migration project folder.

The directory specified must be empty or already be a lucicfg migration project.
If the directory is an existing project, then nothing happens. Otherwise this
will write the following files:
  * .migration - config file for the migration project
  * _plugin/plugin_main.go - No-op Golang 'plugin' used with the 'scan'
    subcommand. See documentation within this file for more information.

The project directory will be used to contain checkouts and status information
of all repos for which scanner returns an 'affected' response, when running
'migrator scan'.`,

		CommandRun: func() subcommands.CommandRun {
			ret := cmdInitImpl{}
			ret.initFlags(opts)
			return &ret
		},
	}
}

type cmdInitImpl struct {
	cmdBase

	path plugsupport.ProjectDir
}

func (r *cmdInitImpl) positionalRange() (min, max int) { return 0, 1 }

func (r *cmdInitImpl) validateFlags(ctx context.Context, positionals []string, env subcommands.Env) (err error) {
	var projDir string
	if len(positionals) == 1 {
		if projDir, err = filepath.Abs(positionals[0]); err != nil {
			return errors.Annotate(err, "resolving init path %q", positionals[0]).Err()
		}
	} else {
		relpath, err := os.Getwd()
		if err != nil {
			return errors.Annotate(err, "getting cwd").Err()
		}
		if projDir, err = filepath.Abs(relpath); err != nil {
			return errors.Annotate(err, "resolving cwd").Err()
		}
	}

	r.path = plugsupport.ProjectDir(projDir)
	return
}

func (r *cmdInitImpl) execute(ctx context.Context) error {
	if _, err := os.Stat(r.path.ConfigFile()); err == nil {
		logging.Infof(ctx, "Directory is already migration directory.")
		return nil
	}

	if err := ensureEmptyDirectory(ctx, string(r.path)); err != nil {
		return errors.Annotate(err, "ensuring directory %q is empty", r.path).Err()
	}

	if err := os.MkdirAll(r.path.ConfigDir(), 0777); err != nil {
		return errors.Annotate(err, "creating config directory").Err()
	}

	plugDir := r.path.PluginDir()

	if err := os.MkdirAll(plugDir, 0777); err != nil {
		return errors.Annotate(err, "creating scan plugin directory").Err()
	}

	scanProgAssetPrefix := filepath.Base(plugDir) + "/"

	for assetName, data := range templates.Assets() {
		if strings.HasPrefix(assetName, scanProgAssetPrefix) {
			outPath := filepath.Join(plugDir, assetName[len(scanProgAssetPrefix):])
			if err := ioutil.WriteFile(outPath, []byte(data), 0666); err != nil {
				return errors.Annotate(err, "writing %q", outPath).Err()
			}
		}
	}

	return ioutil.WriteFile(
		r.path.ConfigFile(), templates.GetAsset("default.cfg"), 0666)
}

func (r *cmdInitImpl) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	return r.doContextExecute(a, r, args, env)
}
