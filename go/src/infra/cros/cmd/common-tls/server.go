// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"net"

	"go.chromium.org/chromiumos/infra/proto/go/tls"
	"golang.org/x/crypto/ssh"
	"google.golang.org/grpc"
)

type server struct {
	tls.UnimplementedCommonServer
	// wiringConn is a connection to the wiring service.
	wiringConn *grpc.ClientConn
	clientPool *sshClientPool
	sshConfig  *ssh.ClientConfig
}

func newServer(c *grpc.ClientConn, sshConfig *ssh.ClientConfig) server {
	return server{
		wiringConn: c,
		sshConfig:  sshConfig,
	}
}

func (s *server) Serve(l net.Listener) error {
	s.clientPool = newSSHClientPool(s.sshConfig)
	defer s.clientPool.Close()

	server := grpc.NewServer()
	tls.RegisterCommonServer(server, s)
	return server.Serve(l)
}

// getSSHAddr returns the SSH address to use for the DUT, through the wiring service.
func (s *server) getSSHAddr(ctx context.Context, dut string) (string, error) {
	c := tls.NewWiringClient(s.wiringConn)
	resp, err := c.OpenDutPort(ctx, &tls.OpenDutPortRequest{
		// TODO(ayatane): Temporarily commenting due to proto change.
		// Dut:     dut,
		// DutPort: 22,
	})
	if err != nil {
		return "", err
	}
	// TODO(ayatane): Temporarily commenting due to proto change.
	// if s := resp.GetStatus(); s != tls.OpenDutPortResponse_STATUS_OK {
	// 	return "", fmt.Errorf("get SSH addr %s: %s %s", dut, s, resp.GetReason())
	// }
	return resp.GetAddress(), nil
}
