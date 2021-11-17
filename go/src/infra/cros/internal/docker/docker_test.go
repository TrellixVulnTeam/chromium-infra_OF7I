package docker_test

import (
	"context"
	"errors"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/strslice"
	"go.chromium.org/chromiumos/config/go/build/api"

	"infra/cros/internal/cmd"
	"infra/cros/internal/docker"
)

func TestRunContainer(t *testing.T) {
	ctx := context.Background()

	containerImageInfo := &api.ContainerImageInfo{
		Repository: &api.GcrRepository{
			Hostname: "gcr.io",
			Project:  "testproject",
		},
	}

	containerConfig := &container.Config{
		Cmd:   strslice.StrSlice{"ls", "-l"},
		User:  "testuser",
		Image: "testimage",
	}

	hostConfig := &container.HostConfig{
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: "/tmp/hostdir",
				Target: "/usr/local/containerdir",
			},
			{
				Type:     mount.TypeBind,
				Source:   "/othersource",
				Target:   "/othertarget",
				ReadOnly: true,
			},
		},
		NetworkMode: "host",
	}

	cmdRunner := &cmd.FakeCommandRunnerMulti{
		CommandRunners: []cmd.FakeCommandRunner{
			{
				ExpectedCmd: []string{
					"gcloud", "auth", "activate-service-account",
					"--key-file=/creds/service_accounts/skylab-drone.json",
				},
			},
			{
				ExpectedCmd: []string{
					"gcloud", "auth", "print-access-token",
				},
				Stdout: "abc123",
			},
			{
				ExpectedCmd: []string{
					"docker", "login", "-u", "oauth2accesstoken",
					"-p", "abc123", "gcr.io/testproject",
				},
			},
			{
				ExpectedCmd: []string{
					"docker", "run",
					"--user", "testuser",
					"--network", "host",
					"--mount=source=/tmp/hostdir,target=/usr/local/containerdir,type=bind",
					"--mount=source=/othersource,target=/othertarget,type=bind,readonly",
					"testimage",
					"ls", "-l",
				},
			},
		},
	}

	err := docker.RunContainer(ctx, cmdRunner, containerConfig, hostConfig, containerImageInfo)
	if err != nil {
		t.Fatalf("RunContainer failed: %s", err)
	}
}

func TestRunContainer_CmdError(t *testing.T) {
	ctx := context.Background()

	containerImageInfo := &api.ContainerImageInfo{
		Name: "testimage",
	}

	containerConfig := &container.Config{
		Cmd: strslice.StrSlice{"ls", "-l"},
	}

	hostConfig := &container.HostConfig{
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: "/tmp/hostdir",
				Target: "/usr/local/containerdir",
			},
		},
	}

	cmdRunner := cmd.FakeCommandRunner{}
	cmdRunner.FailCommand = true
	cmdRunner.FailError = errors.New("docker cmd failed.")

	err := docker.RunContainer(ctx, cmdRunner, containerConfig, hostConfig, containerImageInfo)
	if err == nil {
		t.Errorf("RunContainer expected to fail")
	}
}
