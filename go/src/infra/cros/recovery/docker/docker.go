// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package docker

// TODO: Move package to common lib when developing finished.

import (
	"bytes"
	"context"
	base_error "errors"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"go.chromium.org/luci/common/errors"
)

// TODO (otabek): Add basic unittest for each method.

const (
	// Connection to docker service can be set by socket or by open tcp connection.
	dockerSocketFilePath = "/var/run/docker.sock"
	dockerTcpPath        = "tcp://192.168.231.1:2375"

	// Enable more debug logs to triage issue.
	// Will be set to false after stabilize work with container.
	enableDebugLogging = false
)

// Proxy wraps a Servo object and forwards connections to the servod instance
// over SSH if needed.
type DockerClient struct {
	// Container name managed by client.
	// More details https://docs.docker.com/engine/reference/run/#name---name
	name   string
	client *client.Client
}

// NewClient creates client to work with docker client.
func NewClient(ctx context.Context, containerName string) (*DockerClient, error) {
	dc := &DockerClient{
		name: containerName,
	}
	var err error
	dc.client, err = createDockerClient(ctx)
	if err != nil {
		log.Printf("New docker client: failed to create docker client: %s", err)
		if dc.client != nil {
			dc.client.Close()
		}
		return nil, err
	}
	return dc, nil
}

// Name returns name of container managed by client.
func (dc *DockerClient) Name() string {
	return dc.name
}

func createDockerClient(ctx context.Context) (*client.Client, error) {
	// Create Docker Client.
	// If the dockerd socket exists, use the default option.
	// Otherwise, try to use the tcp connection local host IP 192.168.231.1:2375
	if _, err := os.Stat(dockerSocketFilePath); err != nil {
		if !base_error.Is(err, os.ErrNotExist) {
			return nil, err
		}
		if enableDebugLogging {
			log.Println("Docker client connecting over TCP")
		}
		// Default HTTPClient inside the Docker Client object fails to
		// connects to docker daemon. Create the transport with DialContext and use
		// this while initializing new docker client object.
		timeout := time.Duration(1 * time.Second)
		transport := &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: timeout,
			}).DialContext,
		}
		c := http.Client{Transport: transport}

		return client.NewClientWithOpts(client.WithHost(dockerTcpPath), client.WithHTTPClient(&c), client.WithAPIVersionNegotiation())
	}
	if enableDebugLogging {
		log.Println("Docker client connecting over docker.sock")
	}
	return client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
}

// StartContainerRequest holds info to start container.
type StartContainerRequest struct {
	Detached     bool
	ImageName    string
	PublishPorts []string
	ExposePorts  []string
	EnvVar       []string
	Volumes      []string
	Network      string
	Privileged   bool
	Exec         []string
}

// StartContainer pull and start container by request.
// More details https://docs.docker.com/engine/reference/run/
func (dc *DockerClient) StartContainer(ctx context.Context, req *StartContainerRequest, timeout time.Duration) (string, string, error) {
	// TODO: migrate to use docker SDK.
	// TODO: move logic to separate method with tests.
	args := []string{"run"}
	if req.Detached {
		args = append(args, "-d")
	}
	args = append(args, "--name", dc.name)
	for _, v := range req.PublishPorts {
		args = append(args, "-p", v)
	}
	if len(req.ExposePorts) > 0 {
		for _, v := range req.ExposePorts {
			args = append(args, "--expose", v)
		}
		args = append(args, "-P")
	}
	for _, v := range req.Volumes {
		args = append(args, "-v", v)
	}
	for _, v := range req.EnvVar {
		args = append(args, "--env", v)
	}
	if req.Privileged {
		args = append(args, "--privileged")
	}
	// Always set to remove container when stop it.
	args = append(args, "--rm")
	if req.Network != "" {
		args = append(args, "--network", req.Network)
	}
	args = append(args, req.ImageName)
	if len(req.Exec) > 0 {
		args = append(args, req.Exec...)
	}
	ec, so, se, err := runWithTimeout(ctx, timeout, "docker", args...)
	if enableDebugLogging {
		log.Printf("Run docker image %q: exitcode: %v\n", dc.name, ec)
		log.Printf("Run docker image %q: stdout: %v\n", dc.name, so)
		log.Printf("Run docker image %q: errout: %v\n", dc.name, se)
		log.Printf("Run docker image %q: err: %v\n", dc.name, err)
	}
	return so, se, errors.Annotate(err, "run docker image %q: %s", dc.name, se).Err()
}

