// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cloudtasks

import (
	"context"
	"net"

	cloudtasks "cloud.google.com/go/cloudtasks/apiv2"
	"google.golang.org/api/option"
	taskspb "google.golang.org/genproto/googleapis/cloud/tasks/v2"
	"google.golang.org/grpc"
)

// FakeServer implements an in-memory fake implementation of a Cloud Tasks server,
// suitable for unit testing.
type FakeServer struct {
	taskspb.UnimplementedCloudTasksServer
	// CreateTaskResponse is returned by calls to CreateTask.
	CreateTaskResponse *taskspb.Task
	// CreateTaskError is returned by calls to CreateTask
	CreateTaskError error
	address         string
}

// CreateTask returns the return values set in s.
func (s *FakeServer) CreateTask(context.Context, *taskspb.CreateTaskRequest) (*taskspb.Task, error) {
	return s.CreateTaskResponse, s.CreateTaskError
}

// Start listens to a randomly selected port on localhost, registers a grpc sever
// for fakeServer, starts the server and returns the server address and server instance.
// It is up to the caller to shut down the server instance.
func (s *FakeServer) Start(ctx context.Context) (*grpc.Server, error) {
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return nil, err
	}
	gsrv := grpc.NewServer()
	taskspb.RegisterCloudTasksServer(gsrv, s)
	s.address = l.Addr().String()
	go func() {
		if err := gsrv.Serve(l); err != nil {
			panic(err)
		}
	}()

	return gsrv, nil
}

// NewClient returns a cloudtasks.Client instance for a fake server, suitable for
// in-memory unit tests.
func (s *FakeServer) NewClient(ctx context.Context) (*cloudtasks.Client, error) {
	client, err := cloudtasks.NewClient(ctx,
		option.WithEndpoint(s.address),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithInsecure()),
	)

	return client, err
}
