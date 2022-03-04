// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package gcloud contains helpers for calling `gcloud` tool in PATH.
package gcloud

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"time"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/system/environ"
)

// Modules is a map "module name -> its versions".
type Modules map[string]Versions

// Versions is a map from version name to its attributes.
type Versions map[string]Version

// Version are attributes a version of some module.
type Version struct {
	Name             string    // version name (matches the key in Versions map)
	LastDeployedTime time.Time // when it was deployed
	TrafficSplit     float64   // 0.0 - 1.0
}

// List lists deployed versions of an app.
//
// Wraps `gcloud app versions list` command. If no matching module is found,
// returns nil. If empty `module` is given, returns all modules.
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
	stdout, err := cmd.Output()
	if exitErr, _ := err.(*exec.ExitError); exitErr != nil {
		if bytes.Contains(exitErr.Stderr, []byte("not found")) {
			logging.Warningf(ctx, "%s", exitErr.Stderr)
			return nil, nil
		}
		logging.Errorf(ctx, "%s", exitErr.Stderr)
	}
	if err != nil {
		return nil, errors.Annotate(err, "failed to execute gcloud command").Err()
	}

	// Note: this is a subset of fields we care about.
	var listing []struct {
		ID               string  `json:"id"`            // version name really
		Service          string  `json:"service"`       // module name
		TrafficSplit     float64 `json:"traffic_split"` // 0.0 - 1.0
		LastDeployedTime struct {
			Datetime string `json:"datetime"` // "2020-03-20 17:42:19-07:00"
		} `json:"last_deployed_time"`
	}
	if err := json.Unmarshal(stdout, &listing); err != nil {
		return nil, errors.Annotate(err, "bad JSON in gcloud output").Err()
	}

	// Convert to our preferred format.
	out := Modules{}
	for _, e := range listing {
		vers := out[e.Service]
		if vers == nil {
			vers = Versions{}
			out[e.Service] = vers
		}

		depTime, err := time.Parse("2006-01-02 15:04:05-07:00", e.LastDeployedTime.Datetime)
		if err != nil {
			return nil, errors.Annotate(err, "failed to parse deployment time %q of %s/%s",
				e.LastDeployedTime.Datetime, e.Service, e.ID).Err()
		}

		vers[e.ID] = Version{
			Name:             e.ID,
			TrafficSplit:     e.TrafficSplit,
			LastDeployedTime: depTime,
		}
	}
	return out, nil
}

// Run executes arbitrary `gcloud [cmd]`.
func Run(ctx context.Context, cmd []string, cwd string, env environ.Env, dryRun bool) error {
	cmdLine := append([]string{"gcloud"}, cmd...)

	logging.Infof(ctx, "Running: %v", cmdLine)
	if cwd != "" {
		logging.Infof(ctx, "  in cwd %q", cwd)
	}

	if dryRun {
		logging.Warningf(ctx, "In dry run mode! Not really running anything.")
		return nil
	}

	cmdObj := exec.CommandContext(ctx, cmdLine[0], cmdLine[1:]...)
	cmdObj.Dir = cwd
	cmdObj.Stdout = os.Stdout
	cmdObj.Stderr = os.Stderr
	if env != nil {
		cmdObj.Env = env.Sorted()
	}
	if err := cmdObj.Run(); err != nil {
		return errors.Annotate(err, "gcloud call failed").Err()
	}
	return nil
}
