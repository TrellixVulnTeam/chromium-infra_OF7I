// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package docker

// TODO: Move package to common lib when developing finished.

import (
	"context"
	base_error "errors"
	"fmt"
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
	// TODO(otabek): Set false after testing in the prod.
	enableDebugLogging       = false
	enablePrintAllContainers = false
)

// Proxy wraps a Servo object and forwards connections to the servod instance
// over SSH if needed.
type dockerClient struct {
	client *client.Client
}

// NewClient creates client to work with docker client.
func NewClient(ctx context.Context) (Client, error) {
	if client, err := createDockerClient(ctx); err != nil {
		log.Printf("New docker client: failed to create docker client: %s", err)
		if client != nil {
			client.Close()
		}
		return nil, errors.Annotate(err, "new docker client").Err()
	} else {
		d := &dockerClient{
			client: client,
		}
		if enablePrintAllContainers {
			d.PrintAll(ctx)
		}
		return d, nil
	}
}

// Create Docker Client.
func createDockerClient(ctx context.Context) (*client.Client, error) {
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

// StartContainer pull and start container by request.
// More details https://docs.docker.com/engine/reference/run/
func (dc *dockerClient) Start(ctx context.Context, containerName string, req *ContainerArgs, timeout time.Duration) (*StartResponse, error) {
	// TODO: migrate to use docker SDK.
	// TODO: move logic to separate method with tests.
	args := []string{"run"}
	if req.Detached {
		args = append(args, "-d")
	}
	args = append(args, "--name", containerName)
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
	res, err := runWithTimeout(ctx, timeout, "docker", args...)
	if enableDebugLogging {
		log.Printf("Run docker exec %q: exitcode: %v", containerName, res.ExitCode)
		log.Printf("Run docker exec %q: stdout: %v", containerName, res.Stdout)
		log.Printf("Run docker exec %q: stderr: %v", containerName, res.Stderr)
		log.Printf("Run docker exec %q: err: %v", containerName, err)
	}
	return &StartResponse{
		ExitCode: res.ExitCode,
		Stdout:   res.Stdout,
		Stderr:   res.Stderr,
	}, errors.Annotate(err, "run docker image %q: %s", containerName, res.Stderr).Err()
}

// Remove removes existed container.
func (d *dockerClient) Remove(ctx context.Context, containerName string, force bool) error {
	if enableDebugLogging {
		log.Printf("Removing container %q, using force:%v", containerName, force)
	}
	args := []string{"rm", containerName}
	if force {
		args = append(args, "--force")
	}
	err := exec.CommandContext(ctx, "docker", args...).Run()
	return errors.Annotate(err, "docker remove container  %s", containerName).Err()
}

// Run executes command on running container.
func (d *dockerClient) Exec(ctx context.Context, containerName string, req *ExecRequest) (*ExecResponse, error) {
	if len(req.Cmd) == 0 {
		return &ExecResponse{
			ExitCode: -1,
		}, errors.Reason("exec container: command is not provided").Err()
	}
	if up, err := d.IsUp(ctx, containerName); err != nil {
		return &ExecResponse{
			ExitCode: -1,
		}, errors.Annotate(err, "exec container").Err()
	} else if !up {
		return &ExecResponse{
			ExitCode: -1,
		}, errors.Reason("exec container: container is down").Err()
	}
	// The commands executed is not restricted by logic and it required to run them under sh without TTY.
	args := []string{"exec", "-i", containerName}
	args = append(args, req.Cmd...)
	res, err := runWithTimeout(ctx, req.Timeout, "docker", args...)
	if enableDebugLogging {
		log.Printf("Run docker exec %q: exitcode: %v", containerName, res.ExitCode)
		log.Printf("Run docker exec %q: stdout: %v", containerName, res.Stdout)
		log.Printf("Run docker exec %q: stderr: %v", containerName, res.Stderr)
		log.Printf("Run docker exec %q: err: %v", containerName, err)
	}
	return &ExecResponse{
		ExitCode: res.ExitCode,
		Stdout:   res.Stdout,
		Stderr:   res.Stderr,
	}, errors.Annotate(err, "run docker image %q: %v", containerName, res.Stderr).Err()
}

// PrintAllContainers prints all active containers.
func (dc *dockerClient) PrintAll(ctx context.Context) error {
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
func (d *dockerClient) IsUp(ctx context.Context, containerName string) (bool, error) {
	containers, err := d.client.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		return false, errors.Annotate(err, "container is up: fail to get a list of containers").Err()
	}
	for _, c := range containers {
		for _, n := range c.Names {
			// Remove first chat as names look like `/some_name` where user mostly use 'some_name'.
			if strings.TrimPrefix(n, "/") == containerName {
				return true, nil
			}
		}
	}
	return false, nil
}

// IPAddress reads assigned Ip address for container.
//
// Execution will use docker CLI:
// $ docker inspect '--format={{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' some_container
// 192.168.27.4
func (d *dockerClient) IPAddress(ctx context.Context, containerName string) (string, error) {
	args := []string{"inspect", "--format={{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}", containerName}
	res, err := runWithTimeout(ctx, time.Minute, "docker", args...)
	if enableDebugLogging {
		log.Printf("Run docker exec %q: exitcode: %v", containerName, res.ExitCode)
		log.Printf("Run docker exec %q: stdout: %v", containerName, res.Stdout)
		log.Printf("Run docker exec %q: stderr: %v", containerName, res.Stderr)
		log.Printf("Run docker exec %q: err: %v", containerName, err)
	}
	if err != nil {
		return "", errors.Annotate(err, "ip address %q", containerName).Err()
	} else if res.ExitCode != 0 {
		return "", errors.Reason("ip address %q: fail with exit code %v", containerName, res.ExitCode).Err()
	}
	addr := strings.TrimSpace(res.Stdout)
	if enableDebugLogging {
		log.Printf("IPAddress %q: %v", containerName, addr)
	}
	return addr, nil
}

// CopyTo copies a file from the host to the container.
func (d *dockerClient) CopyTo(ctx context.Context, containerName string, sourcePath, destinationPath string) error {
	// Using `docker cp -- src desc`  where `--` used to avoid interpret src as argument.
	res, err := runWithTimeout(ctx, time.Minute, "docker", "cp", "--", sourcePath, fmt.Sprintf("%s:%s", containerName, destinationPath))
	if enableDebugLogging {
		log.Printf("Run docker copy to %q: exitcode: %v", containerName, res.ExitCode)
		log.Printf("Run docker copy to %q: stdout: %v", containerName, res.Stdout)
		log.Printf("Run docker copy to %q: stderr: %v", containerName, res.Stderr)
		log.Printf("Run docker copy to %q: err: %v", containerName, err)
	}
	if err != nil {
		return errors.Annotate(err, "copy to %q", containerName).Err()
	} else if res.ExitCode != 0 {
		return errors.Reason("copy to %q: fail with exit code %v", containerName, res.ExitCode).Err()
	}
	return nil
}

// CopyFrom copies a file from container to the host.
func (d *dockerClient) CopyFrom(ctx context.Context, containerName string, sourcePath, destinationPath string) error {
	// Using `docker cp -- src desc`  where `--` used to avoid interpret src as argument.
	res, err := runWithTimeout(ctx, time.Minute, "docker", "cp", "--", fmt.Sprintf("%s:%s", containerName, sourcePath), destinationPath)
	if enableDebugLogging {
		log.Printf("Run docker copy from %q: exitcode: %v", containerName, res.ExitCode)
		log.Printf("Run docker copy from %q: stdout: %v", containerName, res.Stdout)
		log.Printf("Run docker copy from %q: stderr: %v", containerName, res.Stderr)
		log.Printf("Run docker copy from %q: err: %v", containerName, err)
	}
	if err != nil {
		return errors.Annotate(err, "copy from %q", containerName).Err()
	} else if res.ExitCode != 0 {
		return errors.Reason("copy from %q: fail with exit code %v", containerName, res.ExitCode).Err()
	}
	return nil
}
