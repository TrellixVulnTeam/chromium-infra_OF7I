// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"net"
	"net/url"

	"infra/libs/lro"

	"go.chromium.org/chromiumos/config/go/api/test/tls"
	"go.chromium.org/chromiumos/config/go/longrunning"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

type server struct {
	tls.UnimplementedWiringServer
	lroMgr *lro.Manager
}

func (s server) Serve(l net.Listener) error {
	server := grpc.NewServer()
	// Register reflection service to support grpc_cli usage.
	reflection.Register(server)
	tls.RegisterWiringServer(server, &s)
	s.lroMgr = lro.New()
	longrunning.RegisterOperationsServer(server, s.lroMgr)
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

func (s server) CacheForDut(ctx context.Context, req *tls.CacheForDutRequest) (*longrunning.Operation, error) {
	rawURL := req.GetUrl()
	if rawURL == "" {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("CacheForDut: unsupported url %s in request", rawURL))
	}
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("CacheForDut: unsupported url %s in request", rawURL))
	}
	dutName := req.GetDutName()
	if dutName == "" {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("CacheForDut: unsupported DutName %s in request", dutName))
	}
	op := s.lroMgr.NewOperation()
	go s.cache(ctx, parsedURL, dutName, op.Name)
	return op, status.Error(codes.OK, "Started: CacheForDut Operation.")
}
