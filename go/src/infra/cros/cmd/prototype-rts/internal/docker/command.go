package docker

import (
	"bytes"
	"context"
	"io"
	"os/exec"
	"strings"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
)

// VolumeContainerDir is the path inside the container to the Docker volume.
const VolumeContainerDir = "/vol"

// Docker executes docker CLI commands to make and manage a single container.
type Docker struct {
	containerHash string
	volumeName    string
}

func runCommand(ctx context.Context, stdout, stderr io.Writer, args ...string) error {
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	logging.Infof(ctx, "running docker %s", strings.Join(args, " "))
	return cmd.Run()
}

// IsRunning returns whether the Docker container is running.
func (d *Docker) IsRunning() bool {
	return d.containerHash != ""
}

func runAndLog(ctx context.Context, stdout, stderr *bytes.Buffer, args ...string) error {
	err := runCommand(ctx, stdout, stderr, args...)
	logging.Debugf(ctx, "docker stdout:\n%v", stdout)
	logging.Debugf(ctx, "docker stderr:\n%v", stderr)
	return err
}

// createVolume creates a Docker volume.
func (d *Docker) createVolume(ctx context.Context, hostDir string) error {
	var stdoutBuf, stderrBuf bytes.Buffer
	if err := runAndLog(ctx, &stdoutBuf, &stderrBuf, "volume", "create", "--driver", "local", "--opt", "type=none", "--opt", "device="+hostDir, "--opt", "o=bind"); err != nil {
		logging.Errorf(ctx, "error while creating volume:\n%v", stderrBuf)
		return errors.Annotate(err, "docker volume create").Err()
	}
	d.volumeName = strings.TrimSpace(stdoutBuf.String())
	return nil
}

// PullImage downloads a Docker image.
func (d *Docker) PullImage(ctx context.Context, imageURI string) error {
	var stdoutBuf, stderrBuf bytes.Buffer
	if err := runAndLog(ctx, &stdoutBuf, &stderrBuf, "pull", imageURI); err != nil {
		logging.Errorf(ctx, "error in docker pull:\n%v", stderrBuf)
		return errors.Annotate(err, "docker pull").Err()
	}
	logging.Infof(ctx, "successfully pulled docker image")
	return nil
}

// RunContainer starts a Docker container process containing a new Docker volume.
// volumeHostDir is the path in the host filesystem that will be bound as the
// volume in the container, e.g. host:/tmp/dir may be bound as container:/vol
func (d *Docker) RunContainer(ctx context.Context, imageURI, volumeHostDir string) error {
	if d.IsRunning() {
		return errors.Reason("expected no container to be running").Err()
	}
	if err := d.createVolume(ctx, volumeHostDir); err != nil {
		return err
	}
	var stdoutBuf, stderrBuf bytes.Buffer
	if err := runAndLog(ctx, &stdoutBuf, &stderrBuf, "run", "--network", "host", "-t", "--mount", "source="+d.volumeName+",target="+VolumeContainerDir, "-d", imageURI); err != nil {
		logging.Errorf(ctx, "error in docker run:\n%v", stderrBuf)
		return errors.Annotate(err, "docker run").Err()
	}
	d.containerHash = strings.TrimSpace(stdoutBuf.String())
	logging.Infof(ctx, "RTD container started with hash %v", d.containerHash)
	return nil
}

// ExecCommand runs a command inside a running container. The container must
// have already been started with StartContainer().
func (d *Docker) ExecCommand(ctx context.Context, cmd []string) error {
	if !d.IsRunning() {
		return errors.Reason("expected container to be running").Err()
	}
	var stdoutBuf, stderrBuf bytes.Buffer
	args := []string{"exec", d.containerHash}
	args = append(args, cmd...)
	if err := runCommand(ctx, &stdoutBuf, &stderrBuf, args...); err != nil {
		logging.Errorf(ctx, "error in docker exec:\n%v", stderrBuf)
		return errors.Annotate(err, "docker exec").Err()
	}
	// Here for debugging purposes for prototyping.
	logging.Infof(ctx, "docker exec output:\n%s", stdoutBuf.String())
	return nil
}

// StopContainer stops a running container.
func (d *Docker) StopContainer(ctx context.Context) error {
	if !d.IsRunning() {
		return errors.Reason("expected container to be running").Err()
	}
	var stdoutBuf, stderrBuf bytes.Buffer
	if err := runAndLog(ctx, &stdoutBuf, &stderrBuf, "stop", d.containerHash); err != nil {
		logging.Errorf(ctx, "error in docker stop:\n%v", stderrBuf)
		return errors.Annotate(err, "docker stop").Err()
	}
	logging.Infof(ctx, "Stopped the RTD container")
	d.containerHash = ""
	return nil
}
