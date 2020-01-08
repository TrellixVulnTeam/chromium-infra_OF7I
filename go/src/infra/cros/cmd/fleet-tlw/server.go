// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"net"

	"go.chromium.org/chromiumos/infra/proto/go/tls"
	"google.golang.org/grpc"
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
	addrs, err := net.LookupHost(req.GetDut())
	if err != nil {
		return &tls.OpenDutPortResponse{
			Status: tls.OpenDutPortResponse_STATUS_BAD_DUT,
			Reason: err.Error(),
		}, nil
	}
	if len(addrs) == 0 {
		return &tls.OpenDutPortResponse{
			Status: tls.OpenDutPortResponse_STATUS_BAD_DUT,
			Reason: "no IP addresses found for DUT",
		}, nil
	}
	return &tls.OpenDutPortResponse{
		Status:  tls.OpenDutPortResponse_STATUS_OK,
		Address: fmt.Sprintf("%s:%d", addrs[0], req.GetDutPort()),
	}, nil
}
