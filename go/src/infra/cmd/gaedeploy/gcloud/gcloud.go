// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package gcloud contains helpers for calling `gcloud` tool in PATH.
package gcloud

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
)

// Modules is a map "module name -> its versions".
type Modules map[string]Versions

// Versions is a map from version name to its attributes.
type Versions map[string]VersionAttrs

// VersionAttrs are attributes a version of some module.
type VersionAttrs struct {
	// Empty for now.
}

// List lists deployed versions of an app.
//
// Wraps `gcloud app versions list` command. If `module` is given, limits the
// output only to that module, otherwise lists all modules.
func List(ctx context.Context, appID, module string) (Modules, error) {
	cmdLine := []string{
		"gcloud", "app", "versions", "list",
		"--format", "json",
		"--project", appID,
	}
	if module != "" {
		cmdLine = append(cmdLine, []string{"--service", module}...)
	}

	logging.Infof(ctx, "Running: %v", cmdLine)

	cmd := exec.CommandContext(ctx, cmdLine[0], cmdLine[1:]...)
	cmd.Stderr = os.Stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, errors.Annotate(err, "failed to open stdout pipe").Err()
	}
	if err := cmd.Start(); err != nil {
		return nil, errors.Annotate(err, "gcloud call failed to start").Err()
	}
	defer cmd.Wait() // release resources no matter what

	// Note: this is a subset of fields we care about.
	var listing []struct {
		ID      string `json:"id"`      // version name really
		Service string `json:"service"` // module name
	}
	if err := json.NewDecoder(stdout).Decode(&listing); err != nil {
		return nil, errors.Annotate(err, "bad JSON in gcloud output").Err()
	}
	if err := cmd.Wait(); err != nil {
		return nil, errors.Annotate(err, "gcloud call failed").Err()
	}

	// Convert to our preferred format.
	out := Modules{}
	for _, e := range listing {
		vers := out[e.Service]
		if vers == nil {
			vers = Versions{}
			out[e.Service] = vers
		}
		vers[e.ID] = VersionAttrs{
			// Empty for now.
		}
	}
	return out, nil
}

// Run executes arbitrary `gcloud [cmd]`.
func Run(ctx context.Context, cmd []string, cwd string, dryRun bool) error {
	cmdLine := append([]string{"gcloud"}, cmd...)

	logging.Infof(ctx, "Running: %v", cmdLine)
	logging.Infof(ctx, "  in cwd %q", cwd)

	if dryRun {
		logging.Warningf(ctx, "In dry run mode! Not really running anything.")
		return nil
	}

	cmdObj := exec.CommandContext(ctx, cmdLine[0], cmdLine[1:]...)
	cmdObj.Dir = cwd
	cmdObj.Stdout = os.Stdout
	cmdObj.Stderr = os.Stderr
	if err := cmdObj.Run(); err != nil {
		return errors.Annotate(err, "gcloud call failed").Err()
	}
	return nil
}
