// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package tls provides utilities to manage a Test Library Services server
// running in the background for a phosphorus command.
package tls

import (
	"context"
	"infra/cros/tlslib"
	"log"
	"net"
	"sync"

	"go.chromium.org/luci/common/errors"
	"google.golang.org/grpc"
)

// Server holds local state for a TLS server running in the background.
type Server struct {
	tlwConn *grpc.ClientConn

	tlsServer *tlslib.Server
	addr      net.Addr

	wg    sync.WaitGroup
	bgErr error
}

// StartBackground starts a new TLS server in the background.
//
// On success, caller is responsible for calling Server.Stop() to stop the
// server and free up resources.
func StartBackground(tlwAddress string) (*Server, error) {
	s := &Server{}
	if err := s.start(tlwAddress); err != nil {
		s.Stop()
		return nil, err
	}
	return s, nil
}

func (s *Server) start(tlwAddress string) error {
	conn, err := grpc.Dial(tlwAddress, grpc.WithInsecure())
	if err != nil {
		return errors.Annotate(err, "start background tls").Err()
	}
	s.tlwConn = conn

	el, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		return errors.Annotate(err, "start background tls").Err()
	}
	closeEl := true
	defer func() {
		if closeEl {
			el.Close()
		}
	}()

	s.addr = el.Addr()

	ts, err := tlslib.NewServer(context.TODO(), s.tlwConn)
	if err != nil {
		return errors.Annotate(err, "start background tls").Err()
	}
	s.tlsServer = ts

	s.wg.Add(1)
	// Transfer ownership of `el` to the following goroutine because
	// tslServer.Serve() takes ownership of `el`.
	closeEl = false
	go func() {
		defer s.wg.Done()
		s.bgErr = s.tlsServer.Serve(el)
	}()

	log.Printf("TLS server listening at address %v\n", s.Address())
	return nil
}

// Address returns the address that this server is listening at.
func (s *Server) Address() string {
	if s.addr == nil {
		return ""
	}
	return s.addr.String()
}

// Stop stops the server and frees up resources.
//
// Stop tries to gracefully shutdown the TLS server, which may be a somewhat
// slow operation.
func (s *Server) Stop() {
	log.Printf("Stopping background TLS server.")

	if s.tlsServer != nil {
		s.tlsServer.GracefulStop()
	}
	s.wg.Wait()
	s.tlsServer = nil

	if s.bgErr != nil {
		log.Printf("Error in background TLS server: %s.\n", s.bgErr.Error())
	}

	if s.tlwConn != nil {
		if err := s.tlwConn.Close(); err != nil {
			log.Printf("Failed to close client connection to TLW: %s.\n", err.Error())
		}
		s.tlwConn = nil
	}

	log.Printf("Stopped background TLS server.")
}
