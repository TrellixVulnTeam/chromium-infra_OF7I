// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"infra/cros/fleet/access"
)

// A sessionContext wraps a session struct with the cancellation context.
// A sessionContext struct is safe to copy.
type sessionContext struct {
	ctx         context.Context
	cancelFunc  context.CancelFunc
	cancelTimer *time.Timer
	session
}

// A session contains the data needed to track a session.
// A session struct is safe to copy.
type session struct {
	listener   net.Listener
	grpcServer *grpc.Server
	tlwServer  *tlwServer
	expire     time.Time
}

// toProto converts the internal session representation to the public
// protobuf representation.
func (s session) toProto(name string) *access.Session {
	return &access.Session{
		Name:       name,
		TlwAddress: s.listener.Addr().String(),
		ExpireTime: timestamppb.New(s.expire),
	}
}

type sessionServer struct {
	access.UnimplementedFleetServer
	wg sync.WaitGroup
	mu sync.Mutex
	// This field is mainly for testing that we can use a fake TLW server.
	newTLWServer func() (*tlwServer, error)
	// sessions intentionally doesn't use pointers to make
	// concurrent ownership easier to reason about.
	sessions map[string]sessionContext
}

// newSessionServer creates a new sessionServer.
// It should be closed after use.
// The gRPC server should be stopped first to ensure there are no new requests.
func newSessionServer() *sessionServer {
	return &sessionServer{
		sessions:     make(map[string]sessionContext),
		newTLWServer: newTLWServer,
	}
}

func (s *sessionServer) registerWith(g *grpc.Server) {
	access.RegisterFleetServer(g, s)
}

func (s *sessionServer) Close() {
	s.mu.Lock()
	for name, sc := range s.sessions {
		sc.cancelFunc()
		delete(s.sessions, name)
	}
	s.mu.Unlock()
	s.wg.Wait()
}

// CreateSession implements access.FleetServer.
func (s *sessionServer) CreateSession(ctx context.Context, req *access.CreateSessionRequest) (*access.Session, error) {
	reqSes := req.GetSession()
	l, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, err.Error())
	}
	gs := grpc.NewServer()
	tlw, err := s.newTLWServer()
	if err != nil {
		return nil, fmt.Errorf("new TLW server: %s", err)
	}

	tlw.registerWith(gs)
	// This goroutine is stopped by the session cancellation that
	// is set up below.
	go gs.Serve(l)
	sc := s.setupSessionContext(session{
		listener:   l,
		grpcServer: gs,
		tlwServer:  tlw,
		expire:     reqSes.GetExpireTime().AsTime(),
	})
	name := "sessions/" + uuid.New().String()
	s.mu.Lock()
	s.sessions[name] = sc
	s.mu.Unlock()
	return sc.toProto(name), nil
}

// setupSessionContext sets up the cancellation context for a new session.
func (s *sessionServer) setupSessionContext(ses session) sessionContext {
	sc := sessionContext{session: ses}
	sc.ctx, sc.cancelFunc = context.WithCancel(context.Background())
	sc.cancelTimer = time.NewTimer(ses.expire.Sub(time.Now()))
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		select {
		case <-sc.ctx.Done():
		case <-sc.cancelTimer.C:
		}
		// Not graceful, the session needs to die now.
		sc.grpcServer.Stop()
		sc.tlwServer.Close()
		// Make sure both the context and timer resources are
		// freed regardless of how we got the cancellation.
		sc.cancelFunc()
		sc.cancelTimer.Stop()
	}()
	return sc
}

// GetSession implements access.FleetServer.
func (s *sessionServer) GetSession(ctx context.Context, req *access.GetSessionRequest) (*access.Session, error) {
	name := req.GetName()
	s.mu.Lock()
	sc, ok := s.sessions[name]
	s.mu.Unlock()
	if !ok {
		return nil, status.Errorf(codes.NotFound, "no session named %s", name)
	}
	return sc.toProto(name), nil
}

// UpdateSession implements access.FleetServer.
func (s *sessionServer) UpdateSession(ctx context.Context, req *access.UpdateSessionRequest) (*access.Session, error) {
	ses := req.GetSession()
	name := ses.GetName()
	s.mu.Lock()
	defer s.mu.Unlock()
	sc, ok := s.sessions[name]
	if !ok {
		return nil, status.Errorf(codes.NotFound, "no session named %s", name)
	}
	var (
		badPaths    []string
		updateTimer bool
	)
	for _, p := range req.GetUpdateMask().GetPaths() {
		switch p {
		case "expire_time":
			sc.expire = ses.GetExpireTime().AsTime()
			// We can't update the timer until we are sure
			// there are no errors, so we set a flag for
			// later.
			updateTimer = true
		default:
			badPaths = append(badPaths)
		}
	}
	if len(badPaths) > 0 {
		return nil, status.Errorf(codes.InvalidArgument, "bad update_mask paths: %v", badPaths)
	}
	if updateTimer {
		if !sc.cancelTimer.Stop() {
			return nil, status.Errorf(codes.Aborted, "session already expired")
		}
		sc.cancelTimer.Reset(sc.expire.Sub(time.Now()))
	}
	s.sessions[name] = sc
	return sc.toProto(name), nil
}

// DeleteSession implements access.FleetServer.
func (s *sessionServer) DeleteSession(ctx context.Context, req *access.DeleteSessionRequest) (*emptypb.Empty, error) {
	name := req.GetName()
	s.mu.Lock()
	sc, ok := s.sessions[name]
	delete(s.sessions, name)
	s.mu.Unlock()
	if !ok {
		return nil, status.Errorf(codes.NotFound, "no session named %s", name)
	}
	sc.cancelFunc()
	return &emptypb.Empty{}, nil
}
