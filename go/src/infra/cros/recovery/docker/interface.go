// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package docker

// TODO: Move package to common lib when developing finished.

import (
	"context"
	"time"
)

// For container names please read https://docs.docker.com/engine/reference/run/#name---name

// Client wraps interface to docker communication.
type Client interface {
	Start(ctx context.Context, containerName string, req *ContainerArgs, timeout time.Duration) (*StartResponse, error)
	IsUp(ctx context.Context, containerName string) (bool, error)
	Remove(ctx context.Context, containerName string, force bool) error
	Exec(ctx context.Context, containerName string, req *ExecRequest) (*ExecResponse, error)
	IPAddress(ctx context.Context, containerName string) (string, error)
	CopyTo(ctx context.Context, containerName string, sourcePath, destinationPath string) error
	CopyFrom(ctx context.Context, containerName string, sourcePath, destinationPath string) error
	PrintAll(ctx context.Context) error
}

// ContainerArgs holds info to start container.
type ContainerArgs struct {
	// Run container in detached mode.
	// Detached container are not wait till the process will be finished.
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

// StartResponse holds result data from starting the container.
type StartResponse struct {
	// Exit code of execution.
	// Negative exit codes are reserved for internal use.
	ExitCode int
	Stdout   string
	Stderr   string
}

// ExecRequest holds data to perform request.
type ExecRequest struct {
	Cmd     []string
	Timeout time.Duration
}

// ExecResponse holds result of the execution.
type ExecResponse struct {
	// Exit code of execution.
	// Negative exit codes are reserved for internal use.
	ExitCode int
	Stdout   string
	Stderr   string
}
