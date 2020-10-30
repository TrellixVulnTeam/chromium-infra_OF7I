// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"

	"go.chromium.org/chromiumos/config/go/api/test/tls"
	"go.chromium.org/chromiumos/config/go/api/test/tls/dependencies/longrunning"
	"golang.org/x/crypto/ssh"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"infra/cros/cmd/common-tls/internal/sshpool"
)

type server struct {
	tls.UnimplementedCommonServer
	grpcServ *grpc.Server
	// wiringConn is a connection to the wiring service.
	wiringConn *grpc.ClientConn
	clientPool *sshpool.Pool
	sshConfig  *ssh.ClientConfig
	lroMgr     *lroManager
}

func newServer(c *grpc.ClientConn, sshConfig *ssh.ClientConfig) server {
	return server{
		grpcServ:   grpc.NewServer(),
		wiringConn: c,
		sshConfig:  sshConfig,
	}
}

func (s *server) Serve(l net.Listener) error {
	s.clientPool = sshpool.New(s.sshConfig)
	defer s.clientPool.Close()
	s.lroMgr = newLROManager()
	defer s.lroMgr.Close()

	tls.RegisterCommonServer(s.grpcServ, s)
	longrunning.RegisterOperationsServer(s.grpcServ, s.lroMgr)
	return s.grpcServ.Serve(l)
}

func (s *server) GracefulStop() {
	s.grpcServ.GracefulStop()
}

func (s *server) Provision(ctx context.Context, req *tls.ProvisionRequest) (*longrunning.Operation, error) {
	op := s.lroMgr.new()
	go s.provision(req, op)

	return op, status.Error(codes.OK, "Provisioning started")
}

func (s *server) ExecDutCommand(req *tls.ExecDutCommandRequest, stream tls.Common_ExecDutCommandServer) error {
	// Batch size of stdout, stderr.
	const messageSize = 5000

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

	c, err := s.clientPool.Get(addr)
	if err != nil {
		resp.ExitInfo.ErrorMessage = err.Error()
		_ = stream.Send(resp)
		return status.Errorf(codes.FailedPrecondition, fmt.Sprintf("ExecDutCommand %s %#v: %s", req.GetName(), req.GetCommand(), err))
	}
	defer s.clientPool.Put(addr, c)
	session, err := c.NewSession()
	if err != nil {
		resp.ExitInfo.ErrorMessage = err.Error()
		_ = stream.Send(resp)
		return status.Errorf(codes.FailedPrecondition, fmt.Sprintf("ExecDutCommand %s %#v: %s", req.GetName(), req.GetCommand(), err))
	}

	var wg sync.WaitGroup
	defer wg.Wait()

	// Reading stdout of session and stream to client.
	stdoutReader, stdoutReaderErr := session.StdoutPipe()
	if stdoutReaderErr != nil {
		return status.Errorf(codes.FailedPrecondition, fmt.Sprintf("ExecDutCommand %s %#v: %s", req.GetName(), req.GetCommand(), stdoutReaderErr))
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		stdout := make([]byte, messageSize)
		stdoutResp := &tls.ExecDutCommandResponse{}
		for {
			stdoutN, stdoutReaderErr := stdoutReader.Read(stdout)
			if stdoutN > 0 {
				stdoutResp.Stdout = stdout[:stdoutN]
				_ = stream.Send(stdoutResp)
			}
			if stdoutReaderErr != nil {
				break
			}
		}
	}()

	// Reading stderr of session and stream to client.
	stderrReader, stderrReaderErr := session.StderrPipe()
	if stderrReaderErr != nil {
		return status.Errorf(codes.FailedPrecondition, fmt.Sprintf("ExecDutCommand %s %#v: %s", req.GetName(), req.GetCommand(), stderrReaderErr))
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		stderr := make([]byte, messageSize)
		stderrResp := &tls.ExecDutCommandResponse{}
		for {
			stderrN, stderrReaderErr := stderrReader.Read(stderr)
			if stderrN > 0 {
				stderrResp.Stderr = stderr[:stderrN]
				_ = stream.Send(stderrResp)
			}
			if stderrReaderErr != nil {
				break
			}
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
	case nil:
		resp.ExitInfo.Status = 0
	case *ssh.ExitError:
		resp.ExitInfo.Status = int32(err.Waitmsg.ExitStatus())
		if err.Waitmsg.Signal() != "" {
			resp.ExitInfo.Signaled = true
		}
		resp.ExitInfo.ErrorMessage = err.Error()
	case *ssh.ExitMissingError:
		resp.ExitInfo.ErrorMessage = err.Error()
	default:
		resp.ExitInfo.ErrorMessage = err.Error()
	}

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
