// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"net"

	"go.chromium.org/chromiumos/config/go/api/test/tls"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type server struct {
	tls.UnimplementedWiringServer
}

func (s server) Serve(l net.Listener) error {
	server := grpc.NewServer()
	tls.RegisterWiringServer(server, &s)
	return server.Serve(l)
}

func (s server) OpenDutPort(ctx context.Context, req *tls.OpenDutPortRequest) (*tls.OpenDutPortResponse, error) {
	addrs, err := net.LookupHost(req.GetName())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, err.Error())
	}
	if len(addrs) == 0 {
		return nil, status.Errorf(codes.NotFound, "no IP addresses found for DUT")
	}
	return &tls.OpenDutPortResponse{
		Address: addrs[0],
		Port:    req.GetPort(),
	}, nil
}
