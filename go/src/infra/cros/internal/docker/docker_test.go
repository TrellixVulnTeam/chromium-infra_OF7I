package docker_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/golang/mock/gomock"

	"infra/cros/internal/docker"
	dockertesting "infra/cros/internal/docker/testing"
)

func TestRunContainer(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := dockertesting.NewMockContainerAPIClient(ctrl)
	ctx := context.Background()

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

	m.
		EXPECT().
		ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, "").
		Return(
			container.ContainerCreateCreatedBody{
				ID:       "123",
				Warnings: []string{"Warning: small problem creating container"},
			}, nil,
		)
	m.
		EXPECT().
		ContainerStart(ctx, "123", types.ContainerStartOptions{})

	respC := make(chan container.ContainerWaitOKBody, 1)
	respC <- container.ContainerWaitOKBody{
		StatusCode: 5,
		Error: &container.ContainerWaitOKBodyError{
			Message: "Error in container",
		},
	}

	m.
		EXPECT().
		ContainerWait(ctx, "123", container.WaitConditionNotRunning).
		Return(respC, nil)

	var bytesWriter bytes.Buffer
	stdWriter := stdcopy.NewStdWriter(&bytesWriter, stdcopy.Stdout)

	_, err := stdWriter.Write([]byte("Some test stdout"))
	if err != nil {
		t.Fatalf("Failed to write to stdWriter: %q", err)
	}

	m.
		EXPECT().
		ContainerLogs(ctx, "123", types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true}).
		Return(io.NopCloser(&bytesWriter), nil)

	resp, err := docker.RunContainer(ctx, m, containerConfig, hostConfig)
	if err != nil {
		t.Fatalf("RunContainer failed: %q", err)
	}

	if resp.StatusCode != 5 {
		t.Errorf("Expected StatusCode to be %d, got %d", 5, resp.StatusCode)
	}
}

func TestRunContainer_WaitError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := dockertesting.NewMockContainerAPIClient(ctrl)
	ctx := context.Background()

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

	m.
		EXPECT().
		ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, "").
		Return(container.ContainerCreateCreatedBody{ID: "123"}, nil)
	m.
		EXPECT().
		ContainerStart(ctx, "123", types.ContainerStartOptions{})

	errC := make(chan error)

	writeErrC := func() { errC <- errors.New("ContainerWait had error") }
	go writeErrC()

	m.
		EXPECT().ContainerWait(ctx, "123", container.WaitConditionNotRunning).
		Return(nil, errC)

	_, err := docker.RunContainer(ctx, m, containerConfig, hostConfig)
	if err == nil {
		t.Errorf("RunContainer expected to fail")
	}
}
