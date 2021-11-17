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
	"os"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/pkg/errors"
	"go.chromium.org/chromiumos/config/go/build/api"
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

// dockerLogin generates an access token with 'gcloud auth print-access-token'
// and then runs 'docker login'.
//
// TODO(b/201431966): Remove this when it is not necessary, e.g. when
// 'gcloud auth configure-docker' is run in the environment setup.
func dockerLogin(ctx context.Context, runner cmd.CommandRunner, registry string) error {
	if err := runner.RunCommand(
		ctx, os.Stdout, os.Stderr, "",
		"gcloud", "auth", "activate-service-account",
		"--key-file=/creds/service_accounts/skylab-drone.json",
	); err != nil {
		return errors.Wrap(err, "failed running 'gcloud auth activate-service-account'")
	}

	var stdoutBuf bytes.Buffer

	err := runner.RunCommand(
		ctx, &stdoutBuf, os.Stderr, "",
		"gcloud", "auth", "print-access-token",
	)

	if err != nil {
		return errors.Wrap(err, "failed running 'gcloud auth print-access-token'")
	}

	accessToken := stdoutBuf.String()

	err = runner.RunCommand(
		ctx, os.Stdout, os.Stderr, "",
		"docker", "login", "-u", "oauth2accesstoken",
		"-p", accessToken, registry,
	)

	if err != nil {
		return errors.Wrap(err, "failed running 'docker login'")
	}

	return nil
}

// RunContainer runs a container with `docker run`.
func RunContainer(
	ctx context.Context,
	runner cmd.CommandRunner,
	containerConfig *container.Config,
	hostConfig *container.HostConfig,
	containerImageInfo *api.ContainerImageInfo,
) error {
	if err := dockerLogin(
		ctx, runner,
		fmt.Sprintf(
			"%s/%s",
			containerImageInfo.GetRepository().GetHostname(),
			containerImageInfo.GetRepository().GetProject(),
		),
	); err != nil {
		return err
	}

	args := []string{"run"}

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

	err = runner.RunCommand(ctx, &stdoutBuf, &stderrBuf, "", "docker", args...)

	logging.Infof(ctx, "stdout from Docker command:\n%s", &stdoutBuf)
	logging.Infof(ctx, "stderr from Docker command:\n%s", &stderrBuf)

	return err
}
