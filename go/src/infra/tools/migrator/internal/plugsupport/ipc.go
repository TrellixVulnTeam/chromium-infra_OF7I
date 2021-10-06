// Copyright 2021 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package plugsupport

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"os/exec"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"

	"infra/tools/migrator"
)

// Command is passed to the plugin subprocess in a JSON-serialized form.
type Command struct {
	Action        string        // what action to perform, e.g. "scan"
	ProjectDir    ProjectDir    // an absolute path to the project directory
	ContextConfig ContextConfig // "instructions" how to setup the root context
	ScanConfig    ScanConfig    // parameters specific to the "scan" action
}

// Invoke launches the plugin subprocess passing it the given command.
//
// This eventually results in Handle call from within the plugin binary.
//
// Blocks until the plugin subprocess exits and returns its error (if any).
func Invoke(ctx context.Context, projectDir ProjectDir, pluginBin string, command Command) error {
	command.ProjectDir = projectDir
	blob, err := json.Marshal(command)
	if err != nil {
		return errors.Annotate(err, "failed to serialize the command").Err()
	}

	// Pass the command through a temp file to avoid occupying the stdin handle.
	// The plugin may potentially need it.
	tmpFile, err := ioutil.TempFile(projectDir.TrashDir(), "*_cmd.json")
	if err != nil {
		return errors.Annotate(err, "failed to create the temp file for the command").Err()
	}
	_, err = tmpFile.Write(blob)
	if closeErr := tmpFile.Close(); closeErr != nil {
		err = closeErr
	}
	if err != nil {
		return errors.Annotate(err, "failed to write to the command temp file").Err()
	}

	cmd := exec.CommandContext(ctx, pluginBin, tmpFile.Name())
	cmd.Dir = projectDir.PluginDir()
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return errors.Annotate(err, "plugin execution failed").Err()
	}

	return nil
}

// Handle is called from within the plugin process to execute the command.
//
// The return value becomes the process exit code.
func Handle(factory migrator.InstantiateAPI, cmd Command) int {
	ctx := RootContext(context.Background())
	ctx, err := cmd.ContextConfig.Apply(ctx)
	if err != nil {
		logging.Errorf(ctx, "failed to prepare the root context in the plugin: %s\n\n", err)
		return 1
	}

	switch cmd.Action {
	case "scan":
		err = (&scanner{
			factory:    factory,
			projectDir: cmd.ProjectDir,
			cfg:        cmd.ScanConfig,
		}).run(ctx)
	default:
		logging.Errorf(ctx, "unrecognized action %q", cmd.Action)
		return 1
	}

	if err != nil {
		errors.Log(ctx, err)
		return 1
	}

	return 0
}
