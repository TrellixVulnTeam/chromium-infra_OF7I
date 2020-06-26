// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strings"
	"sync"

	"go.chromium.org/chromiumos/config/go/api/test/tls"
	"golang.org/x/crypto/ssh"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

func (s *server) ExecDutCommand(req *tls.ExecDutCommandRequest, stream tls.Common_ExecDutCommandServer) error {
	ctx := stream.Context()

	resp := &tls.ExecDutCommandResponse{
		ExitInfo: &tls.ExecDutCommandResponse_ExitInfo{
			Started: false,
			Status:  255,
		},
	}

	addr, err := s.getSSHAddr(ctx, req.GetName())

	if err != nil {
		resp.ExitInfo.ErrorMessage = err.Error()
		stream.Send(resp)
		return status.Errorf(codes.FailedPrecondition, err.Error())
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
			resp.ExitInfo.ErrorMessage = err.Error()
			stream.Send(resp)
			return status.Errorf(codes.FailedPrecondition, fmt.Sprintf("ExecDutCommand %s %#v: %s", req.GetName(), req.GetCommand(), err))
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
		resp.ExitInfo.ErrorMessage = err.Error()
		stream.Send(resp)
		return status.Errorf(codes.FailedPrecondition, fmt.Sprintf("ExecDutCommand %s %#v: %s", req.GetName(), req.GetCommand(), err))
	}

	var wg sync.WaitGroup
	defer wg.Wait()

	stdoutReader, err := session.StdoutPipe()
	if err != nil {
		return status.Errorf(codes.FailedPrecondition, fmt.Sprintf("ExecDutCommand %s %#v: %s", req.GetName(), req.GetCommand(), err))
	}
	stdoutScanner := bufio.NewScanner(stdoutReader)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for stdoutScanner.Scan() {
			stdoutResp := &tls.ExecDutCommandResponse{
				ExitInfo: &tls.ExecDutCommandResponse_ExitInfo{
					Started: true,
					Status:  255,
				},
				Stdout: stdoutScanner.Bytes(),
			}
			stream.Send(stdoutResp)
		}
	}()

	// Reading stderr of session and stream to client.
	stderrReader, err := session.StderrPipe()
	if err != nil {
		return status.Errorf(codes.FailedPrecondition, fmt.Sprintf("ExecDutCommand %s %#v: %s", req.GetName(), req.GetCommand(), err))
	}
	stderrScanner := bufio.NewScanner(stderrReader)
	go func() {
		defer wg.Done()
		for stderrScanner.Scan() {
			stderrResp := &tls.ExecDutCommandResponse{
				ExitInfo: &tls.ExecDutCommandResponse_ExitInfo{
					Started: true,
					Status:  255,
				},
				Stderr: stderrScanner.Bytes(),
			}
			stream.Send(stderrResp)
		}
	}()

	defer session.Close()

	args := req.GetArgs()
	if len(args) == 0 {
		err = session.Run(req.GetCommand())
	} else {
		err = session.Run(req.GetCommand() + " " + strings.Join(args, " "))
	}

	resp.ExitInfo.Started = true

	switch err := err.(type) {
	case *ssh.ExitError:
		clientOk = true
		resp.ExitInfo.Status = int32(err.Waitmsg.ExitStatus())
		if err.Waitmsg.Signal() != "" {
			resp.ExitInfo.Signaled = true
		}
		resp.ExitInfo.ErrorMessage = err.Error()
	case *ssh.ExitMissingError:
		clientOk = true
		resp.ExitInfo.ErrorMessage = err.Error()
	default:
		resp.ExitInfo.ErrorMessage = err.Error()
	}

	resp.ExitInfo.Status = 0
	_ = stream.Send(resp)

	return nil
}

// getSSHAddr returns the SSH address to use for the DUT, through the wiring service.
func (s *server) getSSHAddr(ctx context.Context, name string) (string, error) {
	c := tls.NewWiringClient(s.wiringConn)
	resp, err := c.OpenDutPort(ctx, &tls.OpenDutPortRequest{
		Name: name,
		Port: 22,
	})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s:%d", resp.GetAddress(), resp.GetPort()), nil
}
