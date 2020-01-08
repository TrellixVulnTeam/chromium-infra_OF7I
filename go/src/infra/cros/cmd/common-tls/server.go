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
	conn *grpc.ClientConn
}

func (s server) Serve(l net.Listener) error {
	server := grpc.NewServer()
	tls.RegisterCommonServer(server, s)
	return server.Serve(l)
}

func (s server) DutShell(req *tls.DutShellRequest, stream tls.Common_DutShellServer) error {
	ctx := stream.Context()
	c, err := s.sshToDUT(ctx, req.GetDut())
	if err != nil {
		return fmt.Errorf("DutShell %s %#v: %s", req.GetDut(), req.GetCommand(), err)
	}
	session, err := c.NewSession()
	if err != nil {
		return fmt.Errorf("DutShell %s %#v: %s", req.GetDut(), req.GetCommand(), err)
	}
	defer session.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr
	err = session.Run(req.GetCommand())
	resp := &tls.DutShellResponse{
		Exited: true,
		Stdout: stdout.Bytes(),
		Stderr: stderr.Bytes(),
	}
	if err, ok := err.(*ssh.ExitError); ok {
		resp.Status = int32(err.ExitStatus())
	} else if err != nil {
		resp.Status = 1
	}
	_ = stream.Send(resp)
	return nil
}

func (s server) sshToDUT(ctx context.Context, dut string) (*ssh.Client, error) {
	c := tls.NewWiringClient(s.conn)
	resp, err := c.OpenDutPort(ctx, &tls.OpenDutPortRequest{
		Dut:     dut,
		DutPort: 22,
	})
	if err != nil {
		return nil, fmt.Errorf("sshToDUT %s: %s", dut, err)
	}
	if s := resp.GetStatus(); s != tls.OpenDutPortResponse_STATUS_OK {
		return nil, fmt.Errorf("sshToDUT %s: %s %s", dut, s, resp.GetReason())
	}
	sshC, err := ssh.Dial("tcp", resp.GetAddress(), &ssh.ClientConfig{
		// TODO(ayatane): Add auth here.
		User: "chromeos-test",
		// We don't care about the host key for DUTs.
		// Attackers intercepting our connections to DUTs is not part
		// of our attack profile.
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	})
	if err != nil {
		return nil, fmt.Errorf("sshToDUT %s: %s", dut, err)
	}
	return sshC, nil
}
