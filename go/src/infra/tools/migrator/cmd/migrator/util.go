// Copyright 2020 The LUCI Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"plugin"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"

	"infra/tools/migrator"
	"infra/tools/migrator/plugsupport"
)

func ensureEmptyDirectory(ctx context.Context, path string) error {
	switch fil, err := os.Open(path); {
	case os.IsNotExist(err):
		return errors.Annotate(os.MkdirAll(path, 0777), "creating dir").Err()

	case err == nil:
		switch _, err := fil.Readdirnames(1); err {
		case nil:
			return errors.New("exists but is not empty")
		case io.EOF:
			return nil // exists and is empty
		default:
			return errors.Annotate(err, "reading directory entries").Err()
		}

	default:
		return errors.Annotate(err, "opening %q", path).Err()
	}
}

func withPlugin(ctx context.Context, proj plugsupport.ProjectDir, cb func(migrator.InstantiateAPI) error) (err error) {
	proj.CleanTrash()

	outDir, err := proj.MkTempDir()
	if err != nil {
		return errors.Annotate(err, "creating tempdir for plugin compilation").Err()
	}
	plugFile := filepath.Join(outDir, "plug")

	cmd := exec.CommandContext(
		ctx, "go", "build", "-buildmode=plugin", "-o", plugFile, ".")
	cmd.Dir = proj.PluginDir()

	output := bytes.Buffer{}
	cmd.Stdout = &output
	cmd.Stderr = &output

	logging.Infof(ctx, "Running %q %q", cmd.Path, cmd.Args)
	if err := cmd.Run(); err != nil {
		logging.Errorf(ctx, "Output from building plugin:")
		for scanner := bufio.NewScanner(bufio.NewReader(&output)); scanner.Scan(); {
			logging.Errorf(ctx, "  %s", scanner.Text())
		}
		return errors.Annotate(err, "building plugin").Err()
	}

	plug, err := plugin.Open(plugFile)
	if err != nil {
		return errors.Annotate(err, "loading plugin").Err()
	}

	loadedPlug, err := plugsupport.APIFromPlugin(plug)
	if err != nil {
		return errors.Annotate(err, "parsing plugin").Err()
	}

	defer proj.CleanTrash()

	return cb(loadedPlug)
}
