// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package docker provides helper methods for ChromeOS usage of Docker.
package docker

import (
	"bytes"
	"context"
	"fmt"
	"infra/cros/internal/cmd"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"go.chromium.org/luci/common/logging"
)

func generateMountArgs(mounts []mount.Mount) ([]string, error) {
	var args []string

	for _, m := range mounts {
		mountStrParts := []string{
			fmt.Sprintf("source=%s", m.Source),
			fmt.Sprintf("target=%s", m.Target),
		}

		switch m.Type {
		case mount.TypeBind:
			mountStrParts = append(mountStrParts, "type=bind")
		default:
			return nil, fmt.Errorf("mount type %s not supported", m.Type)
		}

		if m.ReadOnly {
			mountStrParts = append(mountStrParts, "readonly")
		}

		args = append(args, fmt.Sprintf("--mount=%s", strings.Join(mountStrParts, ",")))
	}

	return args, nil
}

// RunContainer runs a container with `docker run`.
func RunContainer(
	ctx context.Context,
	runner cmd.CommandRunner,
	containerConfig *container.Config,
	hostConfig *container.HostConfig,
) error {
	args := []string{
		"run",
	}

	if containerConfig.User != "" {
		args = append(args, "--user", containerConfig.User)
	}

	if hostConfig.NetworkMode != "" {
		args = append(args, "--network", string(hostConfig.NetworkMode))
	}

	mountArgs, err := generateMountArgs(hostConfig.Mounts)
	if err != nil {
		return err
	}

	args = append(args, mountArgs...)
	args = append(args, containerConfig.Image)
	args = append(args, containerConfig.Cmd...)

	logging.Infof(ctx, "Running docker cmd: %q", args)

	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer
	return runner.RunCommand(ctx, &stdoutBuf, &stderrBuf, "", "docker", args...)
}
