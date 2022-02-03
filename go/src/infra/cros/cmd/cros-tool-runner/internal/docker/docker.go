// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package docker provide abstaraction to pull/start/stop/remove docker image.
// Package uses docker-cli from running host.
package docker

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"

	"go.chromium.org/chromiumos/config/go/test/api"
	"go.chromium.org/luci/common/errors"

	"infra/cros/cmd/cros-tool-runner/internal/common"
)

const (
	// Default fallback docket tag.
	DefaultImageTag = "stable"
)

// Docker holds data to perform the docker manipulations.
type Docker struct {
	// Requested docker image, if not exist then use FallbackImageName.
	RequestedImageName string
	// Registry to auth for docker interactions.
	Registry string
	// token to token
	Token string
	// Fall back docker image name. Used if RequestedImageName is empty or image not found.
	FallbackImageName string
	// ServicePort tells which port need to bing bind from docker to the host.
	// Bind is always to the first free port.
	ServicePort int
	// Run container in detach mode.
	Detach bool
	// Name set for assigned to the container.
	Name string
	// ExecCommand tells if we need run special command when we start container.
	ExecCommand []string
	// Attach volumes to the docker image.
	Volumes []string

	// Successful pulled docker image.
	pulledImage string
	// Started container ID.
	containerID string
	// Network used for running container.
	Network string
}

// PullImage pulls docker image.
// The first we try to pull with required tag and if fail then use default tag. Image with default tag always present in repo.
func (d *Docker) PullImage(ctx context.Context) (err error) {
	if d.RequestedImageName != "" {
		d.pulledImage = d.RequestedImageName
		if err = pullImage(ctx, d.pulledImage); err == nil {
			return nil
		}
	}
	if d.FallbackImageName != "" {
		d.pulledImage = d.FallbackImageName
		if err = pullImage(ctx, d.pulledImage); err == nil {
			return nil
		}
	}
	if err != nil {
		return errors.Annotate(err, "pull image").Err()
	}
	return errors.Reason("pull image: failed").Err()
}

// pullImage pulls image by docker-cli.
// docker-cli has to have permission to download images from required repos.
func pullImage(ctx context.Context, image string) error {
	cmd := exec.Command("sudo", "docker", "pull", image)
	if stdout, stderr, err := common.RunWithTimeout(ctx, cmd, 2*time.Minute); err != nil {
		log.Printf("Pull image %q: failed with error %s\nstdout: %s \nstderr %s\n", image, err, stdout, stderr)

		return errors.Annotate(err, "pull image").Err()
	}
	log.Printf("Pull image %q: successful pulled.", image)
	return nil
}

// Auth with docker registry so that pulling and stuff works.
func (d *Docker) Auth(ctx context.Context) (err error) {
	if d.Token == "" {
		log.Printf("No token was provided so skipping docker auth.")
		return nil
	}
	if d.Registry == "" {
		return errors.Reason("docker auth: failed").Err()
	}

	if err = auth(ctx, d.Registry, d.Token); err != nil {
		return errors.Annotate(err, "docker auth").Err()
	}
	return nil
}

// auth authorizes the current process to the given registry, using keys on the drone.
// This will give permissions for pullImage to work :)
func auth(ctx context.Context, registry string, token string) error {
	cmd := exec.Command("sudo", "docker", "login", "-u", "oauth2accesstoken",
		"-p", token, registry)
	stdout, stderr, err := common.RunWithTimeout(ctx, cmd, 1*time.Minute)
	if err != nil {
		return errors.Annotate(err, "failed running 'docker login'").Err()
	}
	log.Printf("Login completed\nstdout: %s \n stderr %s\n", stdout, stderr)
	return nil
}

// Remove removes the containers with matched name.
func (d *Docker) Remove(ctx context.Context) error {
	if d == nil {
		return nil
	}
	// Use force to avoid any un-related issues.
	cmd := exec.Command("sudo", "docker", "rm", "--force", d.Name)
	out, _, err := common.RunWithTimeout(ctx, cmd, time.Minute)
	if err != nil {
		log.Printf("Remote container %q: failed with %s", d.Name, err)
		return errors.Annotate(err, "remove container %q", d.Name).Err()
	}
	log.Printf("Remote container %q: done. Result: %s", d.Name, out)
	return nil
}

// Run docker image.
// The step will create container and start server inside or execution CLI.
func (d *Docker) Run(ctx context.Context) error {
	out, err := d.runDockerImage(ctx)
	if err != nil {
		return errors.Annotate(err, "run docker %q", d.Name).Err()
	}
	if d.Detach {
		d.containerID = strings.TrimSuffix(out, "\n")
		log.Printf("Run docker %q: container Id: %q.", d.Name, d.containerID)
	} else {
		log.Printf("Docker logs %q:\n%s", d.Name, out)
	}
	return nil
}

func (d *Docker) runDockerImage(ctx context.Context) (string, error) {
	args := []string{"run"}
	if d.Detach {
		args = append(args, "-d")
	}
	args = append(args, "--name", d.Name)
	for _, v := range d.Volumes {
		args = append(args, "-v")
		args = append(args, v)
	}
	// Set to automatically remove the container when it exits.
	args = append(args, "--rm")
	args = append(args, "--network", d.Network)
	args = append(args, d.pulledImage)
	if len(d.ExecCommand) > 0 {
		args = append(args, d.ExecCommand...)
	}
	cmd := exec.Command("sudo", append([]string{"docker"}, args...)...)
	so, se, err := common.RunWithTimeout(ctx, cmd, time.Hour)
	log.Printf("Run docker image %q: output: %s", d.Name, so)
	if err != nil {
		return so, errors.Annotate(err, "run docker image %q: %s", d.Name, se).Err()
	}
	return so, nil
}

// CreateImageName creates docker image name from repo-path and tag.
func CreateImageName(repoPath, tag string) string {
	return fmt.Sprintf("%s:%s", repoPath, tag)
}

// CreateImageNameFromInputInfo creates docker image name from input info.
//
// If info is empty then return empty name.
// If one of the fields empty then use related default value.
func CreateImageNameFromInputInfo(di *api.DutInput_DockerImage, defaultRepoPath, defaultTag string) string {
	if di == nil {
		return ""
	}
	if di.GetRepositoryPath() == "" && di.GetTag() == "" {
		return ""
	}
	repoPath := di.GetRepositoryPath()
	if repoPath == "" {
		repoPath = defaultRepoPath
	}
	tag := di.GetTag()
	if tag == "" {
		tag = defaultTag
	}
	if repoPath == "" || tag == "" {
		panic("Default repository path or tag for docker image was not passed.")
	}
	return CreateImageName(repoPath, tag)
}
