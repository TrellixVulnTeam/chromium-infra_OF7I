package docker

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"go.chromium.org/luci/common/logging"
)

// VolumeContainerDir is the path inside the container to the Docker volume.
const VolumeContainerDir = "/vol"

// Docker executes docker CLI commands to make and manage a single container.
type Docker struct {
	containerRunning bool
	containerHash    string
	volumeName       string
}

func runCommand(ctx context.Context, stdoutBuf, stderrBuf io.Writer, args ...string) error {
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdout = stdoutBuf
	cmd.Stderr = stderrBuf
	logging.Infof(ctx, "running %s %s", "docker", strings.Join(args, " "))
	return cmd.Run()
}

// IsRunning returns whether the Docker container is running.
func (d *Docker) IsRunning() bool {
	return d.containerRunning
}

// CreateVolume creates a Docker volume.
func (d *Docker) CreateVolume(ctx context.Context, hostDir string) error {
	var stdoutBuf, stderrBuf bytes.Buffer
	if err := runCommand(ctx, &stdoutBuf, &stderrBuf, "volume", "create", "--driver", "local", "--opt", "type=none", "--opt", "device="+hostDir, "--opt", "o=bind"); err != nil {
		return fmt.Errorf("error while creating volume:\n%s", stderrBuf.String())
	}
	d.volumeName = strings.TrimSpace(stdoutBuf.String())
	return nil
}

// Pull downloads a Docker image.
func (d *Docker) Pull(ctx context.Context, imageURI string) error {
	var stdoutBuf, stderrBuf bytes.Buffer
	if err := runCommand(ctx, &stdoutBuf, &stderrBuf, "pull", imageURI); err != nil {
		return fmt.Errorf("error in docker pull:\n%s", stderrBuf.String())
	}
	logging.Infof(ctx, "successfully pulled docker image")
	return nil
}

// Run starts a Docker container process.
func (d *Docker) Run(ctx context.Context, imageURI string) error {
	var stdoutBuf, stderrBuf bytes.Buffer
	if err := runCommand(ctx, &stdoutBuf, &stderrBuf, "run", "--network", "host", "-t", "--mount", "source="+d.volumeName+",target="+VolumeContainerDir, "-d", imageURI); err != nil {
		return fmt.Errorf("error in docker run:\n%v", stderrBuf.String())
	}
	d.containerHash = strings.TrimSpace(stdoutBuf.String())
	logging.Infof(ctx, "RTD container started with hash %v", d.containerHash)
	d.containerRunning = true
	return nil
}

// Exec runs a command inside a running container.
func (d *Docker) Exec(ctx context.Context, cmd []string) error {
	var stdoutBuf, stderrBuf bytes.Buffer
	args := []string{"exec", d.containerHash}
	args = append(args, cmd...)
	if err := runCommand(ctx, &stdoutBuf, &stderrBuf, args...); err != nil {
		return fmt.Errorf("error in docker exec:\n%s", stderrBuf.String())
	}
	// Here for debugging purposes for prototyping.
	logging.Infof(ctx, "docker exec output:\n%s", stdoutBuf.String())
	return nil
}

// Stop stops a running container.
func (d *Docker) Stop(ctx context.Context) error {
	var stdoutBuf, stderrBuf bytes.Buffer
	if err := runCommand(ctx, &stdoutBuf, &stderrBuf, "stop", d.containerHash); err != nil {
		return fmt.Errorf("error in docker stop:\n%v", stderrBuf.String())
	}
	logging.Infof(ctx, "Stopped the RTD container")
	d.containerRunning = false
	d.containerHash = ""
	return nil
}
