// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"bytes"
	"context"
	"fmt"
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

func (s *server) DutShell(req *tls.DutShellRequest, stream tls.Common_DutShellServer) error {
	ctx := stream.Context()
	addr, err := s.getSSHAddr(ctx, req.GetDut())
	if err != nil {
		return fmt.Errorf("DutShell %s %#v: %s", req.GetDut(), req.GetCommand(), err)
	}

	var c *ssh.Client
	clientOk := false
	defer func() {
		if c == nil {
			return
		}
		if clientOk {
			s.clientPool.Put(addr, c)
		} else {
			go c.Close()
		}
	}()

	var session *ssh.Session
	// Retry once, in case we get a bad SSH client out of the pool.
	for i := 0; i < 2; i++ {
		c, err = s.clientPool.Get(addr)
		if err != nil {
			return fmt.Errorf("DutShell %s %#v: %s", req.GetDut(), req.GetCommand(), err)
		}
		session, err = c.NewSession()
		if err != nil {
			// This client is probably bad, so close and stop using it.
			go c.Close()
			continue
		}
		break
	}
	if err != nil {
		return fmt.Errorf("DutShell %s %#v: %s", req.GetDut(), req.GetCommand(), err)
	}
	defer session.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr
	err = session.Run(req.GetCommand())
	// TODO(ayatane): Stream output
	resp := &tls.DutShellResponse{
		Exited: true,
	}
	switch err := err.(type) {
	case *ssh.ExitError:
		clientOk = true
		resp.Status = int32(err.ExitStatus())
	case *ssh.ExitMissingError:
		clientOk = true
		resp.Status = 1
	default:
		resp.Status = 1
		fmt.Fprintf(&stderr, "tls: unknown SSH session error: %s\n", err)
	}
	resp.Stdout = stdout.Bytes()
	resp.Stderr = stderr.Bytes()
	_ = stream.Send(resp)
	return nil
}

// getSSHAddr returns the SSH address to use for the DUT, through the wiring service.
func (s *server) getSSHAddr(ctx context.Context, dut string) (string, error) {
	c := tls.NewWiringClient(s.wiringConn)
	resp, err := c.OpenDutPort(ctx, &tls.OpenDutPortRequest{
		Dut:     dut,
		DutPort: 22,
	})
	if err != nil {
		return "", err
	}
	if s := resp.GetStatus(); s != tls.OpenDutPortResponse_STATUS_OK {
		return "", fmt.Errorf("get SSH addr %s: %s %s", dut, s, resp.GetReason())
	}
	return resp.GetAddress(), nil
}
