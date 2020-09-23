// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package fakecloudtasks provides an in-memory fake for the GCP
// Cloud Tasks API, suitable for use in unit tests.
package fakecloudtasks

import (
	"context"
	"net"

	cloudtasks "cloud.google.com/go/cloudtasks/apiv2"
	"google.golang.org/api/option"
	taskspb "google.golang.org/genproto/googleapis/cloud/tasks/v2"
	"google.golang.org/grpc"
)

// Server implements an in-memory fake implementation of a Cloud Tasks server,
// suitable for unit testing.
type Server struct {
	taskspb.UnimplementedCloudTasksServer
	// CreateTaskResponse is returned by calls to CreateTask.
	CreateTaskResponse *taskspb.Task
	// CreateTaskError is returned by calls to CreateTask
	CreateTaskError error
}

// CreateTask returns the return values set in s.
func (s *Server) CreateTask(context.Context, *taskspb.CreateTaskRequest) (*taskspb.Task, error) {
	return s.CreateTaskResponse, s.CreateTaskError
}

// StartServer listens to a randomly selected port on localhost, registers a grpc sever
// for fakeServer, starts the server and returns the server address and server instance.
// It is up to the caller to shut down the server instance.
func StartServer(ctx context.Context, fakeServer *Server) (string, *grpc.Server, error) {
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return "", nil, err
	}
	gsrv := grpc.NewServer()
	taskspb.RegisterCloudTasksServer(gsrv, fakeServer)
	fakeServerAddr := l.Addr().String()
	go func() {
		if err := gsrv.Serve(l); err != nil {
			panic(err)
		}
	}()

	return fakeServerAddr, gsrv, nil
}

// NewClient returns a cloudtasks.Client instance for a fake server, suitable for
// in-memory unit tests.
func NewClient(ctx context.Context, fakeServerAddr string) (*cloudtasks.Client, error) {
	client, err := cloudtasks.NewClient(ctx,
		option.WithEndpoint(fakeServerAddr),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithInsecure()),
	)

	return client, err
}