// StopContainer stops running container.
func (d *DockerClient) StopContainer(ctx context.Context) error {
	if enableDebugLogging {
		log.Printf("Stopping container %q\n", d.name)
	}
	err := d.RemoveContainer(ctx, true)
	return errors.Annotate(err, "docker stop container %s", d.name).Err()
}

// RemoveContainer removes existed container.
func (d *DockerClient) RemoveContainer(ctx context.Context, force bool) error {
	if enableDebugLogging {
		log.Printf("Removing container %q, using force:%v", d.name, force)
	}
	args := []string{"rm", d.name}
	if force {
		args = append(args, "--force")
	}
	err := exec.CommandContext(ctx, "docker", args...).Run()
	return errors.Annotate(err, "docker remove container  %s", d.name).Err()
}

type ExecContainerResponse struct {
	ExitCode int
	Stdout   string
	Errout   string
}

// Run executes command on running container.
func (d *DockerClient) ExecContainer(ctx context.Context, timeout time.Duration, cmd ...string) (*ExecContainerResponse, error) {
	if len(cmd) == 0 {
		return &ExecContainerResponse{
			ExitCode: -1,
		}, errors.Reason("exec container: command is not provided").Err()
	}
	// The commands executed is not restricted by logic and it required to run them under sh without TTY.
	args := []string{"exec", "-i", d.name, "sh", "-c"}
	args = append(args, cmd...)
	exitCode, so, eo, err := runWithTimeout(ctx, timeout, "docker", args...)
	if enableDebugLogging {
		log.Printf("Run docker exec %q: stdout: %s", d.name, so)
		log.Printf("Run docker exec %q: errout: %s", d.name, eo)
		log.Printf("Run docker exec %q: err: %s", d.name, err)
	}
	res := &ExecContainerResponse{
		ExitCode: exitCode,
		Stdout:   so,
		Errout:   eo,
	}
	return res, errors.Annotate(err, "run docker image %q: %v", d.name, eo).Err()
}

// Copy copies the file to the container.
func (dc *DockerClient) CopyToContainer(ctx context.Context) error {
	return errors.Reason("Not implemented yet!").Err()
}

// Copy copies the file from the container.
func (dc *DockerClient) CopyFromContainer(ctx context.Context) error {
	return errors.Reason("Not implemented yet!").Err()
}

// PrintAllContainers prints all active containers.
func (dc *DockerClient) PrintAllContainers(ctx context.Context) error {
	containers, err := dc.client.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		return errors.Annotate(err, "docker print all").Err()
	}
	for _, container := range containers {
		log.Printf("docker ps: %s %s\n", container.ID[:10], container.Image)
	}
	return nil
}

// ContainerIsUp checks is container is up.
func (d *DockerClient) ContainerIsUp(ctx context.Context) (bool, error) {
	containers, err := d.client.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		return false, errors.Annotate(err, "container is up: fail to get a list of containers").Err()
	}
	for _, c := range containers {
		for _, n := range c.Names {
			// Remove first chat as names look like `/some_name` where user mostly use 'some_name'.
			if strings.TrimPrefix(n, "/") == d.name {
				return true, nil
			}
		}
	}
	return false, nil
}

// runWithTimeout runs command with timeout limit.
func runWithTimeout(ctx context.Context, timeout time.Duration, command string, args ...string) (exitCode int, stdout string, stderr string, err error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cw := make(chan error, 1)
	var se, so bytes.Buffer
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Stderr = &se
	cmd.Stdout = &so
	defer func() {
		stdout = so.String()
		stderr = se.String()
	}()
	go func() {
		log.Printf("Run cmd: %s", cmd)
		cw <- cmd.Run()
	}()
	select {
	case e := <-cw:
		exitCode = 1
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		}
		err = errors.Annotate(e, "run with timeout %s", timeout).Err()
		return
	case <-ctx.Done():
		err = errors.Reason("run with timeout %s: excited timeout", timeout).Err()
		return
	}
}
